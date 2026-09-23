package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

type dummyNotificationProvider struct {
	name    string
	channel domain.NotificationChannel
	sendErr error
}

func (p *dummyNotificationProvider) Name() string                        { return p.name }
func (p *dummyNotificationProvider) Channel() domain.NotificationChannel { return p.channel }
func (p *dummyNotificationProvider) Send(ctx context.Context, msg *domain.NotificationMessage, config map[string]string) (*domain.NotificationResult, error) {
	if p.sendErr != nil {
		return nil, p.sendErr
	}
	return &domain.NotificationResult{ExternalMessageID: "ext_1", Status: domain.NotificationStatusSent}, nil
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

	patientRepo := sqlite.NewPatientRepository(db)
	logRepo := sqlite.NewNotificationRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	configRepo := sqlite.NewPracticeConfigRepository(db)

	ctx := context.Background()
	if err := configRepo.SaveProvider(ctx, &domain.Provider{ID: "prov_1", Name: "Test Prov", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}

	auditSvc := NewAuditService(auditRepo, configRepo)
	token, err := auditSvc.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	secretsSvc := NewSecretsService()
	notificationSvc := NewNotificationService(patientRepo, logRepo, secretsSvc, auditSvc)

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
	if entry.Status != domain.NotificationStatusSent || entry.Recipient != "jane@example.com" {
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

	list, err := svc.ListNotificationLog(token, "pat_notif_1", 0, 0)
	if err != nil {
		t.Fatalf("Failed to list notification log: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Expected 2 notification log entries for patient, got %d", len(list))
	}

	// Unauthorized access.
	if _, err := svc.SendNotification("bad_token", "pat_notif_1", "", "mock_email", "Reminder", "See you soon"); err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized for invalid token, got: %v", err)
	}
	if _, err := svc.ListNotificationLog("bad_token", "", 0, 0); err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized for invalid token, got: %v", err)
	}
}
