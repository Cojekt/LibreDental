package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/LibreDental/libredental/internal/storage"
)

// SystemSettingsService handles application and desktop environment preferences and system actions.
type SystemSettingsService struct {
	repo   storage.SystemSettingsRepository
	appDir string
}

// NewSystemSettingsService creates a new SystemSettingsService with repo and app data directory.
func NewSystemSettingsService(repo storage.SystemSettingsRepository, appDir string) *SystemSettingsService {
	return &SystemSettingsService{repo: repo, appDir: appDir}
}

// GetTheme returns the saved application theme preference (defaults to "system").
func (s *SystemSettingsService) GetTheme() (string, error) {
	val, err := s.repo.GetSetting(context.Background(), "theme")
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return "system", nil
		}
		return "system", fmt.Errorf("failed to fetch theme setting: %w", err)
	}
	if val == "" {
		return "system", nil
	}
	return val, nil
}

// SetTheme persists the application theme preference ("dark", "light", or "system") in SQLite.
func (s *SystemSettingsService) SetTheme(theme string) error {
	if theme != "light" && theme != "dark" && theme != "system" {
		return fmt.Errorf("invalid theme mode: %s", theme)
	}
	err := s.repo.SetSetting(context.Background(), "theme", theme)
	if err != nil {
		return fmt.Errorf("failed to save theme setting: %w", err)
	}
	return nil
}

// GetSetting fetches any custom system setting by key.
func (s *SystemSettingsService) GetSetting(key string) (string, error) {
	val, err := s.repo.GetSetting(context.Background(), key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("failed to fetch setting (%s): %w", key, err)
	}
	return val, nil
}

// SetSetting saves any custom system setting by key.
func (s *SystemSettingsService) SetSetting(key string, value string) error {
	err := s.repo.SetSetting(context.Background(), key, value)
	if err != nil {
		return fmt.Errorf("failed to save setting (%s): %w", key, err)
	}
	return nil
}

// GetDataDir returns the absolute path to the local application data / save directory.
func (s *SystemSettingsService) GetDataDir() (string, error) {
	return s.appDir, nil
}

// OpenDataDir opens the local application data / save directory in the native OS file manager.
func (s *SystemSettingsService) OpenDataDir() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", s.appDir)
	case "darwin":
		cmd = exec.Command("open", s.appDir)
	default: // linux / bsd
		cmd = exec.Command("xdg-open", s.appDir)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open storage folder (%s): %w", s.appDir, err)
	}
	return nil
}

// IsSystemDarkMode queries the host operating system desktop settings (Linux GTK/GNOME, Windows Registry, macOS defaults) for dark mode.
func (s *SystemSettingsService) IsSystemDarkMode() (bool, error) {
	switch runtime.GOOS {
	case "linux":
		out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output()
		if err == nil && (len(out) > 0) && (string(out) == "'prefer-dark'\n" || string(out) == "'prefer-dark'" || string(out) == "prefer-dark") {
			return true, nil
		}
		outTheme, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme").Output()
		if err == nil && (len(outTheme) > 0) {
			str := string(outTheme)
			if (len(str) >= 4) && (str[len(str)-5:] == "dark'" || str[len(str)-6:] == "dark'\n" || str[len(str)-4:] == "dark") {
				return true, nil
			}
			// Case-insensitive check for dark in gtk-theme name (e.g. Mint-Y-Dark)
			for i := 0; i <= len(str)-4; i++ {
				if (str[i] == 'd' || str[i] == 'D') &&
					(str[i+1] == 'a' || str[i+1] == 'A') &&
					(str[i+2] == 'r' || str[i+2] == 'R') &&
					(str[i+3] == 'k' || str[i+3] == 'K') {
					return true, nil
				}
			}
		}
	case "darwin":
		out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
		if err == nil && len(out) >= 4 {
			return true, nil
		}
	case "windows":
		out, err := exec.Command("reg", "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "/v", "AppsUseLightTheme").Output()
		if err == nil && len(out) > 0 {
			str := string(out)
			for i := 0; i <= len(str)-3; i++ {
				if str[i] == '0' && str[i+1] == 'x' && str[i+2] == '0' {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// GetLanguage returns the saved UI language preference.
// Returns "system" if none has been set, indicating the OS locale should be used.
func (s *SystemSettingsService) GetLanguage() (string, error) {
	val, err := s.repo.GetSetting(context.Background(), "language")
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return "system", nil
		}
		return "system", fmt.Errorf("failed to fetch language setting: %w", err)
	}
	if val == "" {
		return "system", nil
	}
	return val, nil
}

// SetLanguage persists the UI language preference.
// Accepts a BCP 47 language tag (e.g. "en", "fr", "de") or the special value "system".
func (s *SystemSettingsService) SetLanguage(lang string) error {
	if lang == "" {
		return fmt.Errorf("language tag must not be empty")
	}
	err := s.repo.SetSetting(context.Background(), "language", lang)
	if err != nil {
		return fmt.Errorf("failed to save language setting: %w", err)
	}
	return nil
}

// GetSystemLocale queries the host operating system for the user's locale and returns
// a BCP 47 language tag (e.g. "en", "en-US", "fr-FR"). Falls back to "en" on any error.
func (s *SystemSettingsService) GetSystemLocale() (string, error) {
	var tag string

	switch runtime.GOOS {
	case "linux":
		// Prefer LANG env var (e.g. "en_US.UTF-8" → "en-US")
		for _, key := range []string{"LANGUAGE", "LANG", "LC_ALL", "LC_MESSAGES"} {
			if v := os.Getenv(key); v != "" && v != "C" && v != "POSIX" {
				tag = normalizePOSIXLocale(v)
				break
			}
		}
	case "darwin":
		out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
		if err == nil {
			tag = normalizePOSIXLocale(strings.TrimSpace(string(out)))
		}
		if tag == "" {
			out2, err2 := exec.Command("defaults", "read", "-g", "AppleLanguages").Output()
			if err2 == nil {
				// Output is like: ( "en-US", "fr-FR" )
				raw := strings.TrimSpace(string(out2))
				raw = strings.Trim(raw, "()\n")
				parts := strings.Split(raw, ",")
				if len(parts) > 0 {
					tag = strings.Trim(strings.TrimSpace(parts[0]), "\"")
				}
			}
		}
	case "windows":
		out, err := exec.Command("powershell", "-NoProfile", "-Command",
			"[System.Globalization.CultureInfo]::CurrentUICulture.Name").Output()
		if err == nil {
			tag = strings.TrimSpace(string(out))
		}
	}

	if tag == "" {
		return "en", nil
	}
	return tag, nil
}

// GetEffectiveLocale resolves the application's active UI locale by inspecting the saved setting
// (falling back to host OS locale if set to "system" or empty) and matching it against supportedLocales.
func (s *SystemSettingsService) GetEffectiveLocale(supportedLocales []string) (string, error) {
	tag, err := s.GetLanguage()
	if err != nil {
		return "", fmt.Errorf("failed to resolve effective locale: %w", err)
	}
	if tag == "system" || tag == "" {
		sysTag, sysErr := s.GetSystemLocale()
		if sysErr == nil && sysTag != "" {
			tag = sysTag
		} else {
			tag = "en"
		}
	}

	if len(supportedLocales) == 0 {
		return tag, nil
	}

	// 1. Case-insensitive exact match
	for _, l := range supportedLocales {
		if strings.EqualFold(l, tag) {
			return l, nil
		}
	}

	// 2. Match the most specific parent locale on a subtag boundary (e.g. tag "en-US-tx" matches "en-US" over "en")
	bestMatch := ""
	for _, l := range supportedLocales {
		if strings.HasPrefix(strings.ToLower(tag), strings.ToLower(l)+"-") && len(l) > len(bestMatch) {
			bestMatch = l
		}
	}
	if bestMatch != "" {
		return bestMatch, nil
	}

	// 3. Case-insensitive base tag match (e.g. tag "en-US" -> base "en" matches supported "en")
	baseTag := strings.ToLower(strings.Split(tag, "-")[0])
	for _, l := range supportedLocales {
		lBase := strings.ToLower(strings.Split(l, "-")[0])
		if lBase == baseTag {
			return l, nil
		}
	}

	// Default fallback to "en" if available, else first supported locale
	for _, l := range supportedLocales {
		if strings.EqualFold(l, "en") {
			return l, nil
		}
	}
	return supportedLocales[0], nil
}

// normalizePOSIXLocale converts a POSIX locale string (e.g. "en_US.UTF-8") to a BCP 47 tag (e.g. "en-US").
func normalizePOSIXLocale(posix string) string {
	// Strip encoding suffix (e.g. ".UTF-8")
	if idx := strings.IndexByte(posix, '.'); idx != -1 {
		posix = posix[:idx]
	}
	// Strip modifier (e.g. "@euro")
	if idx := strings.IndexByte(posix, '@'); idx != -1 {
		posix = posix[:idx]
	}
	// Convert underscore to hyphen: "en_US" → "en-US"
	return strings.ReplaceAll(posix, "_", "-")
}
