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
	cfg.NPI = "1234567893"
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
	if fetched.NPI != "1234567893" {
		t.Errorf("Expected NPI '1234567893', got '%s'", fetched.NPI)
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
		NPI:           "1234567893",
		TaxonomyCode:  "1223E0200X",
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
	if providers[0].NPI != "1234567893" || providers[0].TaxonomyCode != "1223E0200X" {
		t.Errorf("Unexpected provider NPI/taxonomy: %s / %s", providers[0].NPI, providers[0].TaxonomyCode)
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

func TestPracticeConfigRepository_CreateInitialProvider(t *testing.T) {
	tempDir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(tempDir, "test_initial_provider.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()

	has, err := repo.HasActiveProvider(ctx)
	if err != nil || has {
		t.Fatalf("Expected no active provider on fresh db, got has=%v err=%v", has, err)
	}

	first := &domain.Provider{ID: "prov_first", Name: "First", Role: domain.RoleDentist, Pin: "1234"}
	if err := repo.CreateInitialProvider(ctx, first); err != nil {
		t.Fatalf("Failed to create initial provider: %v", err)
	}
	if !first.IsActive {
		t.Errorf("Expected initial provider to be forced active")
	}

	has, err = repo.HasActiveProvider(ctx)
	if err != nil || !has {
		t.Fatalf("Expected an active provider after bootstrap, got has=%v err=%v", has, err)
	}

	second := &domain.Provider{ID: "prov_second", Name: "Second", Role: domain.RoleDentist, Pin: "5678"}
	if err := repo.CreateInitialProvider(ctx, second); !errors.Is(err, storage.ErrAlreadyInitialized) {
		t.Fatalf("Expected ErrAlreadyInitialized once a provider exists, got %v", err)
	}

	providers, err := repo.ListProviders(ctx)
	if err != nil {
		t.Fatalf("Failed to list providers: %v", err)
	}
	if len(providers) != 1 || providers[0].ID != "prov_first" {
		t.Fatalf("Expected only the first provider to exist, got %+v", providers)
	}
}

func TestPracticeConfigRepository_CreateInitialProvider_ConcurrentRace(t *testing.T) {
	tempDir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(tempDir, "test_initial_provider_race.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()

	ids := []string{"prov_a", "prov_b", "prov_c", "prov_d"}
	var wg sync.WaitGroup
	errs := make([]error, len(ids))
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			errs[i] = repo.CreateInitialProvider(ctx, &domain.Provider{ID: id, Name: id, Role: domain.RoleDentist, Pin: "1234"})
		}(i, id)
	}
	wg.Wait()

	successCount := 0
	for _, err := range errs {
		if err == nil {
			successCount++
		} else if !errors.Is(err, storage.ErrAlreadyInitialized) {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
	if successCount != 1 {
		t.Fatalf("Expected exactly 1 concurrent bootstrap to succeed, got %d", successCount)
	}
}

func TestPracticeConfigRepository_SaveProvider_LastActiveGuard(t *testing.T) {
	tempDir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(tempDir, "test_save_last_active.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()

	only := &domain.Provider{ID: "prov_only", Name: "Only", Role: domain.RoleDentist, IsActive: true}
	if err := repo.SaveProvider(ctx, only); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}

	only.IsActive = false
	if err := repo.SaveProvider(ctx, only); !errors.Is(err, storage.ErrLastActiveProvider) {
		t.Fatalf("Expected ErrLastActiveProvider when deactivating the only active provider, got %v", err)
	}

	only.IsActive = true
	only.Name = "Renamed"
	if err := repo.SaveProvider(ctx, only); err != nil {
		t.Fatalf("Expected editing an active provider to succeed, got %v", err)
	}

	other := &domain.Provider{ID: "prov_other", Name: "Other", Role: domain.RoleDentist, IsActive: true}
	if err := repo.SaveProvider(ctx, other); err != nil {
		t.Fatalf("Failed to save second provider: %v", err)
	}

	only.IsActive = false
	if err := repo.SaveProvider(ctx, only); err != nil {
		t.Fatalf("Expected deactivation to succeed with another active provider, got %v", err)
	}

	// Editing an already-inactive provider must not trip the guard.
	only.Name = "Renamed Again"
	if err := repo.SaveProvider(ctx, only); err != nil {
		t.Fatalf("Expected editing an inactive provider to succeed, got %v", err)
	}
}

func TestPracticeConfigRepository_SaveInitialConfig(t *testing.T) {
	tempDir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(tempDir, "test_initial_config.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()

	if err := repo.SaveInitialConfig(ctx, &domain.PracticeConfig{CountryCode: "US", Currency: "USD"}); err != nil {
		t.Fatalf("Expected initial config save before any provider, got %v", err)
	}
	if err := repo.SaveInitialConfig(ctx, &domain.PracticeConfig{CountryCode: "CA", Currency: "CAD"}); err != nil {
		t.Fatalf("Expected initial config update before any provider, got %v", err)
	}

	if err := repo.CreateInitialProvider(ctx, &domain.Provider{ID: "prov_first", Name: "First", Role: domain.RoleDentist, Pin: "1234"}); err != nil {
		t.Fatalf("Failed to create initial provider: %v", err)
	}

	if err := repo.SaveInitialConfig(ctx, &domain.PracticeConfig{CountryCode: "GB", Currency: "GBP"}); !errors.Is(err, storage.ErrAlreadyInitialized) {
		t.Fatalf("Expected ErrAlreadyInitialized once a provider exists, got %v", err)
	}

	cfg, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}
	if cfg.CountryCode != "CA" {
		t.Fatalf("Expected config to remain CA after rejected save, got %s", cfg.CountryCode)
	}
}

func TestPracticeConfigRepository_SaveInitialConfig_ConcurrentRace(t *testing.T) {
	tempDir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(tempDir, "test_initial_config_race.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()

	countries := []domain.CountryCode{"US", "CA", "GB", "AU", "NZ", "IE"}
	var wg sync.WaitGroup
	saveErrs := make([]error, len(countries))
	var providerErr error
	for i, code := range countries {
		wg.Add(1)
		go func(i int, code domain.CountryCode) {
			defer wg.Done()
			saveErrs[i] = repo.SaveInitialConfig(ctx, &domain.PracticeConfig{CountryCode: code})
		}(i, code)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		providerErr = repo.CreateInitialProvider(ctx, &domain.Provider{ID: "prov_first", Name: "First", Role: domain.RoleDentist, Pin: "1234"})
	}()
	wg.Wait()

	if providerErr != nil {
		t.Fatalf("Expected initial provider creation to succeed, got %v", providerErr)
	}

	saved := map[domain.CountryCode]bool{}
	for i, err := range saveErrs {
		if err == nil {
			saved[countries[i]] = true
		} else if !errors.Is(err, storage.ErrAlreadyInitialized) {
			t.Fatalf("Unexpected error saving %s: %v", countries[i], err)
		}
	}

	// The provider may win before any save; otherwise a rejected save must never be the one that landed.
	cfg, err := repo.Get(ctx)
	switch {
	case errors.Is(err, storage.ErrNotFound):
		if len(saved) != 0 {
			t.Fatalf("Expected no stored config when no save succeeded, got successes %v", saved)
		}
	case err != nil:
		t.Fatalf("Failed to get config: %v", err)
	case !saved[cfg.CountryCode]:
		t.Fatalf("Config country %s came from a save that reported ErrAlreadyInitialized", cfg.CountryCode)
	}

	if err := repo.SaveInitialConfig(ctx, &domain.PracticeConfig{CountryCode: "DE"}); !errors.Is(err, storage.ErrAlreadyInitialized) {
		t.Fatalf("Expected ErrAlreadyInitialized after the race, got %v", err)
	}
}

func TestPracticeConfigRepository_Timezone(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "timezone.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()
	repo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()

	cfg := &domain.PracticeConfig{ClinicName: "Smile Dental", CountryCode: domain.CountryUS, Currency: "USD"}
	if err := repo.Save(ctx, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	got, err := repo.Get(ctx)
	if err != nil || got.Timezone != "" {
		t.Fatalf("Expected no timezone by default, got %q, %v", got.Timezone, err)
	}
	got.Timezone = "America/Los_Angeles"
	if err := repo.Save(ctx, got); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if got, err = repo.Get(ctx); err != nil || got.Timezone != "America/Los_Angeles" {
		t.Errorf("Timezone didn't round-trip: %q, %v", got.Timezone, err)
	}
}
