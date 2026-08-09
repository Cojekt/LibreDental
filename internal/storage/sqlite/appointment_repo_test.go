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

func TestAppointmentRepository_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_libredental.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	patientRepo := sqlite.NewPatientRepository(db)
	repo := sqlite.NewAppointmentRepository(db)
	ctx := context.Background()

	// Create test patient first (foreign key reference)
	patient := &domain.Patient{
		ID:          "pat_123",
		FirstName:   "Jane",
		LastName:    "Doe",
		DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Sex:         domain.SexFemale,
	}
	if err := patientRepo.Create(ctx, patient); err != nil {
		t.Fatalf("Failed to create test patient: %v", err)
	}

	start := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)

	// 1. Create Appointment
	appt := &domain.Appointment{
		ID:          "appt_001",
		PatientID:   "pat_123",
		ProviderID:  "prov_dentist_1",
		OperatoryID: "op_chair_1",
		StartTime:   start,
		EndTime:     end,
		Status:      domain.AppointmentStatusScheduled,
		Reason:      "Routine Cleaning & Checkup",
		Color:       "#3b82f6",
		Notes:       "Patient prefers morning slots",
	}

	err = repo.Create(ctx, appt)
	if err != nil {
		t.Fatalf("Failed to create appointment: %v", err)
	}

	// 2. Read Appointment
	fetched, err := repo.GetByID(ctx, "appt_001")
	if err != nil {
		t.Fatalf("Failed to get appointment by ID: %v", err)
	}
	if fetched.PatientID != "pat_123" || fetched.Reason != "Routine Cleaning & Checkup" {
		t.Errorf("Unexpected appointment data: %v", fetched)
	}
	if fetched.Status != domain.AppointmentStatusScheduled {
		t.Errorf("Expected status 'scheduled', got '%s'", fetched.Status)
	}

	// 3. Update Appointment
	fetched.Status = domain.AppointmentStatusConfirmed
	fetched.Notes = "Confirmed via SMS"
	err = repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Failed to update appointment: %v", err)
	}
	if fetched.Version != 2 {
		t.Errorf("Expected version 2, got %d", fetched.Version)
	}

	// Stale update check
	staleAppt := &domain.Appointment{
		ID:        "appt_001",
		PatientID: "pat_123",
		Version:   1,
	}
	err = repo.Update(ctx, staleAppt)
	if err != storage.ErrConflict {
		t.Errorf("Expected ErrConflict for stale update, got: %v", err)
	}

	// 4. List Appointments
	filter := domain.AppointmentFilter{
		StartDate: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 8, 3, 23, 59, 59, 0, time.UTC),
	}
	list, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to list appointments: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 appointment in date filter, got %d", len(list))
	}

	// 5. Delete Appointment
	err = repo.Delete(ctx, "appt_001")
	if err != nil {
		t.Fatalf("Failed to delete appointment: %v", err)
	}

	_, err = repo.GetByID(ctx, "appt_001")
	if err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got: %v", err)
	}
}
