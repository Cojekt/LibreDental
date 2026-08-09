package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestSystemSettingsRepository_SetAndGet(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_settings.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewSystemSettingsRepository(db)
	ctx := context.Background()

	// 1. Unset key should return ErrNotFound
	_, err = repo.GetSetting(ctx, "theme")
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound for unset setting, got: %v", err)
	}

	// 2. Set theme to dark
	err = repo.SetSetting(ctx, "theme", "dark")
	if err != nil {
		t.Fatalf("Failed to set theme: %v", err)
	}

	val, err := repo.GetSetting(ctx, "theme")
	if err != nil {
		t.Fatalf("Failed to get theme: %v", err)
	}
	if val != "dark" {
		t.Errorf("Expected 'dark', got '%s'", val)
	}

	// 3. Update theme to light
	err = repo.SetSetting(ctx, "theme", "light")
	if err != nil {
		t.Fatalf("Failed to update theme: %v", err)
	}

	valUpdated, err := repo.GetSetting(ctx, "theme")
	if err != nil {
		t.Fatalf("Failed to get updated theme: %v", err)
	}
	if valUpdated != "light" {
		t.Errorf("Expected 'light', got '%s'", valUpdated)
	}
}
