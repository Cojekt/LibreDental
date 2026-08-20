package services_test

import (
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

	patientRepo := sqlite.NewPatientRepository(db)
	service := services.NewPatientService(patientRepo)

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

	created, err := service.CreatePatient(newPatient)
	if err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}
	if created.ID != "pat_101" {
		t.Errorf("Expected ID 'pat_101', got '%s'", created.ID)
	}

	// 2. Get Patient
	fetched, err := service.GetPatient("pat_101")
	if err != nil {
		t.Fatalf("Failed to get patient: %v", err)
	}
	if fetched.FirstName != "Alice" || fetched.LastName != "Smith" {
		t.Errorf("Unexpected patient name: %s %s", fetched.FirstName, fetched.LastName)
	}

	// 3. List Patients
	list, err := service.ListPatients("Alice", string(domain.StatusActive))
	if err != nil {
		t.Fatalf("Failed to list patients: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 patient in list, got %d", len(list))
	}

	// Empty query list
	all, err := service.ListPatients("", "")
	if err != nil {
		t.Fatalf("Failed to list all patients: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("Expected 1 patient in total list, got %d", len(all))
	}

	// 4. Update Patient
	fetched.LastName = "Johnson"
	updated, err := service.UpdatePatient(fetched)
	if err != nil {
		t.Fatalf("Failed to update patient: %v", err)
	}
	if updated.LastName != "Johnson" {
		t.Errorf("Expected updated last name 'Johnson', got '%s'", updated.LastName)
	}

	// 5. Archive Patient
	err = service.ArchivePatient("pat_101")
	if err != nil {
		t.Fatalf("Failed to archive patient: %v", err)
	}

	archived, err := service.GetPatient("pat_101")
	if err != nil {
		t.Fatalf("Failed to get archived patient: %v", err)
	}
	if archived.Status != domain.StatusArchived {
		t.Errorf("Expected status '%s', got '%s'", domain.StatusArchived, archived.Status)
	}

	// Archive nonexistent patient returns error
	err = service.ArchivePatient("non_existent_id")
	if err == nil {
		t.Errorf("Expected error archiving non-existent patient, got nil")
	}
}
