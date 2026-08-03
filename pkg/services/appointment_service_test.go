package services_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/services"
	"github.com/LibreDental/libredental/pkg/storage/sqlite"
)

func TestAppointmentService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_service.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	patientRepo := sqlite.NewPatientRepository(db)
	patientService := services.NewPatientService(patientRepo)

	apptRepo := sqlite.NewAppointmentRepository(db)
	service := services.NewAppointmentService(apptRepo)

	p, err := patientService.CreatePatient(&domain.Patient{
		ID:        "pat_001",
		FirstName: "Alice",
		LastName:  "Smith",
	})
	if err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	start := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC)

	appt, err := service.CreateAppointment(&domain.Appointment{
		PatientID:   p.ID,
		ProviderID:  "prov_1",
		OperatoryID: "chair_1",
		StartTime:   start,
		EndTime:     end,
		Status:      domain.AppointmentStatusScheduled,
		Reason:      "Consultation",
	})
	if err != nil {
		t.Fatalf("Failed to create appointment via service: %v", err)
	}
	if appt.ID == "" {
		t.Errorf("Expected generated ID for appointment")
	}

	// Update status
	updated, err := service.UpdateAppointmentStatus(appt.ID, string(domain.AppointmentStatusConfirmed))
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}
	if updated.Status != domain.AppointmentStatusConfirmed {
		t.Errorf("Expected status 'confirmed', got %s", updated.Status)
	}

	// List appointments
	list, err := service.ListAppointments(domain.AppointmentFilter{
		PatientID: p.ID,
	})
	if err != nil {
		t.Fatalf("Failed to list appointments: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 appointment in list, got %d", len(list))
	}
}
