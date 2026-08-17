package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/LibreDental/libredental/internal/app"
)

// Window defines the interface required by SystemSettingsService to interact with the application window.
type Window interface {
	IsFullscreen() bool
	IsMaximised() bool
	Size() (width, height int)
	Fullscreen()
	UnFullscreen()
	OnResize(fn func())
	OnClose(fn func())
}

// AppConfig represents local machine settings persisted in config.json.
type AppConfig struct {
	Theme        string `json:"theme"`         // "system", "dark", "light"
	Language     string `json:"language"`      // "system" or BCP 47 tag (e.g. "en")
	WindowMode   string `json:"window_mode"`   // "window", "fullscreen"
	WindowWidth  int    `json:"window_width"`  // default 1280
	WindowHeight int    `json:"window_height"` // default 800
}

// SystemSettingsService handles application and desktop environment preferences and system actions.
type SystemSettingsService struct {
	appDir      string
	mu          sync.RWMutex
	window      Window
	cfg         AppConfig
	dirty       bool
	resizeTimer *time.Timer
}

// NewSystemSettingsService creates a new SystemSettingsService with the app data directory and loads config into memory.
func NewSystemSettingsService(appDir string) *SystemSettingsService {
	s := &SystemSettingsService{appDir: appDir}
	s.cfg = s.loadConfigFromDisk()
	s.dirty = false
	return s
}

// AttachWindow attaches the main window reference and registers dynamic window resize listeners.
// We use a standalone function instead of an exported method to prevent Wails from generating frontend bindings and throwing a warning.
func AttachWindow(s *SystemSettingsService, win Window) {
	s.mu.Lock()
	s.window = win
	s.mu.Unlock()

	if win == nil {
		return
	}

	saveCurrentSize := func(flush bool) {
		defer func() { _ = recover() }()
		if !win.IsFullscreen() && !win.IsMaximised() {
			w, h := win.Size()
			if w >= 400 && h >= 300 {
				_ = s.SaveWindowSize(w, h)
				if flush {
					if err := s.FlushConfig(); err != nil {
						log.Printf("failed to flush config on window close: %v", err)
					}
				}
			}
		}
	}

	defer func() { _ = recover() }()

	win.OnResize(func() {
		saveCurrentSize(false)
	})

	win.OnClose(func() {
		saveCurrentSize(true)
	})
}

func (s *SystemSettingsService) getConfigPath() string {
	return filepath.Join(s.appDir, "config.json")
}

func (s *SystemSettingsService) defaultConfig() AppConfig {
	return AppConfig{
		Theme:        "system",
		Language:     "system",
		WindowMode:   "window",
		WindowWidth:  1280,
		WindowHeight: 800,
	}
}

func (s *SystemSettingsService) loadConfigFromDisk() AppConfig {
	cfg := s.defaultConfig()
	configPath := s.getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil || len(data) == 0 {
		return cfg
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return s.defaultConfig()
	}

	if cfg.Theme == "" {
		cfg.Theme = "system"
	}
	if cfg.Language == "" {
		cfg.Language = "system"
	}
	if cfg.WindowMode == "" {
		cfg.WindowMode = "window"
	}
	if cfg.WindowWidth < 400 {
		cfg.WindowWidth = 1280
	}
	if cfg.WindowHeight < 300 {
		cfg.WindowHeight = 800
	}

	return cfg
}

func (s *SystemSettingsService) persistConfigLocked(cfg AppConfig) error {
	if err := os.MkdirAll(s.appDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := s.getConfigPath()
	tmpPath := configPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temporary config file: %w", err)
	}

	if err := os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}

// FlushConfig forces config persistence to disk immediately.
func (s *SystemSettingsService) FlushConfig() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.resizeTimer != nil {
		s.resizeTimer.Stop()
		s.resizeTimer = nil
	}

	if !s.dirty {
		return nil
	}

	err := s.persistConfigLocked(s.cfg)
	if err == nil {
		s.dirty = false
	}
	return err
}

// GetWindowSize returns saved window width and height.
func (s *SystemSettingsService) GetWindowSize() (int, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.WindowWidth, s.cfg.WindowHeight, nil
}

// SaveWindowSize updates window dimensions in memory and debounces persistence to config.json.
func (s *SystemSettingsService) SaveWindowSize(width, height int) error {
	if width < 400 || height < 300 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg.WindowWidth == width && s.cfg.WindowHeight == height && !s.dirty {
		return nil
	}

	s.cfg.WindowWidth = width
	s.cfg.WindowHeight = height
	s.dirty = true

	if s.resizeTimer != nil {
		s.resizeTimer.Stop()
	}

	s.resizeTimer = time.AfterFunc(500*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		if s.dirty {
			if err := s.persistConfigLocked(s.cfg); err == nil {
				s.dirty = false
			}
		}
	})

	return nil
}

// GetTheme returns the saved application theme preference (defaults to "system").
func (s *SystemSettingsService) GetTheme() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.Theme, nil
}

// SetTheme persists the application theme preference ("dark", "light", or "system") in config.json.
func (s *SystemSettingsService) SetTheme(theme string) error {
	if theme != "light" && theme != "dark" && theme != "system" {
		return fmt.Errorf("invalid theme mode: %s", theme)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.resizeTimer != nil {
		s.resizeTimer.Stop()
		s.resizeTimer = nil
	}

	prospective := s.cfg
	prospective.Theme = theme

	err := s.persistConfigLocked(prospective)
	if err == nil {
		s.cfg = prospective
		s.dirty = false
	}
	return err
}

// GetWindowMode returns the current window mode ("window" or "fullscreen").
func (s *SystemSettingsService) GetWindowMode() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.WindowMode, nil
}

// SetWindowMode persists the window mode and applies it live if a Wails window is attached.
func (s *SystemSettingsService) SetWindowMode(mode string) error {
	if mode != "window" && mode != "fullscreen" {
		return fmt.Errorf("invalid window mode: %s", mode)
	}

	s.mu.RLock()
	win := s.window
	s.mu.RUnlock()

	if win != nil {
		if applyErr := s.applyWindowSettingsToWindow(win, mode); applyErr != nil {
			return applyErr
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.resizeTimer != nil {
		s.resizeTimer.Stop()
		s.resizeTimer = nil
	}

	prospective := s.cfg
	prospective.WindowMode = mode

	err := s.persistConfigLocked(prospective)
	if err == nil {
		s.cfg = prospective
		s.dirty = false
	}
	return err
}

// ApplyWindowSettings applies stored window mode to the attached Wails window.
func (s *SystemSettingsService) ApplyWindowSettings() error {
	s.mu.RLock()
	mode := s.cfg.WindowMode
	win := s.window
	s.mu.RUnlock()

	if win != nil {
		return s.applyWindowSettingsToWindow(win, mode)
	}
	return nil
}

func (s *SystemSettingsService) applyWindowSettingsToWindow(win Window, mode string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic recovered while applying window mode %q: %v", mode, r)
			err = fmt.Errorf("panic applying window mode %q: %v", mode, r)
		}
	}()
	switch mode {
	case "fullscreen":
		win.Fullscreen()
	default:
		win.UnFullscreen()
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

// IsSystemDarkMode queries host operating system desktop settings for dark mode.
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
func (s *SystemSettingsService) GetLanguage() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.Language, nil
}

// SetLanguage persists the UI language preference.
func (s *SystemSettingsService) SetLanguage(lang string) error {
	if lang == "" {
		return errors.New("language tag must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.resizeTimer != nil {
		s.resizeTimer.Stop()
		s.resizeTimer = nil
	}

	prospective := s.cfg
	prospective.Language = lang

	err := s.persistConfigLocked(prospective)
	if err == nil {
		s.cfg = prospective
		s.dirty = false
	}
	return err
}

// GetSystemLocale queries the host operating system for the user's locale.
func (s *SystemSettingsService) GetSystemLocale() (string, error) {
	var tag string

	switch runtime.GOOS {
	case "linux":
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

// IsDesktopMode returns true if the application is running as a desktop app with an attached window.
func (s *SystemSettingsService) IsDesktopMode() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if app.IsServerBuild() {
		return false, nil
	}
	return s.window != nil, nil
}

// GetEffectiveLocale resolves the application's active UI locale.
func (s *SystemSettingsService) GetEffectiveLocale(supportedLocales []string) (string, error) {
	tag, err := s.GetLanguage()
	if err != nil {
		return "", fmt.Errorf("failed to resolve effective locale: %w", err)
	}
	return s.GetEffectiveLocaleFromTag(tag, supportedLocales)
}

// GetEffectiveLocaleFromTag resolves a specific language tag (e.g. from client preference) against supported locales.
func (s *SystemSettingsService) GetEffectiveLocaleFromTag(tag string, supportedLocales []string) (string, error) {
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

	for _, l := range supportedLocales {
		if strings.EqualFold(l, tag) {
			return l, nil
		}
	}

	bestMatch := ""
	for _, l := range supportedLocales {
		if strings.HasPrefix(strings.ToLower(tag), strings.ToLower(l)+"-") && len(l) > len(bestMatch) {
			bestMatch = l
		}
	}
	if bestMatch != "" {
		return bestMatch, nil
	}

	baseTag := strings.ToLower(strings.Split(tag, "-")[0])
	for _, l := range supportedLocales {
		lBase := strings.ToLower(strings.Split(l, "-")[0])
		if lBase == baseTag {
			return l, nil
		}
	}

	for _, l := range supportedLocales {
		if strings.EqualFold(l, "en") {
			return l, nil
		}
	}
	return supportedLocales[0], nil
}

func normalizePOSIXLocale(posix string) string {
	if idx := strings.IndexByte(posix, '.'); idx != -1 {
		posix = posix[:idx]
	}
	if idx := strings.IndexByte(posix, '@'); idx != -1 {
		posix = posix[:idx]
	}
	return strings.ReplaceAll(posix, "_", "-")
}
