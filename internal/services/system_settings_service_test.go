package services_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

type failingRepo struct{}

func (f *failingRepo) GetSetting(ctx context.Context, key string) (string, error) {
	return "", errors.New("db connection failure")
}

func (f *failingRepo) SetSetting(ctx context.Context, key string, value string) error {
	return errors.New("db connection failure")
}

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

	// 7. Database error propagation test
	failingService := services.NewSystemSettingsService(&failingRepo{}, tempDir)
	_, err = failingService.GetEffectiveLocale(supported)
	if err == nil {
		t.Errorf("Expected error from GetEffectiveLocale when DB read fails, got nil")
	}
}
