package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
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

	// Deletion testing: deactivating the sole active provider must be rejected.
	if err := repo.DeleteProvider(ctx, "prov_101"); !errors.Is(err, storage.ErrLastActiveProvider) {
		t.Fatalf("Expected ErrLastActiveProvider deleting the last active provider, got: %v", err)
	}

	// Add a second active provider so the first can be deactivated.
	prov2 := &domain.Provider{
		ID:       "prov_102",
		Name:     "Dr. John Wick",
		Role:     domain.RoleDentist,
		IsActive: true,
	}
	if err := repo.SaveProvider(ctx, prov2); err != nil {
		t.Fatalf("Failed to save second provider: %v", err)
	}

	err = repo.DeleteProvider(ctx, "prov_101")
	if err != nil {
		t.Fatalf("Failed to delete provider: %v", err)
	}
	providersAfterDelete, _ := repo.ListProviders(ctx)
	for _, p := range providersAfterDelete {
		if p.ID == "prov_101" && p.IsActive {
			t.Errorf("Expected prov_101 to be inactive after delete")
		}
	}

	// Deactivating the now-sole remaining active provider must be rejected too.
	if err := repo.DeleteProvider(ctx, "prov_102"); !errors.Is(err, storage.ErrLastActiveProvider) {
		t.Fatalf("Expected ErrLastActiveProvider deleting the last remaining active provider, got: %v", err)
	}
}

// TestPracticeConfigRepository_DeleteProvider_ConcurrentRace verifies that concurrent
// DeleteProvider calls on different providers can never both succeed when only two
// active providers remain, which would otherwise leave zero active providers.
func TestPracticeConfigRepository_DeleteProvider_ConcurrentRace(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_providers_race.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()

	ids := []string{"prov_a", "prov_b"}
	for _, id := range ids {
		if err := repo.SaveProvider(ctx, &domain.Provider{ID: id, Name: id, Role: domain.RoleDentist, IsActive: true}); err != nil {
			t.Fatalf("Failed to save provider %s: %v", id, err)
		}
	}

	var wg sync.WaitGroup
	errs := make([]error, len(ids))
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			errs[i] = repo.DeleteProvider(ctx, id)
		}(i, id)
	}
	wg.Wait()

	successCount := 0
	for _, err := range errs {
		if err == nil {
			successCount++
		} else if !errors.Is(err, storage.ErrLastActiveProvider) {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
	if successCount != 1 {
		t.Fatalf("Expected exactly 1 of 2 concurrent deletes to succeed, got %d", successCount)
	}

	providers, err := repo.ListProviders(ctx)
	if err != nil {
		t.Fatalf("Failed to list providers: %v", err)
	}
	activeCount := 0
	for _, p := range providers {
		if p.IsActive {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Fatalf("Expected exactly 1 active provider remaining, got %d", activeCount)
	}
}
