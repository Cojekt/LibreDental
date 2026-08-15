package services_test

import (
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestPracticeConfigService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_config_service.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	service := services.NewPracticeConfigService(repo)

	// 1. Initial GetConfig should return (nil, nil) when unconfigured
	cfg, err := service.GetConfig()
	if err != nil {
		t.Fatalf("Expected no error for unconfigured practice, got: %v", err)
	}
	if cfg != nil {
		t.Errorf("Expected nil config when unconfigured, got: %+v", cfg)
	}

	// 2. SetConfig (Onboarding flow for US)
	setCfg, err := service.SetConfig("US")
	if err != nil {
		t.Fatalf("Failed to set practice config: %v", err)
	}
	if setCfg.CountryCode != domain.CountryUS {
		t.Errorf("Expected country code 'US', got '%s'", setCfg.CountryCode)
	}
	if setCfg.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", setCfg.Currency)
	}

	// 3. GetConfig after set
	fetchedCfg, err := service.GetConfig()
	if err != nil {
		t.Fatalf("Failed to get practice config: %v", err)
	}
	if fetchedCfg.CountryCode != domain.CountryUS {
		t.Errorf("Expected country code 'US', got '%s'", fetchedCfg.CountryCode)
	}

	// 4. GetSupportedCountries & GetCountryConfig
	countries, err := service.GetSupportedCountries()
	if err != nil {
		t.Fatalf("Failed to list supported countries: %v", err)
	}
	if len(countries) == 0 {
		t.Errorf("Expected supported countries list to be non-empty")
	}

	caMeta, err := service.GetCountryConfig("CA")
	if err != nil {
		t.Fatalf("Failed to get country config for CA: %v", err)
	}
	if caMeta.Code != domain.CountryCA {
		t.Errorf("Expected country code 'CA', got '%s'", caMeta.Code)
	}

	// Fallback to default for unknown country
	fallbackMeta, err := service.GetCountryConfig("UNKNOWN_CODE")
	if err != nil {
		t.Fatalf("Failed to get fallback country config: %v", err)
	}
	if fallbackMeta.Code != domain.CountryUS {
		t.Errorf("Expected default fallback country code 'US', got '%s'", fallbackMeta.Code)
	}

	// 5. UpdatePracticeConfig
	fetchedCfg.ClinicName = "Bright Smiles Dental"
	updatedCfg, err := service.UpdatePracticeConfig(*fetchedCfg)
	if err != nil {
		t.Fatalf("Failed to update practice config: %v", err)
	}
	if updatedCfg.ClinicName != "Bright Smiles Dental" {
		t.Errorf("Expected practice name 'Bright Smiles Dental', got '%s'", updatedCfg.ClinicName)
	}

	// 6. Provider Management Flow
	prov, err := service.SaveProvider(domain.Provider{
		Name:      "Dr. Sarah Connor",
		Role:      domain.RoleDentist,
		Specialty: "General Dentistry",
		Email:     "sarah@example.com",
		IsActive:  true,
	})
	if err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	if prov.ID == "" {
		t.Errorf("Expected generated provider ID")
	}

	providers, err := service.ListProviders()
	if err != nil {
		t.Fatalf("Failed to list providers: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(providers))
	}
	if providers[0].Name != "Dr. Sarah Connor" {
		t.Errorf("Expected provider name 'Dr. Sarah Connor', got '%s'", providers[0].Name)
	}

	err = service.DeleteProvider(prov.ID)
	if err != nil {
		t.Fatalf("Failed to delete provider: %v", err)
	}

	providersAfterDelete, err := service.ListProviders()
	if err != nil {
		t.Fatalf("Failed to list providers after delete: %v", err)
	}
	if len(providersAfterDelete) != 0 {
		t.Errorf("Expected 0 providers after deletion, got %d", len(providersAfterDelete))
	}

	// 7. Operatory Management Flow
	op, err := service.SaveOperatory(domain.Operatory{
		Name:     "Hygiene Suite 1",
		RoomCode: "HYG-1",
		Type:     domain.OperatoryTypeHygiene,
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("Failed to save operatory: %v", err)
	}
	if op.ID == "" {
		t.Errorf("Expected generated operatory ID")
	}

	operatories, err := service.ListOperatories()
	if err != nil {
		t.Fatalf("Failed to list operatories: %v", err)
	}
	if len(operatories) != 1 {
		t.Fatalf("Expected 1 operatory, got %d", len(operatories))
	}
	if operatories[0].RoomCode != "HYG-1" {
		t.Errorf("Expected room code 'HYG-1', got '%s'", operatories[0].RoomCode)
	}

	err = service.DeleteOperatory(op.ID)
	if err != nil {
		t.Fatalf("Failed to delete operatory: %v", err)
	}

	opsAfterDelete, err := service.ListOperatories()
	if err != nil {
		t.Fatalf("Failed to list operatories after delete: %v", err)
	}
	if len(opsAfterDelete) != 0 {
		t.Errorf("Expected 0 operatories after deletion, got %d", len(opsAfterDelete))
	}
}
