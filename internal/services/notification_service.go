package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// NotificationService exposes patient notification (email/SMS/voice) operations to the
// Wails frontend. It follows the same provider-registry pattern as BillingService's
// insurance clearinghouse integrations: vendors are registered as domain.NotificationProvider
// implementations, and their credentials live in the OS keychain via SecretsService, never
// in the SQLite database.
type NotificationService struct {
	patientRepo  storage.PatientRepository
	logRepo      storage.NotificationLogRepository
	secrets      *SecretsService
	auditService *AuditService
	providers    map[string]domain.NotificationProvider
}

func NewNotificationService(
	patientRepo storage.PatientRepository,
	logRepo storage.NotificationLogRepository,
	secrets *SecretsService,
	auditService *AuditService,
) *NotificationService {
	return &NotificationService{
		patientRepo:  patientRepo,
		logRepo:      logRepo,
		secrets:      secrets,
		auditService: auditService,
		providers:    make(map[string]domain.NotificationProvider),
	}
}

// ─── Provider Registry ───────────────────────────────────────────────────────

// RegisterNotificationProvider registers a new notification provider for use.
// Exposed as a function rather than a method so Wails does not bind it.
func RegisterNotificationProvider(s *NotificationService, p domain.NotificationProvider) {
	s.registerProvider(p)
}

func (s *NotificationService) registerProvider(p domain.NotificationProvider) {
	if p != nil {
		s.providers[p.Name()] = p
	}
}

// ListProviders returns a list of registered provider names.
func (s *NotificationService) ListProviders() []string {
	var names []string
	for name := range s.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetProviderConfig retrieves configuration for a specific provider.
func (s *NotificationService) GetProviderConfig(providerName string) (map[string]string, error) {
	if providerName == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	return s.secrets.GetProviderConfig(providerName)
}

// SetProviderConfig saves configuration for a specific provider.
func (s *NotificationService) SetProviderConfig(providerName string, config map[string]string) error {
	if providerName == "" {
		return fmt.Errorf("provider name is required")
	}
	return s.secrets.SetProviderConfig(providerName, config)
}

// ─── Sending ─────────────────────────────────────────────────────────────────

// recipientFor resolves the destination address/number for a patient and channel,
// based on the patient's stored contact details.
func recipientFor(patient *domain.Patient, channel domain.NotificationChannel) (string, error) {
	switch channel {
	case domain.NotificationChannelEmail:
		if patient.Email == "" {
			return "", fmt.Errorf("patient has no email address on file")
		}
		return patient.Email, nil
	case domain.NotificationChannelSMS, domain.NotificationChannelVoice:
		if patient.PhonePrimary == "" {
			return "", fmt.Errorf("patient has no phone number on file")
		}
		return patient.PhonePrimary, nil
	default:
		return "", fmt.Errorf("unsupported notification channel: %s", channel)
	}
}

// SendNotification sends a message to a patient through the given provider, subject to the
// patient's reminder opt-in preference, and records the attempt in both the notification
// log and the HIPAA audit trail regardless of outcome.
func (s *NotificationService) SendNotification(token string, patientID string, appointmentID string, providerName string, subject string, body string) (*domain.NotificationLog, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	if patientID == "" {
		return nil, fmt.Errorf("%w: patient ID is required", storage.ErrInvalidInput)
	}
	if providerName == "" {
		return nil, fmt.Errorf("%w: provider name is required", storage.ErrInvalidInput)
	}
	if body == "" {
		return nil, fmt.Errorf("%w: message body is required", storage.ErrInvalidInput)
	}

	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", providerName)
	}

	ctx := context.Background()
	patient, err := s.patientRepo.GetByID(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient for notification: %w", err)
	}

	if !patient.ReminderOptIn {
		return nil, fmt.Errorf("patient has not opted in to reminder notifications")
	}

	recipient, err := recipientFor(patient, provider.Channel())
	if err != nil {
		return nil, err
	}

	config, err := s.secrets.getRawProviderConfig(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve config for provider %q: %w", providerName, err)
	}

	msg := &domain.NotificationMessage{
		Channel: provider.Channel(),
		To:      recipient,
		Subject: subject,
		Body:    body,
	}

	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, sendErr := provider.Send(sendCtx, msg, config)

	entry := &domain.NotificationLog{
		ID:            fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		PatientID:     patientID,
		AppointmentID: appointmentID,
		Channel:       provider.Channel(),
		ProviderName:  providerName,
		Recipient:     recipient,
		Subject:       subject,
		Body:          body,
		SentAt:        time.Now().UTC(),
	}
	if sendErr != nil {
		entry.Status = domain.NotificationStatusFailed
		entry.ErrorMessage = sendErr.Error()
	} else {
		entry.Status = domain.NotificationStatusSent
		if result != nil && result.Status != "" {
			entry.Status = result.Status
		}
	}

	if err := s.logRepo.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to record notification log entry: %w", err)
	}

	_ = s.auditService.LogPatientAction(token, domain.AuditActionCreate, patientID, "notification",
		fmt.Sprintf("Sent %s notification via %s", provider.Channel(), providerName))

	if sendErr != nil {
		return entry, fmt.Errorf("provider %q failed to send notification: %w", providerName, sendErr)
	}
	return entry, nil
}

// ListNotificationLog returns notification history, optionally filtered by patient.
func (s *NotificationService) ListNotificationLog(token string, patientID string, limit, offset int) ([]*domain.NotificationLog, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	entries, err := s.logRepo.List(context.Background(), patientID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list notification log: %w", err)
	}
	if patientID != "" {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, patientID, "notification", "Viewed notification history")
	} else {
		_ = s.auditService.LogAction(token, domain.AuditActionRead, "notification", "Viewed all notification history")
	}
	return entries, nil
}

// ListNotificationLogForAppointment returns notification history for a single appointment.
func (s *NotificationService) ListNotificationLogForAppointment(token string, appointmentID string) ([]*domain.NotificationLog, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	if appointmentID == "" {
		return nil, fmt.Errorf("%w: appointment ID is required", storage.ErrInvalidInput)
	}

	entries, err := s.logRepo.ListByAppointment(context.Background(), appointmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list notification log for appointment: %w", err)
	}
	return entries, nil
}
