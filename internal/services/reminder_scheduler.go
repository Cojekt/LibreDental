package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

const (
	reminderPassInterval = 5 * time.Minute
	// A send is bounded by a 30-second timeout, so a reminder still pending after this was
	// interrupted (the process stopped mid-send).
	stalePendingAfter = 10 * time.Minute
)

// reminderScheduler sends automatic appointment reminders. It is not bound to Wails; the
// ReminderService starts and stops it with the app.
//
// Several schedulers may share one database (the desktop app can run on the LAN server PC
// alongside the server). Each reminder is claimed with one atomic insert before it's sent, so
// it goes out at most once however many are running.
type reminderScheduler struct {
	notifications *NotificationService
	appointments  storage.AppointmentRepository
	patients      storage.PatientRepository
	practice      storage.PracticeConfigRepository
	reminders     storage.ReminderRepository
	logs          storage.NotificationLogRepository
	audit         *AuditService
	now           func() time.Time

	passMu   sync.Mutex
	statusMu sync.Mutex
	status   domain.ReminderJobStatus
}

// reminderPass tallies one pass for the job status.
type reminderPass struct {
	sent, failed, unknown, skipped int
	errs                           []string
}

func (p *reminderPass) problem(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	for _, e := range p.errs {
		if e == msg {
			return
		}
	}
	p.errs = append(p.errs, msg)
}

func (s *reminderScheduler) run(ctx context.Context, interval time.Duration) {
	s.runPass(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runPass(ctx)
		}
	}
}

func (s *reminderScheduler) jobStatus() domain.ReminderJobStatus {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	return s.status
}

func (s *reminderScheduler) setRunning(running bool) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	s.status.Running = running
}

// runPass sends every reminder that is due now. One pass runs at a time per process.
func (s *reminderScheduler) runPass(ctx context.Context) {
	s.passMu.Lock()
	defer s.passMu.Unlock()

	var pass reminderPass
	now := s.now().UTC()
	s.sendDue(ctx, now, &pass)

	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	ranAt := now
	s.status.LastRunAt = &ranAt
	s.status.Sent, s.status.Failed, s.status.Unknown, s.status.Skipped = pass.sent, pass.failed, pass.unknown, pass.skipped
	s.status.LastError = strings.Join(pass.errs, "; ")
}

func (s *reminderScheduler) sendDue(ctx context.Context, now time.Time, pass *reminderPass) {
	if _, err := s.logs.MarkStalePending(ctx, now.Add(-stalePendingAfter)); err != nil {
		pass.problem("%v", err)
	}

	settings, err := s.reminders.GetSettings(ctx)
	if errors.Is(err, storage.ErrNotFound) || (err == nil && !settings.Enabled) {
		return
	}
	if err != nil {
		pass.problem("%v", err)
		return
	}
	practice, err := s.practice.Get(ctx)
	if err != nil {
		pass.problem("practice settings could not be read: %v", err)
		return
	}
	if practice.Timezone == "" {
		pass.problem("the practice timezone is not set")
		return
	}
	loc, err := time.LoadLocation(practice.Timezone)
	if err != nil {
		pass.problem("practice timezone %q is not valid", practice.Timezone)
		return
	}
	hours, err := parseSendingHours(settings.SendingHoursStart, settings.SendingHoursEnd)
	if err != nil {
		pass.problem("%v", err)
		return
	}
	if !hours.within(now, loc) {
		return
	}

	allRules, err := s.reminders.ListRules(ctx)
	if err != nil {
		pass.problem("%v", err)
		return
	}
	var rules []*domain.ReminderRule
	var longest time.Duration
	for _, r := range allRules {
		if r.Enabled {
			rules = append(rules, r)
			longest = max(longest, r.Offset())
		}
	}
	if len(rules) == 0 {
		return
	}

	appointments, err := s.appointments.List(ctx, domain.AppointmentFilter{StartDate: now, EndDate: now.Add(longest)})
	if err != nil {
		pass.problem("appointments could not be read: %v", err)
		return
	}

	// Provider settings come from the OS keychain; read each once per pass, so a missing or
	// locked keychain is reported once instead of for every appointment.
	configs := map[string]map[string]string{}
	configFor := func(rule *domain.ReminderRule) (domain.NotificationProvider, map[string]string, bool) {
		provider, ok := s.notifications.providerFor(rule.ProviderName, rule.Channel)
		if !ok {
			pass.problem("%s provider %q is not available", rule.Channel, rule.ProviderName)
			return nil, nil, false
		}
		config, cached := configs[rule.ProviderName]
		if !cached {
			config, err = s.notifications.providerConfig(rule.ProviderName)
			if err != nil {
				pass.problem("%v", err)
				config = nil
			}
			configs[rule.ProviderName] = config
		}
		return provider, config, config != nil
	}

	for _, appt := range appointments {
		if !appt.StartTime.After(now) || !remindableStatus(appt.Status) {
			continue
		}
		for _, rule := range rules {
			if !reminderDueNow(rule, appt.StartTime, now, hours, loc) {
				continue
			}
			provider, config, ok := configFor(rule)
			if !ok {
				continue
			}
			s.sendReminder(ctx, now, rule, appt.ID, provider, config, practice, loc, hours, pass)
		}
	}
}

func remindableStatus(status domain.AppointmentStatus) bool {
	return status == domain.AppointmentStatusScheduled || status == domain.AppointmentStatusConfirmed
}

// sendReminder sends one rule's reminder for one appointment, if it still applies.
func (s *reminderScheduler) sendReminder(
	ctx context.Context, now time.Time, rule *domain.ReminderRule, appointmentID string,
	provider domain.NotificationProvider, config map[string]string,
	practice *domain.PracticeConfig, loc *time.Location, hours sendingHours, pass *reminderPass,
) {
	// Re-read both records: the appointment may have been cancelled or moved, or the patient
	// opted out, since the pass started. In any of those cases nothing is recorded, so the
	// reminder can still go out later if that changes in time.
	appt, err := s.appointments.GetByID(ctx, appointmentID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			pass.problem("appointment could not be read: %v", err)
		}
		return
	}
	if !appt.StartTime.After(now) || !remindableStatus(appt.Status) ||
		!reminderDueNow(rule, appt.StartTime, now, hours, loc) {
		return
	}
	patient, err := s.patients.GetByID(ctx, appt.PatientID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			pass.problem("patient could not be read: %v", err)
		}
		return
	}
	if patient.Status == domain.StatusArchived || !patient.ReminderOptIn {
		return
	}

	start := appt.StartTime.UTC()
	entry := &domain.NotificationLog{
		ID:               newNotificationID(),
		PatientID:        patient.ID,
		AppointmentID:    appt.ID,
		Channel:          rule.Channel,
		ProviderName:     rule.ProviderName,
		SentAt:           now,
		ReminderKind:     rule.Kind(),
		AppointmentStart: &start,
	}

	var skipReason string
	switch rule.Channel {
	case domain.NotificationChannelEmail:
		if strings.TrimSpace(patient.Email) == "" {
			return
		}
		entry.Recipient = strings.TrimSpace(patient.Email)
	case domain.NotificationChannelSMS:
		if strings.TrimSpace(patient.PhonePrimary) == "" {
			return
		}
		number, err := toSMSNumber(patient.PhonePrimary, practice.CountryCode)
		if err != nil {
			entry.Recipient = patient.PhonePrimary
			skipReason = strings.TrimPrefix(err.Error(), storage.ErrInvalidInput.Error()+": ")
		} else {
			entry.Recipient = number
		}
	default:
		return
	}

	values := reminderValues(patient, appt.StartTime, practice, loc)
	entry.Body = renderReminderTemplate(rule.BodyTemplate, values)
	if rule.Channel == domain.NotificationChannelEmail {
		entry.Subject = renderReminderTemplate(rule.SubjectTemplate, values)
	}
	if skipReason == "" && rule.Channel == domain.NotificationChannelSMS {
		if err := checkReminderSMSLength(entry.Body); err != nil {
			skipReason = strings.TrimPrefix(err.Error(), storage.ErrInvalidInput.Error()+": ")
		}
	}
	if skipReason != "" {
		s.recordSkipped(ctx, entry, skipReason, pass)
		return
	}

	var limit *domain.ReminderLimit
	if rule.Channel == domain.NotificationChannelSMS {
		limit = smsReminderLimit(now, loc)
	}
	claim, err := s.logs.ClaimReminder(ctx, entry, limit)
	switch {
	case err != nil:
		pass.problem("%v", err)
		return
	case claim == domain.ReminderAlreadyClaimed:
		return
	case claim == domain.ReminderLimitReached:
		// Deferred, not skipped: later passes try again, so a reminder held back today by
		// another appointment's text can still go out tomorrow if it's within its window.
		return
	}

	result, sendErr := s.notifications.send(ctx, provider, &domain.NotificationMessage{
		Channel: rule.Channel,
		To:      entry.Recipient,
		Subject: entry.Subject,
		Body:    entry.Body,
	}, config)

	// The outcome is recorded even if the app is shutting down, so a sent reminder is never
	// left looking unsent.
	recordCtx := context.WithoutCancel(ctx)
	status := deliveryStatus(sendErr)
	externalID, errMsg := "", ""
	if result != nil {
		externalID = result.ExternalMessageID
	}
	if sendErr != nil {
		errMsg = sendErr.Error()
	}
	if err := s.logs.UpdateResult(recordCtx, entry.ID, status, externalID, errMsg); err != nil {
		pass.problem("reminder outcome could not be recorded: %v", err)
	}

	label := fmt.Sprintf("automatic %s reminder (%s before the appointment) via %s", rule.Channel, describeOffset(rule.OffsetMinutes), rule.ProviderName)
	var detail string
	switch status {
	case domain.NotificationStatusSent:
		pass.sent++
		detail = "Sent " + label
	case domain.NotificationStatusUnknown:
		pass.unknown++
		detail = fmt.Sprintf("Sent %s, but delivery is uncertain: %v", label, sendErr)
	default:
		pass.failed++
		detail = fmt.Sprintf("Failed to send %s: %v", label, sendErr)
	}
	if err := s.audit.logSystemPatientAction(domain.SystemActorReminders, domain.AuditActionCreate,
		patient.ID, "notification", entry.ID, detail); err != nil {
		pass.problem("audit log entry could not be written: %v", err)
	}
}

func (s *reminderScheduler) recordSkipped(ctx context.Context, entry *domain.NotificationLog, reason string, pass *reminderPass) {
	entry.ErrorMessage = reason
	if err := s.logs.RecordSkipped(ctx, entry); err != nil {
		pass.problem("%v", err)
		return
	}
	pass.skipped++
}

// describeOffset renders a rule's timing for audit entries, for example "2 days" or "2 hours".
func describeOffset(minutes int) string {
	plural := func(n int, unit string) string {
		if n == 1 {
			return fmt.Sprintf("1 %s", unit)
		}
		return fmt.Sprintf("%d %ss", n, unit)
	}
	switch {
	case minutes%(24*60) == 0:
		return plural(minutes/(24*60), "day")
	case minutes%60 == 0:
		return plural(minutes/60, "hour")
	default:
		return plural(minutes, "minute")
	}
}
