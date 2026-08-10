package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestPatientRepository_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_libredental.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPatientRepository(db)
	ctx := context.Background()

	// 1. Create Patient
	patient := &domain.Patient{
		ID:                    "pat_123",
		FirstName:             "John",
		LastName:              "Doe",
		DateOfBirth:           time.Date(1985, 5, 20, 0, 0, 0, 0, time.UTC),
		Sex:                   domain.SexMale,
		Email:                 "john.doe@example.com",
		PhonePrimary:          "555-0199",
		EmergencyContactName:  "Mary Doe",
		EmergencyContactRel:   "Spouse",
		EmergencyContactPhone: "555-0188",
		InsuranceCarrier:      "Delta Dental",
		InsurancePolicyNumber: "DEL-12345",
		StateProvince:         "CA",
		PostalCode:            "90210",
		CountryCode:           domain.CountryUS,
		NationalIDType:        "ssn",
		NationalID:            "123-45-6789",
		MedicalAlerts:         []string{"Penicillin Allergy", "High Blood Pressure"},
		Allergies:             []string{"Latex"},
	}

	err = repo.Create(ctx, patient)
	if err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	// 2. Read Patient
	fetched, err := repo.GetByID(ctx, "pat_123")
	if err != nil {
		t.Fatalf("Failed to get patient by ID: %v", err)
	}
	if fetched.FirstName != "John" || fetched.LastName != "Doe" {
		t.Errorf("Unexpected patient name: %s %s", fetched.FirstName, fetched.LastName)
	}
	if fetched.NationalID != "123-45-6789" || fetched.CountryCode != domain.CountryUS {
		t.Errorf("Unexpected national ID / country: %s / %s", fetched.NationalID, fetched.CountryCode)
	}
	if len(fetched.MedicalAlerts) != 2 {
		t.Errorf("Expected 2 medical alerts, got %d", len(fetched.MedicalAlerts))
	}

	// 3. Update Patient (Optimistic Concurrency)
	fetched.LastName = "Smith"
	err = repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Failed to update patient: %v", err)
	}
	if fetched.Version != 2 {
		t.Errorf("Expected version 2, got %d", fetched.Version)
	}

	// Attempt stale update (should cause storage.ErrConflict)
	stalePatient := &domain.Patient{
		ID:        "pat_123",
		FirstName: "Stale",
		LastName:  "Stale",
		Version:   1, // Stale version
	}
	err = repo.Update(ctx, stalePatient)
	if err != storage.ErrConflict {
		t.Errorf("Expected ErrConflict for stale update, got: %v", err)
	}

	// 4. List Patients
	patients, total, err := repo.List(ctx, domain.PatientFilter{Query: "smith"})
	if err != nil {
		t.Fatalf("Failed to list patients: %v", err)
	}
	if total != 1 || len(patients) != 1 {
		t.Errorf("Expected 1 result for 'smith', got total=%d len=%d", total, len(patients))
	}

	// 5. Delete Patient
	err = repo.Delete(ctx, "pat_123")
	if err != nil {
		t.Fatalf("Failed to delete patient: %v", err)
	}

	_, err = repo.GetByID(ctx, "pat_123")
	if err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got: %v", err)
	}
}
