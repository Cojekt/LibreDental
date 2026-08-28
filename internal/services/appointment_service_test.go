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

func TestAppointmentService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_service.db")

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
	configRepo.SaveProvider(context.Background(), &domain.Provider{ID: "test_user", Name: "Test User", Pin: "1234", IsActive: true})
	auditService := services.NewAuditService(auditRepo, configRepo)
	token, _ := auditService.CreateSession("test_user", "1234")

	patientRepo := sqlite.NewPatientRepository(db)
	patientService := services.NewPatientService(patientRepo, auditService)

	appointmentRepo := sqlite.NewAppointmentRepository(db)
	service := services.NewAppointmentService(appointmentRepo, auditService)

	p, err := patientService.CreatePatient(token, &domain.Patient{
		ID:        "pat_001",
		FirstName: "Alice",
		LastName:  "Smith",
	})
	if err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	start := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC)

	appt, err := service.CreateAppointment(token, &domain.Appointment{
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
	updated, err := service.UpdateAppointmentStatus(token, appt.ID, string(domain.AppointmentStatusConfirmed))
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}
	if updated.Status != domain.AppointmentStatusConfirmed {
		t.Errorf("Expected status 'confirmed', got %s", updated.Status)
	}

	// List appointments
	list, err := service.ListAppointments(token, domain.AppointmentFilter{
		PatientID: p.ID,
	})
	if err != nil {
		t.Fatalf("Failed to list appointments: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 appointment in list, got %d", len(list))
	}
}
