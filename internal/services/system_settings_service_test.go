package services_test

import (
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestSystemSettingsService_GetEffectiveLocale(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_settings.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	settingsRepo := sqlite.NewSystemSettingsRepository(db)
	service := services.NewSystemSettingsService(settingsRepo, tempDir)

	supported := []string{"en", "fr"}

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

	// 3. Explicitly set to "en-US" (should match prefix/base "en")
	err = service.SetLanguage("en-US")
	if err != nil {
		t.Fatalf("SetLanguage failed: %v", err)
	}
	effective, err = service.GetEffectiveLocale(supported)
	if err != nil {
		t.Fatalf("GetEffectiveLocale failed: %v", err)
	}
	if effective != "en" {
		t.Errorf("Expected 'en', got %q", effective)
	}

	// 4. Set to unsupported language ("es"), should fallback to "en"
	err = service.SetLanguage("es")
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
