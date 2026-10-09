package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// ReminderService exposes automatic appointment reminder settings to the Wails frontend and
// owns the background job that sends them. Reminders are off until staff turn them on.
type ReminderService struct {
	reminders     storage.ReminderRepository
	patients      storage.PatientRepository
	practice      storage.PracticeConfigRepository
	notifications *NotificationService
	auditService  *AuditService
	scheduler     *reminderScheduler

	jobMu     sync.Mutex
	jobCancel context.CancelFunc
	jobDone   chan struct{}
}

func NewReminderService(
	reminders storage.ReminderRepository,
	patients storage.PatientRepository,
	appointments storage.AppointmentRepository,
	practice storage.PracticeConfigRepository,
	logs storage.NotificationLogRepository,
	notifications *NotificationService,
	auditService *AuditService,
) *ReminderService {
	return &ReminderService{
		reminders:     reminders,
		patients:      patients,
		practice:      practice,
		notifications: notifications,
		auditService:  auditService,
		scheduler: &reminderScheduler{
			notifications: notifications,
			appointments:  appointments,
			patients:      patients,
			practice:      practice,
			reminders:     reminders,
			logs:          logs,
			audit:         auditService,
			now:           time.Now,
		},
	}
}

// ─── Job lifecycle ───────────────────────────────────────────────────────────

// StartReminderJob runs the reminder job until ctx is cancelled or StopReminderJob is called.
// Exposed as a function rather than a method so Wails does not bind it.
func StartReminderJob(s *ReminderService, ctx context.Context) {
	s.startJob(ctx, reminderPassInterval)
}

// StopReminderJob stops the reminder job and waits for a pass in progress to finish.
func StopReminderJob(s *ReminderService) {
	s.stopJob()
}

func (s *ReminderService) startJob(ctx context.Context, interval time.Duration) {
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	if s.jobCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.jobCancel, s.jobDone = cancel, done
	s.scheduler.setRunning(true)
	go func() {
		defer close(done)
		defer s.scheduler.setRunning(false)
		s.scheduler.run(ctx, interval)
	}()
}

func (s *ReminderService) stopJob() {
	s.jobMu.Lock()
	cancel, done := s.jobCancel, s.jobDone
	s.jobCancel, s.jobDone = nil, nil
	s.jobMu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

// ─── Settings ────────────────────────────────────────────────────────────────

func (s *ReminderService) requireSession(token string) (*domain.Provider, error) {
	user := s.auditService.GetSessionUser(token)
	if user == nil {
		return nil, ErrUnauthorized
	}
	return user, nil
}

func (s *ReminderService) settings(ctx context.Context) (*domain.ReminderSettings, error) {
	settings, err := s.reminders.GetSettings(ctx)
	if errors.Is(err, storage.ErrNotFound) {
		defaults := domain.DefaultReminderSettings()
		return &defaults, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reminder settings: %w", err)
	}
	return settings, nil
}

// GetReminderSettings returns whether reminders are on and the sending hours.
func (s *ReminderService) GetReminderSettings(token string) (*domain.ReminderSettings, error) {
	if _, err := s.requireSession(token); err != nil {
		return nil, err
	}
	return s.settings(context.Background())
}

// SaveSendingHours sets the daily window ("HH:MM", practice timezone) for sending reminders.
func (s *ReminderService) SaveSendingHours(token string, start string, end string) (*domain.ReminderSettings, error) {
	if _, err := s.requireSession(token); err != nil {
		return nil, err
	}
	if _, err := parseSendingHours(start, end); err != nil {
		return nil, err
	}
	ctx := context.Background()
	settings, err := s.settings(ctx)
	if err != nil {
		return nil, err
	}
	settings.SendingHoursStart, settings.SendingHoursEnd = strings.TrimSpace(start), strings.TrimSpace(end)
	if err := s.reminders.SaveSettings(ctx, settings); err != nil {
		return nil, err
	}
	if err := s.auditService.LogAction(token, domain.AuditActionUpdate, "reminder_settings",
		fmt.Sprintf("Set reminder sending hours to %s-%s", settings.SendingHoursStart, settings.SendingHoursEnd)); err != nil {
		return settings, fmt.Errorf("sending hours saved but failed to log audit: %w", err)
	}
	return settings, nil
}

// GetRecipientCounts returns how many active patients are opted in to reminders, and how many
// of them can be reached by text and by email. It's shown before reminders are turned on.
func (s *ReminderService) GetRecipientCounts(token string) (*domain.ReminderRecipientCounts, error) {
	if _, err := s.requireSession(token); err != nil {
		return nil, err
	}
	counts, err := s.recipientCounts(context.Background())
	if err != nil {
		return nil, err
	}
	_ = s.auditService.LogAction(token, domain.AuditActionRead, "reminder_settings", "Counted patients opted in to reminders")
	return counts, nil
}

func (s *ReminderService) recipientCounts(ctx context.Context) (*domain.ReminderRecipientCounts, error) {
	patients, _, err := s.patients.List(ctx, domain.PatientFilter{Status: string(domain.StatusActive)})
	if err != nil {
		return nil, fmt.Errorf("failed to list patients: %w", err)
	}
	var country domain.CountryCode
	if cfg, err := s.practice.Get(ctx); err == nil {
		country = cfg.CountryCode
	}
	counts := &domain.ReminderRecipientCounts{}
	for _, p := range patients {
		if !p.ReminderOptIn {
			continue
		}
		counts.OptedIn++
		if p.PhonePrimary != "" {
			if _, err := toSMSNumber(p.PhonePrimary, country); err == nil {
				counts.WithMobile++
			}
		}
		if _, err := mail.ParseAddress(strings.TrimSpace(p.Email)); err == nil {
			counts.WithEmail++
		}
	}
	return counts, nil
}

// EnableReminders turns automatic reminders on. The first time, it creates defaultRules (the
// frontend supplies their text, so it's translated); a rule with no provider gets the
// channel's registered provider. The practice timezone must be set first, so reminders never
// show a guessed time.
func (s *ReminderService) EnableReminders(token string, defaultRules []domain.ReminderRule) (*domain.ReminderSettings, error) {
	user, err := s.requireSession(token)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	practice, err := s.practice.Get(ctx)
	if err != nil || practice.Timezone == "" {
		return nil, fmt.Errorf("%w: set the practice timezone before turning on reminders", storage.ErrInvalidInput)
	}

	existing, err := s.reminders.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		if err := s.createRules(ctx, defaultRules); err != nil {
			return nil, err
		}
	}

	counts, err := s.recipientCounts(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := s.settings(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	settings.Enabled, settings.EnabledAt, settings.EnabledBy = true, &now, user.ID
	if err := s.reminders.SaveSettings(ctx, settings); err != nil {
		return nil, err
	}
	if err := s.auditService.LogAction(token, domain.AuditActionUpdate, "reminder_settings", fmt.Sprintf(
		"Turned on automatic reminders for %d opted-in patients (%d with a mobile number, %d with an email address)",
		counts.OptedIn, counts.WithMobile, counts.WithEmail)); err != nil {
		return settings, fmt.Errorf("reminders turned on but failed to log audit: %w", err)
	}
	return settings, nil
}

func (s *ReminderService) createRules(ctx context.Context, rules []domain.ReminderRule) error {
	if len(rules) == 0 {
		return fmt.Errorf("%w: no reminder rules to create", storage.ErrInvalidInput)
	}
	// Validate every rule before saving any: a partly created set would stop the defaults from
	// ever being created, since rules are only created when none exist.
	seen := map[string]bool{}
	prepared := make([]domain.ReminderRule, 0, len(rules))
	for _, rule := range rules {
		if rule.OffsetMinutes <= 0 {
			return fmt.Errorf("%w: reminder timing must be before the appointment", storage.ErrInvalidInput)
		}
		key := fmt.Sprintf("%d/%s", rule.OffsetMinutes, rule.Channel)
		if seen[key] {
			return fmt.Errorf("%w: duplicate reminder rule for %s", storage.ErrInvalidInput, key)
		}
		seen[key] = true
		if rule.ProviderName == "" {
			if names := s.notifications.providersFor(rule.Channel); len(names) > 0 {
				rule.ProviderName = names[0]
			}
		}
		rule.ID = "rule_" + uuid.NewString()
		rule.CreatedAt = time.Time{}
		if err := s.validateRule(&rule); err != nil {
			return err
		}
		prepared = append(prepared, rule)
	}
	for i := range prepared {
		if err := s.reminders.SaveRule(ctx, &prepared[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReminderService) validateRule(rule *domain.ReminderRule) error {
	if rule.Channel != domain.NotificationChannelSMS && rule.Channel != domain.NotificationChannelEmail {
		return fmt.Errorf("%w: reminders can only be sent by text or email", storage.ErrInvalidInput)
	}
	if _, ok := s.notifications.providerFor(rule.ProviderName, rule.Channel); !ok {
		return fmt.Errorf("%w: %q is not a registered %s provider", storage.ErrInvalidInput, rule.ProviderName, rule.Channel)
	}
	if strings.TrimSpace(rule.BodyTemplate) == "" {
		return fmt.Errorf("%w: reminder message is empty", storage.ErrInvalidInput)
	}
	if rule.Channel == domain.NotificationChannelEmail && strings.TrimSpace(rule.SubjectTemplate) == "" {
		return fmt.Errorf("%w: reminder email subject is empty", storage.ErrInvalidInput)
	}
	if err := validateReminderTemplate(rule.SubjectTemplate); err != nil {
		return err
	}
	return validateReminderTemplate(rule.BodyTemplate)
}

// DisableReminders turns automatic reminders off. Rules and their text are kept.
func (s *ReminderService) DisableReminders(token string) (*domain.ReminderSettings, error) {
	if _, err := s.requireSession(token); err != nil {
		return nil, err
	}
	ctx := context.Background()
	settings, err := s.settings(ctx)
	if err != nil {
		return nil, err
	}
	settings.Enabled, settings.EnabledAt, settings.EnabledBy = false, nil, ""
	if err := s.reminders.SaveSettings(ctx, settings); err != nil {
		return nil, err
	}
	if err := s.auditService.LogAction(token, domain.AuditActionUpdate, "reminder_settings", "Turned off automatic reminders"); err != nil {
		return settings, fmt.Errorf("reminders turned off but failed to log audit: %w", err)
	}
	return settings, nil
}

// ─── Rules ───────────────────────────────────────────────────────────────────

// ListReminderRules returns the reminder rules, longest lead time first.
func (s *ReminderService) ListReminderRules(token string) ([]*domain.ReminderRule, error) {
	if _, err := s.requireSession(token); err != nil {
		return nil, err
	}
	return s.reminders.ListRules(context.Background())
}

// SaveReminderRule updates an existing rule's provider, text, and whether it's on. Its timing
// and channel can't change.
func (s *ReminderService) SaveReminderRule(token string, rule domain.ReminderRule) (*domain.ReminderRule, error) {
	if _, err := s.requireSession(token); err != nil {
		return nil, err
	}
	ctx := context.Background()
	rules, err := s.reminders.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	var existing *domain.ReminderRule
	for _, r := range rules {
		if r.ID == rule.ID {
			existing = r
		}
	}
	if existing == nil {
		return nil, fmt.Errorf("%w: reminder rule %q not found", storage.ErrNotFound, rule.ID)
	}
	existing.ProviderName = rule.ProviderName
	existing.SubjectTemplate = rule.SubjectTemplate
	existing.BodyTemplate = rule.BodyTemplate
	existing.Enabled = rule.Enabled
	if err := s.validateRule(existing); err != nil {
		return nil, err
	}
	if err := s.reminders.SaveRule(ctx, existing); err != nil {
		return nil, err
	}
	state := "off"
	if existing.Enabled {
		state = "on"
	}
	if err := s.auditService.LogAction(token, domain.AuditActionUpdate, "reminder_settings", fmt.Sprintf(
		"Updated the %s %s reminder (now %s)", describeOffset(existing.OffsetMinutes), existing.Channel, state)); err != nil {
		return existing, fmt.Errorf("reminder rule saved but failed to log audit: %w", err)
	}
	return existing, nil
}

// ListReminderProviders returns the registered provider names for each reminder channel.
func (s *ReminderService) ListReminderProviders(token string) (map[string][]string, error) {
	if _, err := s.requireSession(token); err != nil {
		return nil, err
	}
	return map[string][]string{
		string(domain.NotificationChannelSMS):   s.notifications.providersFor(domain.NotificationChannelSMS),
		string(domain.NotificationChannelEmail): s.notifications.providersFor(domain.NotificationChannelEmail),
	}, nil
}

// PreviewReminderTemplate renders templates with sample values and reports the text length,
// for the template editor. Text messages over 160 characters are flagged; they would be
// skipped rather than sent.
func (s *ReminderService) PreviewReminderTemplate(token string, channel string, subject string, body string) (*domain.ReminderPreview, error) {
	if _, err := s.requireSession(token); err != nil {
		return nil, err
	}
	practice, err := s.practice.Get(context.Background())
	if err != nil {
		practice = &domain.PracticeConfig{}
	}
	return previewReminder(domain.NotificationChannel(channel), subject, body, practice)
}

// GetReminderJobStatus reports the reminder job's most recent pass in this process.
func (s *ReminderService) GetReminderJobStatus(token string) (*domain.ReminderJobStatus, error) {
	if _, err := s.requireSession(token); err != nil {
		return nil, err
	}
	status := s.scheduler.jobStatus()
	return &status, nil
}
