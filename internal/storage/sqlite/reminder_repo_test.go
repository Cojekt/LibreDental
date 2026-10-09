package sqlite_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestReminderRepository_Settings(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "reminders.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()
	repo := sqlite.NewReminderRepository(db)
	ctx := context.Background()

	if _, err := repo.GetSettings(ctx); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Expected ErrNotFound before settings are saved, got %v", err)
	}

	enabledAt := time.Date(2026, 10, 6, 17, 30, 0, 0, time.UTC)
	settings := &domain.ReminderSettings{
		Enabled: true, SendingHoursStart: "09:00", SendingHoursEnd: "19:30",
		EnabledAt: &enabledAt, EnabledBy: "prov_1",
	}
	if err := repo.SaveSettings(ctx, settings); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}
	got, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if !got.Enabled || got.SendingHoursStart != "09:00" || got.SendingHoursEnd != "19:30" ||
		got.EnabledAt == nil || !got.EnabledAt.Equal(enabledAt) || got.EnabledBy != "prov_1" || got.UpdatedAt.IsZero() {
		t.Errorf("Settings didn't round-trip: %+v", got)
	}

	got.Enabled = false
	got.EnabledAt = nil
	got.EnabledBy = ""
	if err := repo.SaveSettings(ctx, got); err != nil {
		t.Fatalf("SaveSettings (update) failed: %v", err)
	}
	got, _ = repo.GetSettings(ctx)
	if got.Enabled || got.EnabledAt != nil || got.EnabledBy != "" {
		t.Errorf("Expected reminders off with no enabler, got %+v", got)
	}
}

func TestReminderRepository_Rules(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "reminders.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()
	repo := sqlite.NewReminderRepository(db)
	ctx := context.Background()

	if rules, err := repo.ListRules(ctx); err != nil || len(rules) != 0 {
		t.Fatalf("Expected no rules, got %v, %v", rules, err)
	}

	rules := []*domain.ReminderRule{
		{ID: "rule_2h_sms", OffsetMinutes: 120, Channel: domain.NotificationChannelSMS, ProviderName: "aws_sms", BodyTemplate: "See you at {time}", Enabled: true},
		{ID: "rule_2d_sms", OffsetMinutes: 2880, Channel: domain.NotificationChannelSMS, ProviderName: "aws_sms", BodyTemplate: "Reminder {date}", Enabled: true},
		{ID: "rule_2d_email", OffsetMinutes: 2880, Channel: domain.NotificationChannelEmail, ProviderName: "smtp_email", SubjectTemplate: "Reminder", BodyTemplate: "Reminder {date}", Enabled: false},
	}
	for _, r := range rules {
		if err := repo.SaveRule(ctx, r); err != nil {
			t.Fatalf("SaveRule(%s) failed: %v", r.ID, err)
		}
	}
	got, err := repo.ListRules(ctx)
	if err != nil || len(got) != 3 {
		t.Fatalf("Expected 3 rules, got %d, %v", len(got), err)
	}
	if got[0].ID != "rule_2d_email" || got[1].ID != "rule_2d_sms" || got[2].ID != "rule_2h_sms" {
		t.Errorf("Expected rules ordered by offset then channel, got %s, %s, %s", got[0].ID, got[1].ID, got[2].ID)
	}
	if got[0].Enabled || got[0].SubjectTemplate != "Reminder" || got[0].CreatedAt.IsZero() {
		t.Errorf("Rule didn't round-trip: %+v", got[0])
	}

	// Saving updates provider, templates, and enabled, but never the timing or channel.
	update := *got[2]
	update.OffsetMinutes = 60
	update.Channel = domain.NotificationChannelEmail
	update.BodyTemplate = "Changed {time}"
	update.Enabled = false
	if err := repo.SaveRule(ctx, &update); err != nil {
		t.Fatalf("SaveRule (update) failed: %v", err)
	}
	got, _ = repo.ListRules(ctx)
	if r := got[2]; r.OffsetMinutes != 120 || r.Channel != domain.NotificationChannelSMS || r.BodyTemplate != "Changed {time}" || r.Enabled {
		t.Errorf("Expected only templates and enabled to change, got %+v", r)
	}

	// One rule per timing and channel.
	dup := &domain.ReminderRule{ID: "rule_other", OffsetMinutes: 120, Channel: domain.NotificationChannelSMS, ProviderName: "aws_sms", BodyTemplate: "x"}
	if err := repo.SaveRule(ctx, dup); err == nil {
		t.Errorf("Expected a second rule with the same timing and channel to be rejected")
	}
	if err := repo.SaveRule(ctx, &domain.ReminderRule{ID: "rule_bad"}); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected an incomplete rule to be rejected, got %v", err)
	}
}

// reminderFixture creates the patients and appointments the claim tests need.
type reminderFixture struct {
	db   *sqlite.DB
	repo *sqlite.NotificationRepository
}

func newReminderFixture(t *testing.T, appointmentIDs ...string) *reminderFixture {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "claims.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()

	if err := sqlite.NewPatientRepository(db).Create(ctx, &domain.Patient{ID: "pat_1", FirstName: "Jane", LastName: "Doe"}); err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}
	configRepo := sqlite.NewPracticeConfigRepository(db)
	if err := configRepo.SaveProvider(ctx, &domain.Provider{ID: "prov_1", Name: "Dr Test", IsActive: true}); err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	if err := configRepo.SaveOperatory(ctx, &domain.Operatory{ID: "op_1", Name: "Chair 1", IsActive: true}); err != nil {
		t.Fatalf("Failed to create operatory: %v", err)
	}
	appointments := sqlite.NewAppointmentRepository(db)
	start := time.Date(2026, 10, 9, 17, 0, 0, 0, time.UTC)
	for _, id := range appointmentIDs {
		if err := appointments.Create(ctx, &domain.Appointment{
			ID: id, PatientID: "pat_1", ProviderID: "prov_1", OperatoryID: "op_1",
			StartTime: start, EndTime: start.Add(time.Hour), Status: domain.AppointmentStatusScheduled,
		}); err != nil {
			t.Fatalf("Failed to create appointment: %v", err)
		}
	}
	return &reminderFixture{db: db, repo: sqlite.NewNotificationRepository(db)}
}

var claimSeq int

func reminderEntry(appointmentID string, start time.Time, kind string, channel domain.NotificationChannel, sentAt time.Time) *domain.NotificationLog {
	claimSeq++
	return &domain.NotificationLog{
		ID: fmt.Sprintf("notif_test_%d", claimSeq), PatientID: "pat_1", AppointmentID: appointmentID,
		Channel: channel, ProviderName: "aws_sms", Recipient: "+12025550123", Body: "Reminder",
		SentAt: sentAt, ReminderKind: kind, AppointmentStart: &start,
	}
}

func TestNotificationRepository_ClaimReminder(t *testing.T) {
	f := newReminderFixture(t, "appt_1")
	ctx := context.Background()
	start := time.Date(2026, 10, 9, 17, 0, 0, 0, time.UTC)
	now := time.Date(2026, 10, 7, 17, 0, 0, 0, time.UTC)
	sms := domain.NotificationChannelSMS

	claim := func(e *domain.NotificationLog, limit *domain.ReminderLimit) domain.ReminderClaim {
		t.Helper()
		got, err := f.repo.ClaimReminder(ctx, e, limit)
		if err != nil {
			t.Fatalf("ClaimReminder failed: %v", err)
		}
		return got
	}

	first := reminderEntry("appt_1", start, "2880m", sms, now)
	if got := claim(first, nil); got != domain.ReminderClaimed {
		t.Fatalf("First claim = %s; want claimed", got)
	}
	if got := claim(reminderEntry("appt_1", start, "2880m", sms, now), nil); got != domain.ReminderAlreadyClaimed {
		t.Errorf("Second claim for the same reminder = %s; want already_claimed", got)
	}
	// Same appointment, but another timing, channel, or appointment time is a different reminder.
	if got := claim(reminderEntry("appt_1", start, "120m", sms, now), nil); got != domain.ReminderClaimed {
		t.Errorf("Claim for another timing = %s; want claimed", got)
	}
	if got := claim(reminderEntry("appt_1", start, "2880m", domain.NotificationChannelEmail, now), nil); got != domain.ReminderClaimed {
		t.Errorf("Claim for another channel = %s; want claimed", got)
	}
	rescheduled := start.Add(24 * time.Hour)
	if got := claim(reminderEntry("appt_1", rescheduled, "2880m", sms, now), nil); got != domain.ReminderClaimed {
		t.Errorf("Claim after rescheduling = %s; want claimed", got)
	}
	// The same instant written with another offset is the same appointment time.
	sameInstant := start.In(time.FixedZone("PDT", -7*3600))
	if got := claim(reminderEntry("appt_1", sameInstant, "2880m", sms, now), nil); got != domain.ReminderAlreadyClaimed {
		t.Errorf("Claim with the same start in another zone = %s; want already_claimed", got)
	}

	entries, err := f.repo.ListByAppointment(ctx, "appt_1")
	if err != nil {
		t.Fatalf("ListByAppointment failed: %v", err)
	}
	var found *domain.NotificationLog
	for _, e := range entries {
		if e.ID == first.ID {
			found = e
		}
	}
	if found == nil || found.Status != domain.NotificationStatusPending || found.ReminderKind != "2880m" ||
		found.AppointmentStart == nil || !found.AppointmentStart.Equal(start) {
		t.Fatalf("Claimed row didn't round-trip: %+v", found)
	}

	if err := f.repo.UpdateResult(ctx, first.ID, domain.NotificationStatusSent, "msg-1", ""); err != nil {
		t.Fatalf("UpdateResult failed: %v", err)
	}
	if err := f.repo.UpdateResult(ctx, "missing", domain.NotificationStatusSent, "", ""); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Expected ErrNotFound updating a missing row, got %v", err)
	}
	entries, _ = f.repo.ListByAppointment(ctx, "appt_1")
	for _, e := range entries {
		if e.ID == first.ID && (e.Status != domain.NotificationStatusSent || e.ExternalMessageID != "msg-1") {
			t.Errorf("UpdateResult didn't apply: %+v", e)
		}
	}

	if _, err := f.repo.ClaimReminder(ctx, &domain.NotificationLog{ID: "x", PatientID: "pat_1"}, nil); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected an incomplete reminder to be rejected, got %v", err)
	}
}

func TestNotificationRepository_ClaimReminderLimits(t *testing.T) {
	f := newReminderFixture(t, "appt_1", "appt_2", "appt_3", "appt_4", "appt_5")
	ctx := context.Background()
	start := time.Date(2026, 10, 20, 17, 0, 0, 0, time.UTC)
	sms := domain.NotificationChannelSMS
	day := time.Date(2026, 10, 12, 7, 0, 0, 0, time.UTC) // midnight in Los Angeles

	limitAt := func(now time.Time) *domain.ReminderLimit {
		dayStart := day
		for !dayStart.Add(24 * time.Hour).After(now) {
			dayStart = dayStart.Add(24 * time.Hour)
		}
		return &domain.ReminderLimit{DayStart: dayStart, PerDay: 1, WeekStart: now.Add(-7 * 24 * time.Hour), PerWeek: 3}
	}
	claim := func(appt string, now time.Time) domain.ReminderClaim {
		t.Helper()
		got, err := f.repo.ClaimReminder(ctx, reminderEntry(appt, start, "2880m", sms, now), limitAt(now))
		if err != nil {
			t.Fatalf("ClaimReminder failed: %v", err)
		}
		return got
	}

	monday := day.Add(9 * time.Hour)
	if got := claim("appt_1", monday); got != domain.ReminderClaimed {
		t.Fatalf("First text of the day = %s; want claimed", got)
	}
	if got := claim("appt_2", monday.Add(time.Hour)); got != domain.ReminderLimitReached {
		t.Errorf("Second text the same day = %s; want limit_reached", got)
	}
	// Email has no limit here, and a failed send doesn't count.
	if got, _ := f.repo.ClaimReminder(ctx, reminderEntry("appt_2", start, "2880m", domain.NotificationChannelEmail, monday), nil); got != domain.ReminderClaimed {
		t.Errorf("Email claim = %s; want claimed", got)
	}
	if got := claim("appt_2", monday.Add(24*time.Hour)); got != domain.ReminderClaimed {
		t.Fatalf("Next day's text = %s; want claimed", got)
	}
	failed := reminderEntry("appt_3", start, "2880m", sms, monday.Add(48*time.Hour))
	if got, _ := f.repo.ClaimReminder(ctx, failed, limitAt(monday.Add(48*time.Hour))); got != domain.ReminderClaimed {
		t.Fatalf("Third day's text = %s; want claimed", got)
	}
	if err := f.repo.UpdateResult(ctx, failed.ID, domain.NotificationStatusFailed, "", "rejected"); err != nil {
		t.Fatal(err)
	}
	if got := claim("appt_4", monday.Add(48*time.Hour+time.Hour)); got != domain.ReminderClaimed {
		t.Errorf("Text after a failed one the same day = %s; want claimed (failures don't count)", got)
	}
	// Three texts went out this week (days 1, 2, and 3), so day 4 is over the weekly limit.
	if got := claim("appt_5", monday.Add(72*time.Hour)); got != domain.ReminderLimitReached {
		t.Errorf("Fourth text in a week = %s; want limit_reached", got)
	}
	// A week after the first, there's room again.
	if got := claim("appt_5", monday.Add(7*24*time.Hour+time.Hour)); got != domain.ReminderClaimed {
		t.Errorf("Text a week later = %s; want claimed", got)
	}

	// Manual texts to the patient count toward the limit too.
	manualDay := monday.Add(14 * 24 * time.Hour)
	if err := f.repo.Create(ctx, &domain.NotificationLog{
		ID: "notif_manual", PatientID: "pat_1", Channel: sms, ProviderName: "aws_sms",
		Recipient: "+12025550123", Body: "Manual", Status: domain.NotificationStatusSent, SentAt: manualDay,
	}); err != nil {
		t.Fatal(err)
	}
	afterManual := reminderEntry("appt_1", start.Add(30*24*time.Hour), "2880m", sms, manualDay.Add(time.Hour))
	if got, _ := f.repo.ClaimReminder(ctx, afterManual, limitAt(manualDay.Add(time.Hour))); got != domain.ReminderLimitReached {
		t.Errorf("Reminder after a manual text the same day = %s; want limit_reached", got)
	}
}

// Two processes sharing the database each try to text the same patient about a different
// appointment at the same moment: the limit must still allow only one.
func TestNotificationRepository_ClaimReminderIsAtomic(t *testing.T) {
	appointments := []string{"appt_1", "appt_2", "appt_3", "appt_4", "appt_5", "appt_6", "appt_7", "appt_8"}
	f := newReminderFixture(t, appointments...)
	start := time.Date(2026, 10, 20, 17, 0, 0, 0, time.UTC)
	now := time.Date(2026, 10, 12, 17, 0, 0, 0, time.UTC)
	limit := &domain.ReminderLimit{DayStart: now.Add(-10 * time.Hour), PerDay: 1, WeekStart: now.Add(-7 * 24 * time.Hour), PerWeek: 3}

	var wg sync.WaitGroup
	results := make([]domain.ReminderClaim, len(appointments))
	errs := make([]error, len(appointments))
	entries := make([]*domain.NotificationLog, len(appointments))
	for i, appt := range appointments {
		entries[i] = reminderEntry(appt, start, "2880m", domain.NotificationChannelSMS, now)
	}
	for i := range appointments {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = f.repo.ClaimReminder(context.Background(), entries[i], limit)
		}(i)
	}
	wg.Wait()

	claimed := 0
	for i, r := range results {
		if errs[i] != nil {
			t.Fatalf("ClaimReminder failed: %v", errs[i])
		}
		if r == domain.ReminderClaimed {
			claimed++
		}
	}
	if claimed != 1 {
		t.Errorf("Expected exactly 1 of %d simultaneous claims to succeed, got %d (%v)", len(appointments), claimed, results)
	}
}

func TestNotificationRepository_RecordSkippedAndStalePending(t *testing.T) {
	f := newReminderFixture(t, "appt_1")
	ctx := context.Background()
	start := time.Date(2026, 10, 9, 17, 0, 0, 0, time.UTC)
	now := time.Date(2026, 10, 7, 17, 0, 0, 0, time.UTC)

	skipped := reminderEntry("appt_1", start, "2880m", domain.NotificationChannelSMS, now)
	skipped.ErrorMessage = "phone number is not valid"
	if err := f.repo.RecordSkipped(ctx, skipped); err != nil {
		t.Fatalf("RecordSkipped failed: %v", err)
	}
	if err := f.repo.RecordSkipped(ctx, reminderEntry("appt_1", start, "2880m", domain.NotificationChannelSMS, now)); err != nil {
		t.Fatalf("Recording the same skip twice should be a no-op, got %v", err)
	}
	if got, _ := f.repo.ClaimReminder(ctx, reminderEntry("appt_1", start, "2880m", domain.NotificationChannelSMS, now), nil); got != domain.ReminderAlreadyClaimed {
		t.Errorf("Claim after a skip = %s; want already_claimed (skips are final)", got)
	}

	stale := reminderEntry("appt_1", start, "120m", domain.NotificationChannelSMS, now)
	fresh := reminderEntry("appt_1", start, "120m", domain.NotificationChannelEmail, now.Add(20*time.Minute))
	for _, e := range []*domain.NotificationLog{stale, fresh} {
		if got, _ := f.repo.ClaimReminder(ctx, e, nil); got != domain.ReminderClaimed {
			t.Fatalf("Claim = %s; want claimed", got)
		}
	}
	n, err := f.repo.MarkStalePending(ctx, now.Add(10*time.Minute))
	if err != nil || n != 1 {
		t.Fatalf("MarkStalePending = %d, %v; want 1", n, err)
	}
	entries, _ := f.repo.ListByAppointment(ctx, "appt_1")
	statuses := map[string]domain.NotificationStatus{}
	for _, e := range entries {
		statuses[e.ID] = e.Status
	}
	if statuses[skipped.ID] != domain.NotificationStatusSkipped || statuses[stale.ID] != domain.NotificationStatusUnknown || statuses[fresh.ID] != domain.NotificationStatusPending {
		t.Errorf("Unexpected statuses: %v", statuses)
	}
}
