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
