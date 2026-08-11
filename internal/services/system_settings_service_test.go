package services_test

import (
	"os"
	"path/filepath"
	"testing"

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
	theme, _ = service2.GetTheme()
	if theme != "dark" {
		t.Errorf("Expected persisted theme 'dark', got %q", theme)
	}

	mode, _ = service2.GetWindowMode()
	if mode != "fullscreen" {
		t.Errorf("Expected persisted window mode 'fullscreen', got %q", mode)
	}

	w2, h2, _ := service2.GetWindowSize()
	if w2 != 1600 || h2 != 900 {
		t.Errorf("Expected persisted window size 1600x900, got %dx%d", w2, h2)
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
