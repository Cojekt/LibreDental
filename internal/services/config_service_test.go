package services_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage"
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
	service := services.NewPracticeConfigService(repo, nil)

	// 1. Initial GetConfig should return (nil, nil) when unconfigured
	cfg, err := service.GetConfig()
	if err != nil {
		t.Fatalf("Expected no error for unconfigured practice, got: %v", err)
	}
	if cfg != nil {
		t.Errorf("Expected nil config when unconfigured, got: %+v", cfg)
	}

	// 2. SetConfig (Onboarding flow for US)
	setCfg, err := service.SetConfig("", "US")
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
	updatedCfg, err := service.UpdatePracticeConfig("", *fetchedCfg)
	if err != nil {
		t.Fatalf("Failed to update practice config: %v", err)
	}
	if updatedCfg.ClinicName != "Bright Smiles Dental" {
		t.Errorf("Expected practice name 'Bright Smiles Dental', got '%s'", updatedCfg.ClinicName)
	}

	// 6. Provider Management Flow
	prov, err := service.SaveProvider("", domain.Provider{
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

	// Deactivating the sole active provider must be forbidden.
	if err := service.DeleteProvider("", prov.ID); err == nil {
		t.Fatalf("Expected error deactivating the last active provider, got nil")
	}

	// Add a second provider so the first can be deactivated.
	prov2, err := service.SaveProvider("", domain.Provider{
		Name:     "Dr. John Wick",
		Role:     domain.RoleDentist,
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("Failed to save second provider: %v", err)
	}

	err = service.DeleteProvider("", prov.ID)
	if err != nil {
		t.Fatalf("Failed to delete provider: %v", err)
	}

	providersAfterDelete, err := service.ListProviders()
	if err != nil {
		t.Fatalf("Failed to list providers after delete: %v", err)
	}
	if len(providersAfterDelete) != 2 {
		t.Fatalf("Expected 2 providers after delete, got %d", len(providersAfterDelete))
	}

	// Now only prov2 remains active; deactivating it must also be forbidden.
	if err := service.DeleteProvider("", prov2.ID); err == nil {
		t.Fatalf("Expected error deactivating the last remaining active provider, got nil")
	}

	// 7. Operatory Management Flow
	op, err := service.SaveOperatory("", domain.Operatory{
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

	err = service.DeleteOperatory("", op.ID)
	if err != nil {
		t.Fatalf("Failed to delete operatory: %v", err)
	}

	opsAfterDelete, err := service.ListOperatories()
	if err != nil {
		t.Fatalf("Failed to list operatories after delete: %v", err)
	}
	if len(opsAfterDelete) != 1 || opsAfterDelete[0].IsActive {
		t.Errorf("Expected operatory to be inactive after deletion, got active or wrong count: %d", len(opsAfterDelete))
	}
}

func TestPracticeConfigService_OnboardingAndSessionGating(t *testing.T) {
	tempDir := t.TempDir()

	db, err := sqlite.Open(filepath.Join(tempDir, "main.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	auditDb, err := sqlite.OpenAudit(filepath.Join(tempDir, "audit.db"))
	if err != nil {
		t.Fatalf("Failed to open audit sqlite db: %v", err)
	}
	defer auditDb.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	auditService := services.NewAuditService(sqlite.NewAuditRepository(auditDb), repo)
	service := services.NewPracticeConfigService(repo, auditService)

	// Step 1: country selection works without a session on a fresh install.
	if _, err := service.SetConfig("", "US"); err != nil {
		t.Fatalf("Expected unauthenticated SetConfig before any provider exists, got %v", err)
	}

	needs, err := service.NeedsInitialProvider()
	if err != nil || !needs {
		t.Fatalf("Expected fresh install to need an initial provider, got needs=%v err=%v", needs, err)
	}

	// Nothing else can be changed without a session.
	if _, err := service.SaveProvider("", domain.Provider{Name: "Sneaky", Pin: "0000", IsActive: true}); !errors.Is(err, services.ErrUnauthorized) {
		t.Fatalf("Expected SaveProvider without session to be unauthorized, got %v", err)
	}

	for _, bad := range []domain.Provider{
		{Name: "", Pin: "1234"},
		{Name: "Dr. Short", Pin: "123"},
		{Name: "Dr. Alpha", Pin: "12a4"},
	} {
		if _, err := service.CreateInitialProvider(bad); !errors.Is(err, storage.ErrInvalidInput) {
			t.Fatalf("Expected ErrInvalidInput for %+v, got %v", bad, err)
		}
	}

	// Step 2: first provider is created and logged in.
	token, err := service.CreateInitialProvider(domain.Provider{Name: "Dr. First", Pin: "1234"})
	if err != nil {
		t.Fatalf("Failed to create initial provider: %v", err)
	}
	user := auditService.GetSessionUser(token)
	if user == nil || user.Name != "Dr. First" || user.Role != domain.RoleDentist {
		t.Fatalf("Expected session for the new dentist provider, got %+v", user)
	}

	logs, err := auditService.GetAuditLogs(token, "", 10, 0)
	if err != nil {
		t.Fatalf("Failed to read audit logs: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != domain.AuditActionCreate || logs[0].UserID != user.ID || logs[0].Resource != "provider" {
		t.Fatalf("Expected one provider-create audit entry by the new provider, got %+v", logs)
	}

	needs, err = service.NeedsInitialProvider()
	if err != nil || needs {
		t.Fatalf("Expected no initial provider needed after bootstrap, got needs=%v err=%v", needs, err)
	}

	if _, err := service.CreateInitialProvider(domain.Provider{Name: "Dr. Second", Pin: "5678"}); !errors.Is(err, storage.ErrAlreadyInitialized) {
		t.Fatalf("Expected second bootstrap to fail with ErrAlreadyInitialized, got %v", err)
	}

	// Once a provider exists, every config mutation requires a session.
	if _, err := service.SetConfig("", "US"); !errors.Is(err, services.ErrUnauthorized) {
		t.Fatalf("Expected SetConfig without session to be unauthorized after bootstrap, got %v", err)
	}
	cfg, err := service.GetConfig()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}
	if _, err := service.UpdatePracticeConfig("", *cfg); !errors.Is(err, services.ErrUnauthorized) {
		t.Fatalf("Expected UpdatePracticeConfig without session to be unauthorized, got %v", err)
	}
	if err := service.DeleteProvider("", user.ID); !errors.Is(err, services.ErrUnauthorized) {
		t.Fatalf("Expected DeleteProvider without session to be unauthorized, got %v", err)
	}
	if _, err := service.SaveOperatory("", domain.Operatory{Name: "Op"}); !errors.Is(err, services.ErrUnauthorized) {
		t.Fatalf("Expected SaveOperatory without session to be unauthorized, got %v", err)
	}
	if err := service.DeleteOperatory("", "op_x"); !errors.Is(err, services.ErrUnauthorized) {
		t.Fatalf("Expected DeleteOperatory without session to be unauthorized, got %v", err)
	}

	// With the session, the same calls go through.
	if _, err := service.SetConfig(token, "US"); err != nil {
		t.Fatalf("Expected SetConfig with session to succeed, got %v", err)
	}
	if _, err := service.SaveProvider(token, domain.Provider{Name: "Dr. Second", Pin: "5678", IsActive: true}); err != nil {
		t.Fatalf("Expected SaveProvider with session to succeed, got %v", err)
	}
	if _, err := service.SaveOperatory(token, domain.Operatory{Name: "Op 1", IsActive: true}); err != nil {
		t.Fatalf("Expected SaveOperatory with session to succeed, got %v", err)
	}

	// PIN lookup for forgotten PINs: requires a session and is audit-logged.
	if _, err := service.GetProviderPin("", user.ID); !errors.Is(err, services.ErrUnauthorized) {
		t.Fatalf("Expected GetProviderPin without session to be unauthorized, got %v", err)
	}
	pin, err := service.GetProviderPin(token, user.ID)
	if err != nil || pin != "1234" {
		t.Fatalf("Expected GetProviderPin to return 1234, got %q err=%v", pin, err)
	}
	if _, err := service.GetProviderPin(token, "prov_missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Expected ErrNotFound for unknown provider, got %v", err)
	}
	logs, err = auditService.GetAuditLogs(token, "", 50, 0)
	if err != nil {
		t.Fatalf("Failed to read audit logs: %v", err)
	}
	revealLogged := false
	for _, l := range logs {
		if l.Action == domain.AuditActionRead && l.Resource == "provider" {
			revealLogged = true
		}
	}
	if !revealLogged {
		t.Fatalf("Expected a READ audit entry for the PIN reveal, got %+v", logs)
	}
}

func TestPracticeConfigService_RejectsMalformedClaimIdentifiers(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test_config_identifiers.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()
	service := services.NewPracticeConfigService(sqlite.NewPracticeConfigRepository(db), nil)

	cfg, err := service.SetConfig("", "US")
	if err != nil {
		t.Fatalf("Failed to set practice config: %v", err)
	}
	cfg.NPI = "1234567890" // bad check digit
	if _, err := service.UpdatePracticeConfig("", *cfg); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected invalid practice NPI to be rejected, got %v", err)
	}
	cfg.NPI = " 1234567893 "
	saved, err := service.UpdatePracticeConfig("", *cfg)
	if err != nil || saved.NPI != "1234567893" {
		t.Errorf("Expected a valid NPI to be trimmed and saved, got %+v, %v", saved, err)
	}

	for _, p := range []domain.Provider{
		{Name: "Dr. A", Pin: "1111", IsActive: true, NPI: "12345"},
		{Name: "Dr. B", Pin: "2222", IsActive: true, NPI: "1234567893", TaxonomyCode: "dentist"},
	} {
		if _, err := service.SaveProvider("", p); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected %s to be rejected, got %v", p.Name, err)
		}
	}
	saved2, err := service.SaveProvider("", domain.Provider{Name: "Dr. C", Pin: "3333", IsActive: true,
		NPI: "1234567893", TaxonomyCode: " 1223g0001x "})
	if err != nil || saved2.TaxonomyCode != "1223G0001X" {
		t.Errorf("Expected taxonomy to be normalized and saved, got %+v, %v", saved2, err)
	}
}

// System actor IDs are reserved for the audit trail: a staff account holding one would make its
// actions look like they were taken automatically.
func TestPracticeConfigService_RejectsReservedProviderIDs(t *testing.T) {
	tempDir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(tempDir, "main.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()
	auditDb, err := sqlite.OpenAudit(filepath.Join(tempDir, "audit.db"))
	if err != nil {
		t.Fatalf("Failed to open audit sqlite db: %v", err)
	}
	defer auditDb.Close()

	repo := sqlite.NewPracticeConfigRepository(db)
	auditService := services.NewAuditService(sqlite.NewAuditRepository(auditDb), repo)
	service := services.NewPracticeConfigService(repo, auditService)
	if _, err := service.SetConfig("", "US"); err != nil {
		t.Fatalf("Failed to set practice config: %v", err)
	}

	reserved := []string{domain.SystemActorReminders, "system:other", "SYSTEM:reminders", " system:reminders"}

	for _, id := range reserved {
		if _, err := service.CreateInitialProvider(domain.Provider{ID: id, Name: "Imposter", Pin: "1234", IsActive: true}); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected CreateInitialProvider to reject %q, got %v", id, err)
		}
	}
	if needs, err := service.NeedsInitialProvider(); err != nil || !needs {
		t.Fatalf("Expected no provider to have been created, got needs=%v err=%v", needs, err)
	}

	token, err := service.CreateInitialProvider(domain.Provider{Name: "Dr. Owner", Pin: "1234", IsActive: true})
	if err != nil {
		t.Fatalf("Failed to create initial provider: %v", err)
	}

	for _, id := range reserved {
		if _, err := service.SaveProvider(token, domain.Provider{ID: id, Name: "Imposter", Pin: "5678", IsActive: true}); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected SaveProvider to reject %q, got %v", id, err)
		}
	}

	// IDs that merely contain "system" are ordinary staff IDs.
	for _, id := range []string{"", "prov_system", "systemadmin"} {
		if _, err := service.SaveProvider(token, domain.Provider{ID: id, Name: "Dr. Real", Pin: "5678", IsActive: true}); err != nil {
			t.Errorf("Expected SaveProvider to accept %q, got %v", id, err)
		}
	}

	providers, err := service.ListProviders()
	if err != nil {
		t.Fatalf("Failed to list providers: %v", err)
	}
	for _, p := range providers {
		if domain.IsSystemActorID(strings.ToLower(strings.TrimSpace(p.ID))) {
			t.Errorf("Reserved provider ID %q was saved", p.ID)
		}
	}
}
