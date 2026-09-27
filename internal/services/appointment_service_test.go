package services_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage"
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
	if err := configRepo.SaveProvider(context.Background(), &domain.Provider{ID: "test_user", Name: "Test User", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	auditService := services.NewAuditService(auditRepo, configRepo)
	token, err := auditService.CreateSession("test_user", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	patientRepo := sqlite.NewPatientRepository(db)
	patientService := services.NewPatientService(patientRepo, auditService)

	appointmentRepo := sqlite.NewAppointmentRepository(db)
	service := services.NewAppointmentService(appointmentRepo, auditService)

	if err := configRepo.SaveProvider(context.Background(), &domain.Provider{ID: "prov_1", Name: "Dr One", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	if err := configRepo.SaveOperatory(context.Background(), &domain.Operatory{ID: "chair_1", Name: "Chair 1", IsActive: true}); err != nil {
		t.Fatalf("Failed to save operatory: %v", err)
	}

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

	invalid := []struct {
		name string
		appt domain.Appointment
	}{
		{"end before start", domain.Appointment{PatientID: p.ID, StartTime: end, EndTime: start}},
		{"zero length", domain.Appointment{PatientID: p.ID, StartTime: start, EndTime: start}},
		{"missing times", domain.Appointment{PatientID: p.ID}},
		{"unknown status", domain.Appointment{PatientID: p.ID, StartTime: start, EndTime: end, Status: "bogus"}},
	}
	for _, tc := range invalid {
		appt := tc.appt
		if _, err := service.CreateAppointment(token, &appt); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("CreateAppointment(%s): expected ErrInvalidInput, got %v", tc.name, err)
		}
	}

	bad := *updated
	bad.EndTime = bad.StartTime.Add(-time.Hour)
	if _, err := service.UpdateAppointment(token, &bad); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("UpdateAppointment with inverted times: expected ErrInvalidInput, got %v", err)
	}
	if _, err := service.UpdateAppointmentStatus(token, appt.ID, "bogus"); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("UpdateAppointmentStatus with unknown status: expected ErrInvalidInput, got %v", err)
	}
}
