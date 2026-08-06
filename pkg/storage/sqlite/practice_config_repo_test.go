package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage"
	"github.com/LibreDental/libredental/pkg/storage/sqlite"
)

func TestPracticeConfigRepository_SaveAndGet(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_config.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()

	// 1. Initial Get should return ErrNotFound before setup
	_, err = repo.Get(ctx)
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound before saving config, got: %v", err)
	}

	// 2. Save PracticeConfig for Canada fetched from SQL database
	caMeta, err := repo.GetCountryConfig(ctx, "CA")
	if err != nil {
		t.Fatalf("Failed to fetch CA country config from DB: %v", err)
	}
	cfg := domain.NewPracticeConfig(*caMeta)
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
	gbMeta, err := repo.GetCountryConfig(ctx, "GB")
	if err != nil {
		t.Fatalf("Failed to fetch GB country config from DB: %v", err)
	}
	updatedCfg := domain.NewPracticeConfig(*gbMeta)
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

	// 5. Test ListCountryConfigs & GetDefaultCountryConfig
	configs, err := repo.ListCountryConfigs(ctx)
	if err != nil {
		t.Fatalf("Failed to list country configs: %v", err)
	}
	if len(configs) < 6 {
		t.Errorf("Expected at least 6 country configs in DB, got %d", len(configs))
	}

	defConfig, err := repo.GetDefaultCountryConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to get default country config: %v", err)
	}
	if defConfig.Code != domain.CountryUS {
		t.Errorf("Expected default country code 'US', got '%s'", defConfig.Code)
	}
}

func TestPracticeConfigRepository_ProvidersAndOperatories(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_providers.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()

	// Provider testing
	prov := &domain.Provider{
		ID:            "prov_101",
		Name:          "Dr. Jane Doe",
		Role:          domain.RoleDentist,
		Specialty:     "Endodontics",
		LicenseNumber: "DEN-99281",
		Email:         "jane.doe@example.com",
		Phone:         "555-0199",
		Color:         "#10b981",
		IsActive:      true,
	}

	err = repo.SaveProvider(ctx, prov)
	if err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}

	providers, err := repo.ListProviders(ctx)
	if err != nil {
		t.Fatalf("Failed to list providers: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(providers))
	}
	if providers[0].Name != "Dr. Jane Doe" {
		t.Errorf("Expected provider name 'Dr. Jane Doe', got '%s'", providers[0].Name)
	}

	// Operatory testing
	op := &domain.Operatory{
		ID:       "op_201",
		Name:     "Operatory 1",
		RoomCode: "ROOM-A",
		Type:     domain.OperatoryTypeGeneral,
		IsActive: true,
	}

	err = repo.SaveOperatory(ctx, op)
	if err != nil {
		t.Fatalf("Failed to save operatory: %v", err)
	}

	operatories, err := repo.ListOperatories(ctx)
	if err != nil {
		t.Fatalf("Failed to list operatories: %v", err)
	}
	if len(operatories) != 1 {
		t.Fatalf("Expected 1 operatory, got %d", len(operatories))
	}
	if operatories[0].RoomCode != "ROOM-A" {
		t.Errorf("Expected room code 'ROOM-A', got '%s'", operatories[0].RoomCode)
	}

	// Deletion testing
	err = repo.DeleteProvider(ctx, "prov_101")
	if err != nil {
		t.Fatalf("Failed to delete provider: %v", err)
	}
	providersAfterDelete, _ := repo.ListProviders(ctx)
	if len(providersAfterDelete) != 0 {
		t.Errorf("Expected 0 providers after delete, got %d", len(providersAfterDelete))
	}
}

