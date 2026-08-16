package services_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/app"
	"github.com/LibreDental/libredental/internal/services"
)

func TestSystemSettingsService_JSONStorage(t *testing.T) {
	tempDir := t.TempDir()
	service := services.NewSystemSettingsService(tempDir)

	// Test default values
	theme, err := service.GetTheme()
	if err != nil || theme != "system" {
		t.Errorf("Expected default theme 'system', got %q (err: %v)", theme, err)
	}

	mode, err := service.GetWindowMode()
	if err != nil || mode != "window" {
		t.Errorf("Expected default window mode 'window', got %q (err: %v)", mode, err)
	}

	w, h, err := service.GetWindowSize()
	if err != nil || w != 1280 || h != 800 {
		t.Errorf("Expected default window size 1280x800, got %dx%d (err: %v)", w, h, err)
	}

	// Test setting theme, window mode, and saving dynamic window size
	if err := service.SetTheme("dark"); err != nil {
		t.Fatalf("SetTheme failed: %v", err)
	}
	if err := service.SetWindowMode("fullscreen"); err != nil {
		t.Fatalf("SetWindowMode failed: %v", err)
	}
	if err := service.SaveWindowSize(1600, 900); err != nil {
		t.Fatalf("SaveWindowSize failed: %v", err)
	}
	if err := service.FlushConfig(); err != nil {
		t.Fatalf("FlushConfig failed: %v", err)
	}

	// Verify values persist in JSON
	configPath := filepath.Join(tempDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Expected config.json file to exist at %s", configPath)
	}

	// Create new service instance pointing to same directory
	service2 := services.NewSystemSettingsService(tempDir)
	theme, err = service2.GetTheme()
	if err != nil {
		t.Fatalf("GetTheme failed on re-read: %v", err)
	}
	if theme != "dark" {
		t.Errorf("Expected persisted theme 'dark', got %q", theme)
	}

	mode, err = service2.GetWindowMode()
	if err != nil {
		t.Fatalf("GetWindowMode failed on re-read: %v", err)
	}
	if mode != "fullscreen" {
		t.Errorf("Expected persisted window mode 'fullscreen', got %q", mode)
	}

	w2, h2, err := service2.GetWindowSize()
	if err != nil {
		t.Fatalf("GetWindowSize failed on re-read: %v", err)
	}
	if w2 != 1600 || h2 != 900 {
		t.Errorf("Expected persisted window size 1600x900, got %dx%d", w2, h2)
	}
}

func TestSystemSettingsService_WindowSizeBoundaryRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	service := services.NewSystemSettingsService(tempDir)

	// Save minimum allowed window dimensions (400x300)
	if err := service.SaveWindowSize(400, 300); err != nil {
		t.Fatalf("SaveWindowSize failed: %v", err)
	}
	if err := service.FlushConfig(); err != nil {
		t.Fatalf("FlushConfig failed: %v", err)
	}

	// Re-load config from disk to ensure 400x300 round-trips correctly
	service2 := services.NewSystemSettingsService(tempDir)
	w, h, err := service2.GetWindowSize()
	if err != nil {
		t.Fatalf("GetWindowSize failed on re-read: %v", err)
	}
	if w != 400 || h != 300 {
		t.Errorf("Expected boundary size 400x300 to round-trip, got %dx%d", w, h)
	}
}

func TestSystemSettingsService_GetEffectiveLocale(t *testing.T) {
	tempDir := t.TempDir()
	service := services.NewSystemSettingsService(tempDir)

	supported := []string{"en", "fr", "en-US"}

	// 1. Default when language is unconfigured / "system"
	effective, err := service.GetEffectiveLocale(supported)
	if err != nil {
		t.Fatalf("GetEffectiveLocale failed: %v", err)
	}
	if effective == "" {
		t.Errorf("Expected non-empty effective locale")
	}

	// 2. Explicitly set to supported language ("fr")
	err = service.SetLanguage("fr")
	if err != nil {
		t.Fatalf("SetLanguage failed: %v", err)
	}
	effective, err = service.GetEffectiveLocale(supported)
	if err != nil {
		t.Fatalf("GetEffectiveLocale failed: %v", err)
	}
	if effective != "fr" {
		t.Errorf("Expected 'fr', got %q", effective)
	}

	// 3. Case-insensitive exact match ("FR" -> "fr")
	err = service.SetLanguage("FR")
	if err != nil {
		t.Fatalf("SetLanguage failed: %v", err)
	}
	effective, err = service.GetEffectiveLocale(supported)
	if err != nil {
		t.Fatalf("GetEffectiveLocale failed: %v", err)
	}
	if effective != "fr" {
		t.Errorf("Expected 'fr' via case-insensitive match, got %q", effective)
	}

	// 4. Subtag boundary & longest matching parent ("en-US-tx" -> "en-US" instead of "en")
	err = service.SetLanguage("en-US-tx")
	if err != nil {
		t.Fatalf("SetLanguage failed: %v", err)
	}
	effective, err = service.GetEffectiveLocale(supported)
	if err != nil {
		t.Fatalf("GetEffectiveLocale failed: %v", err)
	}
	if effective != "en-US" {
		t.Errorf("Expected longest parent match 'en-US', got %q", effective)
	}

	// 5. Raw prefix boundary test ("esperanto" must NOT match "es")
	err = service.SetLanguage("esperanto")
	if err != nil {
		t.Fatalf("SetLanguage failed: %v", err)
	}
	effective, err = service.GetEffectiveLocale([]string{"es", "en"})
	if err != nil {
		t.Fatalf("GetEffectiveLocale failed: %v", err)
	}
	if effective != "en" {
		t.Errorf("Expected fallback 'en' for 'esperanto' against ['es', 'en'], got %q", effective)
	}

	// 6. Set to unsupported language ("de"), should fallback to "en"
	err = service.SetLanguage("de")
	if err != nil {
		t.Fatalf("SetLanguage failed: %v", err)
	}
	effective, err = service.GetEffectiveLocale(supported)
	if err != nil {
		t.Fatalf("GetEffectiveLocale failed: %v", err)
	}
	if effective != "en" {
		t.Errorf("Expected fallback 'en', got %q", effective)
	}
}

func TestSystemSettingsService_RemoveWindowedFullscreen(t *testing.T) {
	tempDir := t.TempDir()
	service := services.NewSystemSettingsService(tempDir)

	if err := service.SetWindowMode("windowed_fullscreen"); err == nil {
		t.Errorf("Expected SetWindowMode('windowed_fullscreen') to fail, but it succeeded")
	}

	if err := service.SetWindowMode("fullscreen"); err != nil {
		t.Errorf("Expected SetWindowMode('fullscreen') to succeed, got %v", err)
	}

	if err := service.SetWindowMode("window"); err != nil {
		t.Errorf("Expected SetWindowMode('window') to succeed, got %v", err)
	}
}

func TestSystemSettingsService_DebounceAndFlush(t *testing.T) {
	tempDir := t.TempDir()
	service := services.NewSystemSettingsService(tempDir)

	// Save window size updates in-memory immediately
	if err := service.SaveWindowSize(1400, 900); err != nil {
		t.Fatalf("SaveWindowSize failed: %v", err)
	}
	w, h, err := service.GetWindowSize()
	if err != nil || w != 1400 || h != 900 {
		t.Fatalf("Expected immediate in-memory size 1400x900, got %dx%d (err: %v)", w, h, err)
	}

	// Flush forces persistence immediately
	if err := service.FlushConfig(); err != nil {
		t.Fatalf("FlushConfig failed: %v", err)
	}

	service2 := services.NewSystemSettingsService(tempDir)
	w2, h2, _ := service2.GetWindowSize()
	if w2 != 1400 || h2 != 900 {
		t.Fatalf("Expected persisted size 1400x900 after flush, got %dx%d", w2, h2)
	}
}

func TestSystemSettingsService_WriteFailureRollback(t *testing.T) {
	// Provide a path where creating directory or writing file fails (file used as directory path)
	filePath := filepath.Join(t.TempDir(), "not_a_dir")
	if err := os.WriteFile(filePath, []byte("block"), 0o644); err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	service := services.NewSystemSettingsService(filePath)
	origTheme, _ := service.GetTheme()
	if origTheme != "system" {
		t.Fatalf("Expected default theme 'system', got %q", origTheme)
	}

	// SetTheme should fail and NOT alter in-memory theme
	err := service.SetTheme("dark")
	if err == nil {
		t.Fatalf("Expected SetTheme to fail due to invalid appDir path")
	}

	themeAfterFail, err := service.GetTheme()
	if err != nil {
		t.Fatalf("GetTheme failed after failed write: %v", err)
	}
	if themeAfterFail != "system" {
		t.Errorf("Expected in-memory theme to remain 'system' after failed write, got %q", themeAfterFail)
	}
}

type mockWindow struct {
	fullscreen bool
	maximised  bool
	width      int
	height     int
	onResize   func()
	onClose    func()
}

func (m *mockWindow) IsFullscreen() bool { return m.fullscreen }
func (m *mockWindow) IsMaximised() bool  { return m.maximised }
func (m *mockWindow) Size() (int, int)   { return m.width, m.height }
func (m *mockWindow) Fullscreen()        { m.fullscreen = true }
func (m *mockWindow) UnFullscreen()      { m.fullscreen = false }
func (m *mockWindow) OnResize(fn func()) { m.onResize = fn }
func (m *mockWindow) OnClose(fn func())  { m.onClose = fn }

func TestSystemSettingsService_ApplyWindowSettings(t *testing.T) {
	tempDir := t.TempDir()
	service := services.NewSystemSettingsService(tempDir)

	win := &mockWindow{width: 1280, height: 800}
	services.AttachWindow(service, win)

	if err := service.SetWindowMode("fullscreen"); err != nil {
		t.Fatalf("SetWindowMode('fullscreen') failed: %v", err)
	}

	mode, err := service.GetWindowMode()
	if err != nil || mode != "fullscreen" {
		t.Fatalf("Expected mode 'fullscreen', got %q (err: %v)", mode, err)
	}

	if err := service.ApplyWindowSettings(); err != nil {
		t.Fatalf("ApplyWindowSettings failed: %v", err)
	}

	if !win.fullscreen {
		t.Errorf("Expected window to be fullscreen")
	}
}

func TestSystemSettingsService_IsDesktopMode(t *testing.T) {
	tempDir := t.TempDir()
	service := services.NewSystemSettingsService(tempDir)

	isDesktop, err := service.IsDesktopMode()
	if err != nil {
		t.Fatalf("IsDesktopMode failed: %v", err)
	}
	if isDesktop {
		t.Errorf("Expected IsDesktopMode to be false before window attachment")
	}

	win := &mockWindow{width: 1280, height: 800}
	services.AttachWindow(service, win)

	isDesktop, err = service.IsDesktopMode()
	if err != nil {
		t.Fatalf("IsDesktopMode failed after attach: %v", err)
	}
	expected := !app.IsServerBuild()
	if isDesktop != expected {
		t.Errorf("Expected IsDesktopMode to be %v after window attachment (server tag build: %v)", expected, app.IsServerBuild())
	}
}

func TestSystemSettingsService_GetEffectiveLocaleFromTag(t *testing.T) {
	tempDir := t.TempDir()
	service := services.NewSystemSettingsService(tempDir)
	supported := []string{"en", "fr", "es"}

	eff, err := service.GetEffectiveLocaleFromTag("fr", supported)
	if err != nil || eff != "fr" {
		t.Errorf("Expected 'fr', got %q (err: %v)", eff, err)
	}

	eff, err = service.GetEffectiveLocaleFromTag("es-MX", supported)
	if err != nil || eff != "es" {
		t.Errorf("Expected 'es' for 'es-MX', got %q (err: %v)", eff, err)
	}
}
