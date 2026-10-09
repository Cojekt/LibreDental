package services_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestPatientService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_patient_service.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	auditDbPath := filepath.Join(tempDir, "test_audit_service.db")
	auditDb, err := sqlite.OpenAudit(auditDbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite audit db: %v", err)
	}
	defer auditDb.Close()

	auditRepo := sqlite.NewAuditRepository(auditDb)
	configRepo := sqlite.NewPracticeConfigRepository(db)
	if err := configRepo.SaveProvider(context.Background(), &domain.Provider{ID: "test_user", Name: "Test User", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	auditService := services.NewAuditService(auditRepo, configRepo)
	token, err := auditService.CreateSession("test_user", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	patientRepo := sqlite.NewPatientRepository(db)
	service := services.NewPatientService(patientRepo, auditService)

	// 1. Create Patient
	newPatient := &domain.Patient{
		ID:          "pat_101",
		FirstName:   "Alice",
		LastName:    "Smith",
		DateOfBirth: time.Date(1990, 1, 15, 0, 0, 0, 0, time.UTC),
		Sex:         domain.SexFemale,
		Email:       "alice@example.com",
		Status:      domain.StatusActive,
	}

	created, err := service.CreatePatient(token, newPatient)
	if err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}
	if created.ID != "pat_101" {
		t.Errorf("Expected ID 'pat_101', got '%s'", created.ID)
	}

	// 2. Get Patient
	fetched, err := service.GetPatient(token, "pat_101")
	if err != nil {
		t.Fatalf("Failed to get patient: %v", err)
	}
	if fetched.FirstName != "Alice" || fetched.LastName != "Smith" {
		t.Errorf("Unexpected patient name: %s %s", fetched.FirstName, fetched.LastName)
	}

	// 3. List Patients
	list, err := service.ListPatients(token, "Alice", string(domain.StatusActive))
	if err != nil {
		t.Fatalf("Failed to list patients: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 patient in list, got %d", len(list))
	}

	// Empty query list
	all, err := service.ListPatients(token, "", "")
	if err != nil {
		t.Fatalf("Failed to list all patients: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("Expected 1 patient in total list, got %d", len(all))
	}

	// 4. Update Patient
	fetched.LastName = "Johnson"
	updated, err := service.UpdatePatient(token, fetched)
	if err != nil {
		t.Fatalf("Failed to update patient: %v", err)
	}
	if updated.LastName != "Johnson" {
		t.Errorf("Expected updated last name 'Johnson', got '%s'", updated.LastName)
	}

	// 5. Archive Patient
	err = service.ArchivePatient(token, "pat_101")
	if err != nil {
		t.Fatalf("Failed to archive patient: %v", err)
	}

	archived, err := service.GetPatient(token, "pat_101")
	if err != nil {
		t.Fatalf("Failed to get archived patient: %v", err)
	}
	if archived.Status != domain.StatusArchived {
		t.Errorf("Expected status '%s', got '%s'", domain.StatusArchived, archived.Status)
	}

	// Archive nonexistent patient returns error
	err = service.ArchivePatient(token, "non_existent_id")
	if err == nil {
		t.Errorf("Expected error archiving non-existent patient, got nil")
	}
}

// Automatic reminders are sent on the opt-in flag alone, so every change to it must be
// identifiable in the audit trail.
func TestPatientService_AuditsReminderConsent(t *testing.T) {
	tempDir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(tempDir, "patients.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()
	auditDb, err := sqlite.OpenAudit(filepath.Join(tempDir, "audit.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite audit db: %v", err)
	}
	defer auditDb.Close()

	configRepo := sqlite.NewPracticeConfigRepository(db)
	if err := configRepo.SaveProvider(context.Background(), &domain.Provider{ID: "prov_1", Name: "Front Desk", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	auditService := services.NewAuditService(sqlite.NewAuditRepository(auditDb), configRepo)
	token, err := auditService.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	service := services.NewPatientService(sqlite.NewPatientRepository(db), auditService)

	latest := func(patientID string) string {
		t.Helper()
		logs, err := auditService.GetAuditLogs(token, patientID, 1, 0)
		if err != nil || len(logs) == 0 {
			t.Fatalf("Failed to read audit log: %v", err)
		}
		if logs[0].UserID != "prov_1" {
			t.Errorf("Expected the entry attributed to the staff member, got %q", logs[0].UserID)
		}
		return logs[0].Details
	}

	if _, err := service.CreatePatient(token, &domain.Patient{ID: "pat_out", FirstName: "Ann", LastName: "Lee"}); err != nil {
		t.Fatal(err)
	}
	if got := latest("pat_out"); got != "Created new patient record; not opted in to automated reminders" {
		t.Errorf("Create (not opted in): %q", got)
	}
	if _, err := service.CreatePatient(token, &domain.Patient{ID: "pat_in", FirstName: "Bo", LastName: "Kim", ReminderOptIn: true}); err != nil {
		t.Fatal(err)
	}
	if got := latest("pat_in"); got != "Created new patient record; opted in to automated reminders" {
		t.Errorf("Create (opted in): %q", got)
	}

	steps := []struct {
		name   string
		mutate func(p *domain.Patient)
		want   string
	}{
		{"unrelated change", func(p *domain.Patient) { p.LastName = "Park" }, "Updated patient record"},
		{"opt in", func(p *domain.Patient) { p.ReminderOptIn = true }, "Updated patient record; opted in to automated reminders"},
		{"save again", func(p *domain.Patient) { p.Notes = "Prefers mornings" }, "Updated patient record"},
		{"opt out", func(p *domain.Patient) { p.ReminderOptIn = false }, "Updated patient record; opted out of automated reminders"},
	}
	for _, step := range steps {
		p, err := service.GetPatient(token, "pat_out")
		if err != nil {
			t.Fatal(err)
		}
		step.mutate(p)
		if _, err := service.UpdatePatient(token, p); err != nil {
			t.Fatalf("%s: UpdatePatient failed: %v", step.name, err)
		}
		if got := latest("pat_out"); got != step.want {
			t.Errorf("%s: audit detail %q; want %q", step.name, got, step.want)
		}
	}

	if _, err := service.UpdatePatient(token, &domain.Patient{ID: "missing", FirstName: "X", LastName: "Y"}); err == nil {
		t.Errorf("Expected updating a missing patient to fail")
	}
}
