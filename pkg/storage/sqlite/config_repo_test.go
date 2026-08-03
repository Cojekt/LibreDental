package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage"
	"github.com/LibreDental/libredental/pkg/storage/sqlite"
)

func TestConfigRepository_SaveAndGet(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_config.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewConfigRepository(db)
	ctx := context.Background()

	// 1. Initial Get should return ErrNotFound before setup
	_, err = repo.Get(ctx)
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound before saving config, got: %v", err)
	}

	// 2. Save PracticeConfig for Canada
	cfg := domain.NewPracticeConfig(domain.CountryCA)
	err = repo.Save(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to save practice config: %v", err)
	}

	// 3. Get PracticeConfig
	fetched, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get practice config: %v", err)
	}

	if fetched.CountryCode != domain.CountryCA {
		t.Errorf("Expected country code 'CA', got '%s'", fetched.CountryCode)
	}
	if fetched.Currency != "CAD" {
		t.Errorf("Expected currency 'CAD', got '%s'", fetched.Currency)
	}
	if fetched.ToothSystem != domain.ToothSystemFDI {
		t.Errorf("Expected tooth system 'fdi', got '%s'", fetched.ToothSystem)
	}

	// 4. Save/Update PracticeConfig to United Kingdom
	updatedCfg := domain.NewPracticeConfig(domain.CountryGB)
	err = repo.Save(ctx, updatedCfg)
	if err != nil {
		t.Fatalf("Failed to update practice config: %v", err)
	}

	fetchedUpdated, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get updated config: %v", err)
	}

	if fetchedUpdated.CountryCode != domain.CountryGB {
		t.Errorf("Expected country code 'GB', got '%s'", fetchedUpdated.CountryCode)
	}
	if fetchedUpdated.Currency != "GBP" {
		t.Errorf("Expected currency 'GBP', got '%s'", fetchedUpdated.Currency)
	}
}
