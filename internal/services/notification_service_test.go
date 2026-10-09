package services

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
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
	sent       []*domain.NotificationMessage
}

func (p *dummyNotificationProvider) Name() string                        { return p.name }
func (p *dummyNotificationProvider) Channel() domain.NotificationChannel { return p.channel }
func (p *dummyNotificationProvider) Send(ctx context.Context, msg *domain.NotificationMessage, config map[string]string) (*domain.NotificationResult, error) {
	p.sent = append(p.sent, msg)
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
	notificationSvc := NewNotificationService(patientRepo, appointmentRepo, configRepo, logRepo, secretsSvc, auditSvc)

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
		PhonePrimary:  "+1 (202) 555-0123",
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
	if failedEntry != nil && failedEntry.Recipient != "+12025550123" {
		t.Errorf("Expected the phone number normalized to E.164, got %q", failedEntry.Recipient)
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
	list, err := svc.ListNotificationLogForAppointment(token, "appt_owned")
	if err != nil {
		t.Fatalf("Failed to list notification log for appointment: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Expected no notification recorded for mismatched appointment, got %d", len(list))
	}

	if _, err := svc.SendNotification(token, "pat_appt_owner", "appt_owned", "mock_email", "Reminder", "See you soon"); err != nil {
		t.Fatalf("Failed to send notification for owned appointment: %v", err)
	}
	list, err = svc.ListNotificationLogForAppointment(token, "appt_owned")
	if err != nil {
		t.Fatalf("Failed to list notification log for appointment: %v", err)
	}
	if len(list) != 1 {
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

func TestNotificationService_SendTestMessage(t *testing.T) {
	svc, auditSvc, token := newTestNotificationService(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := svc.practiceRepo.Save(ctx, &domain.PracticeConfig{
		ID: 1, ClinicName: "Smile Dental", CountryCode: domain.CountryUS, Currency: "USD",
		ToothSystem: domain.ToothSystemUniversal, DateFormat: "YYYY-MM-DD", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("Failed to save practice config: %v", err)
	}

	email := &dummyNotificationProvider{name: "mock_email", channel: domain.NotificationChannelEmail}
	sms := &dummyNotificationProvider{name: "mock_sms", channel: domain.NotificationChannelSMS}
	failing := &dummyNotificationProvider{name: "mock_failing", channel: domain.NotificationChannelEmail, sendErr: errors.New("server said no")}
	for _, p := range []*dummyNotificationProvider{email, sms, failing} {
		RegisterNotificationProvider(svc, p)
	}

	if _, err := svc.SendTestMessage("bad_token", "mock_email", "me@example.com", "Test", "Hello"); err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got %v", err)
	}
	if _, err := svc.SendTestMessage(token, "nope", "me@example.com", "Test", "Hello"); err == nil {
		t.Errorf("Expected an unregistered provider to be rejected")
	}
	for _, args := range [][2]string{{" ", "Hello"}, {"me@example.com", ""}} {
		if _, err := svc.SendTestMessage(token, "mock_email", args[0], "Test", args[1]); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected to=%q body=%q to be rejected, got %v", args[0], args[1], err)
		}
	}

	result, err := svc.SendTestMessage(token, "mock_email", " me@example.com ", "Test", "Hello")
	if err != nil || result.Status != domain.NotificationStatusSent {
		t.Fatalf("Test email failed: %+v, %v", result, err)
	}
	if got := email.sent[len(email.sent)-1]; got.To != "me@example.com" || got.Subject != "Test" || got.Body != "Hello" {
		t.Errorf("Unexpected message passed to provider: %+v", got)
	}

	// Phone numbers are read in the practice's country and normalized before sending.
	if _, err := svc.SendTestMessage(token, "mock_sms", "(202) 555-0123", "", "Hello"); err != nil {
		t.Fatalf("Test SMS failed: %v", err)
	}
	if got := sms.sent[len(sms.sent)-1].To; got != "+12025550123" {
		t.Errorf("Expected normalized number +12025550123, got %q", got)
	}
	if _, err := svc.SendTestMessage(token, "mock_sms", "555-0123", "", "Hello"); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected an invalid number to be rejected, got %v", err)
	}
	if len(sms.sent) != 1 {
		t.Errorf("Expected the invalid number never to reach the provider, got %d sends", len(sms.sent))
	}

	if _, err := svc.SendTestMessage(token, "mock_failing", "me@example.com", "Test", "Hello"); err == nil || !strings.Contains(err.Error(), "server said no") {
		t.Errorf("Expected the provider's error to be returned, got %v", err)
	}

	logs, err := auditSvc.GetAuditLogs(token, "", 50, 0)
	if err != nil {
		t.Fatalf("Failed to read audit log: %v", err)
	}
	var sent, failed int
	for _, l := range logs {
		if l.Resource != "notification_test" {
			continue
		}
		if l.UserID != "prov_1" || l.PatientID != "" {
			t.Errorf("Expected test sends attributed to the staff member with no patient, got %+v", l)
		}
		switch {
		case strings.HasPrefix(l.Details, "Sent test"):
			sent++
		case strings.HasPrefix(l.Details, "Failed to send test") && strings.Contains(l.Details, "server said no"):
			failed++
		}
	}
	if sent != 2 || failed != 1 {
		t.Errorf("Expected 2 successful and 1 failed test send audited, got %d and %d", sent, failed)
	}

	// Test sends aren't tied to a patient, so they leave no delivery history.
	if entries, err := svc.ListNotificationLog(token, "", 50, 0); err != nil || len(entries) != 0 {
		t.Errorf("Expected no notification log entries, got %d, %v", len(entries), err)
	}
}

// failingPracticeRepo returns err from Get; NotificationService uses no other method.
type failingPracticeRepo struct {
	storage.PracticeConfigRepository
	err error
}

func (r failingPracticeRepo) Get(context.Context) (*domain.PracticeConfig, error) {
	return nil, r.err
}

// The practice's country is only needed to read phone numbers. A storage failure must be
// reported as such, not as an unreadable number; a practice that isn't set up yet just
// can't use national-format numbers.
func TestNotificationService_PracticeCountryErrors(t *testing.T) {
	svc, _, token := newTestNotificationService(t)
	sms := &dummyNotificationProvider{name: "mock_sms", channel: domain.NotificationChannelSMS}
	email := &dummyNotificationProvider{name: "mock_email", channel: domain.NotificationChannelEmail}
	RegisterNotificationProvider(svc, sms)
	RegisterNotificationProvider(svc, email)
	ctx := context.Background()
	if err := svc.patientRepo.Create(ctx, &domain.Patient{
		ID: "pat_country", FirstName: "Jane", LastName: "Doe", Email: "jane@example.com",
		PhonePrimary: "(202) 555-0123", ReminderOptIn: true,
	}); err != nil {
		t.Fatal(err)
	}

	svc.practiceRepo = failingPracticeRepo{err: errors.New("disk I/O error")}
	for name, send := range map[string]func() error{
		"test send": func() error {
			_, err := svc.SendTestMessage(token, "mock_sms", "(202) 555-0123", "", "Hello")
			return err
		},
		"patient send": func() error {
			_, err := svc.SendNotification(token, "pat_country", "", "mock_sms", "", "Hello")
			return err
		},
	} {
		err := send()
		if err == nil || !strings.Contains(err.Error(), "disk I/O error") || errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("%s: expected the storage error, got %v", name, err)
		}
	}
	if len(sms.sent) != 0 {
		t.Errorf("Expected nothing sent while practice settings can't be read, got %d", len(sms.sent))
	}
	if _, err := svc.SendTestMessage(token, "mock_email", "me@example.com", "Test", "Hello"); err != nil {
		t.Errorf("Email doesn't need the practice's country, but failed: %v", err)
	}

	// Not set up yet: only numbers with a country code can be read.
	svc.practiceRepo = failingPracticeRepo{err: storage.ErrNotFound}
	if _, err := svc.SendTestMessage(token, "mock_sms", "(202) 555-0123", "", "Hello"); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected a national number to be unreadable without a practice country, got %v", err)
	}
	if _, err := svc.SendTestMessage(token, "mock_sms", "+1 202 555 0123", "", "Hello"); err != nil {
		t.Errorf("Expected an international number to work without a practice country, got %v", err)
	}
}
