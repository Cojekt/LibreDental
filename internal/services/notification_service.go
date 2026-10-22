package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services/notify"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/google/uuid"
)

// NotificationService exposes patient notification (email/SMS/voice) operations to the
// Wails frontend. It follows the same provider-registry pattern as BillingService's
// insurance clearinghouse integrations: vendors are registered as domain.NotificationProvider
// implementations, and their credentials live in the OS keychain via SecretsService, never
// in the SQLite database.
type NotificationService struct {
	patientRepo     storage.PatientRepository
	appointmentRepo storage.AppointmentRepository
	practiceRepo    storage.PracticeConfigRepository
	logRepo         storage.NotificationLogRepository
	secrets         *SecretsService
	auditService    *AuditService
	providers       map[string]domain.NotificationProvider
}

func NewNotificationService(
	patientRepo storage.PatientRepository,
	appointmentRepo storage.AppointmentRepository,
	practiceRepo storage.PracticeConfigRepository,
	logRepo storage.NotificationLogRepository,
	secrets *SecretsService,
	auditService *AuditService,
) *NotificationService {
	return &NotificationService{
		patientRepo:     patientRepo,
		appointmentRepo: appointmentRepo,
		practiceRepo:    practiceRepo,
		logRepo:         logRepo,
		secrets:         secrets,
		auditService:    auditService,
		providers:       make(map[string]domain.NotificationProvider),
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

// GetProviderConfig retrieves configuration (secrets redacted) for a specific provider.
func (s *NotificationService) GetProviderConfig(token string, providerName string) (map[string]string, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	if providerName == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	return s.secrets.GetProviderConfig(providerName)
}

// SetProviderConfig saves configuration for a specific provider.
func (s *NotificationService) SetProviderConfig(token string, providerName string, config map[string]string) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}
	if providerName == "" {
		return fmt.Errorf("provider name is required")
	}
	if err := s.secrets.SetProviderConfig(providerName, config); err != nil {
		return err
	}
	if err := s.auditService.LogAction(token, domain.AuditActionUpdate, "notification_provider_config",
		"Updated configuration for notification provider "+providerName); err != nil {
		return fmt.Errorf("provider config saved but failed to log audit: %w", err)
	}
	return nil
}

// ─── Sending ─────────────────────────────────────────────────────────────────

// recipientFor resolves the destination address/number for a patient and channel,
// based on the patient's stored contact details. Phone numbers are stored as typed, so they
// are normalized here; country is the practice's, for numbers without a country code.
func recipientFor(patient *domain.Patient, channel domain.NotificationChannel, country domain.CountryCode) (string, error) {
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
		return normalizePhoneFor(channel, patient.PhonePrimary, country)
	default:
		return "", fmt.Errorf("unsupported notification channel: %s", channel)
	}
}

func normalizePhoneFor(channel domain.NotificationChannel, phone string, country domain.CountryCode) (string, error) {
	if channel == domain.NotificationChannelSMS {
		return notify.ToSMSNumber(phone, country)
	}
	return notify.ToE164(phone, country)
}

// practiceCountry returns the practice's country, used to read phone numbers that have no
// country code. Before onboarding sets it, it's empty and only numbers with a country code
// can be used. Any other error is returned, so a storage failure isn't mistaken for an
// unreadable phone number.
func (s *NotificationService) practiceCountry(ctx context.Context) (domain.CountryCode, error) {
	cfg, err := s.practiceRepo.Get(ctx)
	if errors.Is(err, storage.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read practice settings: %w", err)
	}
	return cfg.CountryCode, nil
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

	if appointmentID != "" {
		appt, err := s.appointmentRepo.GetByID(ctx, appointmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get appointment for notification: %w", err)
		}
		if appt.PatientID != patientID {
			return nil, fmt.Errorf("%w: appointment does not belong to patient", storage.ErrInvalidInput)
		}
	}

	var country domain.CountryCode
	if provider.Channel() != domain.NotificationChannelEmail {
		if country, err = s.practiceCountry(ctx); err != nil {
			return nil, err
		}
	}
	recipient, err := recipientFor(patient, provider.Channel(), country)
	if err != nil {
		return nil, err
	}

	config, err := s.providerConfig(providerName)
	if err != nil {
		return nil, err
	}

	result, sendErr := s.send(ctx, provider, &domain.NotificationMessage{
		Channel: provider.Channel(),
		To:      recipient,
		Subject: subject,
		Body:    body,
	}, config)

	entry := &domain.NotificationLog{
		ID:            newNotificationID(),
		PatientID:     patientID,
		AppointmentID: appointmentID,
		Channel:       provider.Channel(),
		ProviderName:  providerName,
		Recipient:     recipient,
		Subject:       subject,
		Body:          body,
		SentAt:        time.Now().UTC(),
	}
	if result != nil {
		entry.ExternalMessageID = result.ExternalMessageID
	}
	entry.Status = deliveryStatus(sendErr)
	if sendErr != nil {
		entry.ErrorMessage = sendErr.Error()
	} else if result != nil && result.Status != "" {
		entry.Status = result.Status
	}

	// The provider call has already happened at this point, so a log failure must not hide
	// that from the caller: the entry is still returned and the audit trail still records the
	// attempt, so a retry does not silently send the patient a duplicate.
	logErr := s.logRepo.Create(ctx, entry)

	detail := fmt.Sprintf("Sent %s notification via %s", provider.Channel(), providerName)
	if sendErr != nil {
		detail = fmt.Sprintf("Failed to send %s notification via %s: %v", provider.Channel(), providerName, sendErr)
	}
	if logErr != nil {
		detail += " (notification log entry not recorded)"
	}
	auditErr := s.auditService.LogPatientAction(token, domain.AuditActionCreate, patientID, "notification", detail)

	switch {
	case sendErr != nil:
		return entry, fmt.Errorf("provider %q failed to send notification: %w", providerName, sendErr)
	case logErr != nil:
		return entry, fmt.Errorf("notification sent but failed to record notification log entry: %w", logErr)
	case auditErr != nil:
		return entry, fmt.Errorf("notification sent but failed to log audit: %w", auditErr)
	}
	return entry, nil
}

// SendTestMessage sends a message through a provider to an address or number staff type in,
// to check the provider's saved settings. It isn't tied to a patient, so it writes no
// notification log entry, but every attempt is audited.
func (s *NotificationService) SendTestMessage(token string, providerName string, to string, subject string, body string) (*domain.NotificationResult, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", providerName)
	}
	to = strings.TrimSpace(to)
	if to == "" {
		return nil, fmt.Errorf("%w: recipient is required", storage.ErrInvalidInput)
	}
	if body == "" {
		return nil, fmt.Errorf("%w: message body is required", storage.ErrInvalidInput)
	}

	ctx := context.Background()
	if provider.Channel() != domain.NotificationChannelEmail {
		country, err := s.practiceCountry(ctx)
		if err != nil {
			return nil, err
		}
		normalized, err := normalizePhoneFor(provider.Channel(), to, country)
		if err != nil {
			return nil, err
		}
		to = normalized
	}

	config, err := s.providerConfig(providerName)
	if err != nil {
		return nil, err
	}

	result, sendErr := s.send(ctx, provider, &domain.NotificationMessage{
		Channel: provider.Channel(),
		To:      to,
		Subject: subject,
		Body:    body,
	}, config)

	detail := fmt.Sprintf("Sent test %s message via %s to %s", provider.Channel(), providerName, to)
	if sendErr != nil {
		detail = fmt.Sprintf("Failed to send test %s message via %s to %s: %v", provider.Channel(), providerName, to, sendErr)
	}
	auditErr := s.auditService.LogAction(token, domain.AuditActionCreate, "notification_test", detail)

	switch {
	case sendErr != nil:
		if auditErr != nil {
			return result, fmt.Errorf("provider %q failed to send test message: %w (audit logging also failed: %v)", providerName, sendErr, auditErr)
		}
		return result, fmt.Errorf("provider %q failed to send test message: %w", providerName, sendErr)
	case auditErr != nil:
		return result, fmt.Errorf("test message sent but failed to log audit: %w", auditErr)
	}
	return result, nil
}

// providerConfig returns a provider's saved settings, including secrets, from the keychain.
func (s *NotificationService) providerConfig(providerName string) (map[string]string, error) {
	config, err := s.secrets.getRawProviderConfig(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve config for provider %q: %w", providerName, err)
	}
	return config, nil
}

// send delivers msg through provider, bounded by a timeout. A provider that reports a failed
// status without an error is treated as having failed.
func (s *NotificationService) send(ctx context.Context, provider domain.NotificationProvider, msg *domain.NotificationMessage, config map[string]string) (*domain.NotificationResult, error) {
	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	result, err := provider.Send(sendCtx, msg, config)
	if err == nil && result != nil && result.Status == domain.NotificationStatusFailed {
		err = fmt.Errorf("provider reported failed delivery")
	}
	return result, err
}

// deliveryStatus is the status recorded for a send that returned err.
func deliveryStatus(err error) domain.NotificationStatus {
	switch {
	case err == nil:
		return domain.NotificationStatusSent
	case errors.Is(err, domain.ErrDeliveryUnknown):
		return domain.NotificationStatusUnknown
	default:
		return domain.NotificationStatusFailed
	}
}

// providerFor returns the registered provider with that name and channel.
func (s *NotificationService) providerFor(name string, channel domain.NotificationChannel) (domain.NotificationProvider, bool) {
	p, ok := s.providers[name]
	if !ok || p.Channel() != channel {
		return nil, false
	}
	return p, true
}

// providersFor lists registered provider names for a channel, sorted.
func (s *NotificationService) providersFor(channel domain.NotificationChannel) []string {
	var names []string
	for name, p := range s.providers {
		if p.Channel() == channel {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func newNotificationID() string {
	return "notif_" + uuid.NewString()
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
	_ = s.auditService.LogAction(token, domain.AuditActionRead, "notification", "Viewed notification history for appointment "+appointmentID)
	return entries, nil
}
