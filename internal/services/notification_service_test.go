package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

type dummyNotificationProvider struct {
	name       string
	channel    domain.NotificationChannel
	sendErr    error
	sendStatus domain.NotificationStatus
}

func (p *dummyNotificationProvider) Name() string                        { return p.name }
func (p *dummyNotificationProvider) Channel() domain.NotificationChannel { return p.channel }
func (p *dummyNotificationProvider) Send(ctx context.Context, msg *domain.NotificationMessage, config map[string]string) (*domain.NotificationResult, error) {
	if p.sendErr != nil {
		return nil, p.sendErr
	}
	status := p.sendStatus
	if status == "" {
		status = domain.NotificationStatusSent
	}
	return &domain.NotificationResult{ExternalMessageID: "ext_1", Status: status}, nil
}

func newTestNotificationService(t *testing.T) (*NotificationService, *AuditService, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_notification_service.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	auditDb, err := sqlite.OpenAudit(filepath.Join(tmpDir, "test_notification_audit.db"))
	if err != nil {
		t.Fatalf("Failed to open audit db: %v", err)
	}
	t.Cleanup(func() { auditDb.Close() })

	patientRepo := sqlite.NewPatientRepository(db)
	appointmentRepo := sqlite.NewAppointmentRepository(db)
	logRepo := sqlite.NewNotificationRepository(db)
	auditRepo := sqlite.NewAuditRepository(auditDb)
	configRepo := sqlite.NewPracticeConfigRepository(db)

	ctx := context.Background()
	if err := configRepo.SaveProvider(ctx, &domain.Provider{ID: "prov_1", Name: "Test Prov", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	if err := configRepo.SaveOperatory(ctx, &domain.Operatory{ID: "op_1", Name: "Chair 1", IsActive: true}); err != nil {
		t.Fatalf("Failed to save operatory: %v", err)
	}

	auditSvc := NewAuditService(auditRepo, configRepo)
	token, err := auditSvc.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	secretsSvc := NewSecretsService()
	notificationSvc := NewNotificationService(patientRepo, appointmentRepo, logRepo, secretsSvc, auditSvc)

	return notificationSvc, auditSvc, token
}

func init() {
	keyring.MockInit()
}

func TestNotificationService_SendNotification(t *testing.T) {
	svc, _, token := newTestNotificationService(t)

	ctx := context.Background()
	optedIn := &domain.Patient{
		ID:            "pat_notif_1",
		FirstName:     "Jane",
		LastName:      "Doe",
		DateOfBirth:   time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Sex:           domain.SexFemale,
		Status:        domain.StatusActive,
		Email:         "jane@example.com",
		PhonePrimary:  "+15555550100",
		ReminderOptIn: true,
	}
	if err := svc.patientRepo.Create(ctx, optedIn); err != nil {
		t.Fatalf("Failed to create opted-in patient: %v", err)
	}

	optedOut := &domain.Patient{
		ID:            "pat_notif_2",
		FirstName:     "John",
		LastName:      "Roe",
		DateOfBirth:   time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Sex:           domain.SexMale,
		Status:        domain.StatusActive,
		Email:         "john@example.com",
		ReminderOptIn: false,
	}
	if err := svc.patientRepo.Create(ctx, optedOut); err != nil {
		t.Fatalf("Failed to create opted-out patient: %v", err)
	}

	// Unregistered provider
	if _, err := svc.SendNotification(token, "pat_notif_1", "", "mock_email", "Reminder", "See you soon"); err == nil {
		t.Errorf("Expected error for unregistered provider")
	}

	emailProvider := &dummyNotificationProvider{name: "mock_email", channel: domain.NotificationChannelEmail}
	RegisterNotificationProvider(svc, emailProvider)

	if got := svc.ListProviders(); len(got) != 1 || got[0] != "mock_email" {
		t.Errorf("Expected ListProviders to return [mock_email], got %v", got)
	}

	// Opted-out patient must be refused.
	if _, err := svc.SendNotification(token, "pat_notif_2", "", "mock_email", "Reminder", "See you soon"); err == nil {
		t.Errorf("Expected error sending to a patient who has not opted in to reminders")
	}

	entry, err := svc.SendNotification(token, "pat_notif_1", "", "mock_email", "Reminder", "See you soon")
	if err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}
	if entry.Status != domain.NotificationStatusSent || entry.Recipient != "jane@example.com" || entry.ExternalMessageID != "ext_1" {
		t.Errorf("Unexpected notification log entry: %+v", entry)
	}

	// Provider failure should still be logged, with a failed status, and returned as an error.
	failingProvider := &dummyNotificationProvider{name: "mock_sms", channel: domain.NotificationChannelSMS, sendErr: errors.New("carrier rejected message")}
	RegisterNotificationProvider(svc, failingProvider)

	failedEntry, err := svc.SendNotification(token, "pat_notif_1", "", "mock_sms", "", "Reminder: appointment tomorrow")
	if err == nil {
		t.Fatalf("Expected error from failing provider")
	}
	if failedEntry == nil || failedEntry.Status != domain.NotificationStatusFailed {
		t.Errorf("Expected failed notification log entry to be recorded, got %+v", failedEntry)
	}

	// A provider reporting failure without an error must still surface as an error.
	softFailProvider := &dummyNotificationProvider{name: "mock_soft_fail", channel: domain.NotificationChannelEmail, sendStatus: domain.NotificationStatusFailed}
	RegisterNotificationProvider(svc, softFailProvider)

	softFailed, err := svc.SendNotification(token, "pat_notif_1", "", "mock_soft_fail", "Reminder", "See you soon")
	if err == nil {
		t.Fatalf("Expected error when provider reports failed status")
	}
	if softFailed == nil || softFailed.Status != domain.NotificationStatusFailed || softFailed.ErrorMessage == "" {
		t.Errorf("Expected failed notification log entry with error message, got %+v", softFailed)
	}

	list, err := svc.ListNotificationLog(token, "pat_notif_1", 0, 0)
	if err != nil {
		t.Fatalf("Failed to list notification log: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("Expected 3 notification log entries for patient, got %d", len(list))
	}

	// Unauthorized access.
	if _, err := svc.SendNotification("bad_token", "pat_notif_1", "", "mock_email", "Reminder", "See you soon"); err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized for invalid token, got: %v", err)
	}
	if _, err := svc.ListNotificationLog("bad_token", "", 0, 0); err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized for invalid token, got: %v", err)
	}
	if _, err := svc.ListNotificationLogForAppointment("bad_token", "appt_1"); err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized for invalid token, got: %v", err)
	}
}

func TestNotificationService_AppointmentMustBelongToPatient(t *testing.T) {
	svc, _, token := newTestNotificationService(t)
	ctx := context.Background()

	for _, id := range []string{"pat_appt_owner", "pat_other"} {
		if err := svc.patientRepo.Create(ctx, &domain.Patient{
			ID: id, FirstName: "Test", LastName: id, Email: id + "@example.com", ReminderOptIn: true,
		}); err != nil {
			t.Fatalf("Failed to create patient %s: %v", id, err)
		}
	}

	if err := svc.appointmentRepo.Create(ctx, &domain.Appointment{
		ID:          "appt_owned",
		PatientID:   "pat_appt_owner",
		ProviderID:  "prov_1",
		OperatoryID: "op_1",
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(time.Hour),
		Status:      domain.AppointmentStatusScheduled,
	}); err != nil {
		t.Fatalf("Failed to create appointment: %v", err)
	}

	RegisterNotificationProvider(svc, &dummyNotificationProvider{name: "mock_email", channel: domain.NotificationChannelEmail})

	if _, err := svc.SendNotification(token, "pat_other", "appt_owned", "mock_email", "Reminder", "See you soon"); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput when appointment belongs to another patient, got: %v", err)
	}
	if list, _ := svc.ListNotificationLogForAppointment(token, "appt_owned"); len(list) != 0 {
		t.Errorf("Expected no notification recorded for mismatched appointment, got %d", len(list))
	}

	if _, err := svc.SendNotification(token, "pat_appt_owner", "appt_owned", "mock_email", "Reminder", "See you soon"); err != nil {
		t.Fatalf("Failed to send notification for owned appointment: %v", err)
	}
	if list, _ := svc.ListNotificationLogForAppointment(token, "appt_owned"); len(list) != 1 {
		t.Errorf("Expected 1 notification recorded for appointment, got %d", len(list))
	}
}

func TestNotificationService_ProviderConfigRequiresSession(t *testing.T) {
	svc, _, token := newTestNotificationService(t)

	if err := svc.SetProviderConfig("bad_token", "mock_email", map[string]string{"api_key": "secret"}); err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized setting config with invalid token, got: %v", err)
	}
	if _, err := svc.GetProviderConfig("bad_token", "mock_email"); err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized getting config with invalid token, got: %v", err)
	}

	if err := svc.SetProviderConfig(token, "mock_email", map[string]string{"api_key": "secret"}); err != nil {
		t.Fatalf("Failed to set provider config: %v", err)
	}
	cfg, err := svc.GetProviderConfig(token, "mock_email")
	if err != nil {
		t.Fatalf("Failed to get provider config: %v", err)
	}
	if cfg["api_key"] == "" || cfg["api_key"] == "secret" {
		t.Errorf("Expected redacted api_key, got %q", cfg["api_key"])
	}
}
