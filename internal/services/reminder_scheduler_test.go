package services

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

// reminderEnv is a practice in Los Angeles with reminders on, the three default rules, two
// fake providers, and a clock the test controls.
type reminderEnv struct {
	t            *testing.T
	db           *sqlite.DB
	la           *time.Location
	service      *ReminderService
	audit        *AuditService
	token        string
	patients     *sqlite.PatientRepository
	appointments *sqlite.AppointmentRepository
	logs         *sqlite.NotificationRepository
	reminders    *sqlite.ReminderRepository
	practice     *sqlite.PracticeConfigRepository
	sms, email   *dummyNotificationProvider
}

func newReminderEnv(t *testing.T) *reminderEnv {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "main.db"))
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	auditDB, err := sqlite.OpenAudit(filepath.Join(dir, "audit.db"))
	if err != nil {
		t.Fatalf("Failed to open audit db: %v", err)
	}
	t.Cleanup(func() { auditDB.Close() })

	ctx := context.Background()
	e := &reminderEnv{
		t:            t,
		db:           db,
		la:           mustLocation(t, "America/Los_Angeles"),
		patients:     sqlite.NewPatientRepository(db),
		appointments: sqlite.NewAppointmentRepository(db),
		logs:         sqlite.NewNotificationRepository(db),
		reminders:    sqlite.NewReminderRepository(db),
		practice:     sqlite.NewPracticeConfigRepository(db),
		sms:          &dummyNotificationProvider{name: "mock_sms", channel: domain.NotificationChannelSMS},
		email:        &dummyNotificationProvider{name: "mock_email", channel: domain.NotificationChannelEmail},
	}
	if err := e.practice.SaveProvider(ctx, &domain.Provider{ID: "prov_1", Name: "Dr Test", Pin: "1234", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	if err := e.practice.SaveOperatory(ctx, &domain.Operatory{ID: "op_1", Name: "Chair 1", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	if err := e.practice.Save(ctx, &domain.PracticeConfig{
		ClinicName: "Smile Dental", Phone: "(206) 555-0100", CountryCode: domain.CountryUS, Currency: "USD",
		DateFormat: "MM/DD/YYYY", Timezone: "America/Los_Angeles",
	}); err != nil {
		t.Fatal(err)
	}

	e.audit = NewAuditService(sqlite.NewAuditRepository(auditDB), e.practice)
	if e.token, err = e.audit.CreateSession("prov_1", "1234"); err != nil {
		t.Fatal(err)
	}
	notifications := NewNotificationService(e.patients, e.appointments, e.practice, e.logs, NewSecretsService(), e.audit)
	RegisterNotificationProvider(notifications, e.sms)
	RegisterNotificationProvider(notifications, e.email)
	e.service = NewReminderService(e.reminders, e.patients, e.appointments, e.practice, e.logs, notifications, e.audit)

	if _, err := e.service.EnableReminders(e.token, defaultTestRules()); err != nil {
		t.Fatalf("EnableReminders failed: %v", err)
	}
	return e
}

func defaultTestRules() []domain.ReminderRule {
	return []domain.ReminderRule{
		{OffsetMinutes: domain.ReminderOffsetTwoDays, Channel: domain.NotificationChannelSMS, Enabled: true,
			BodyTemplate: "{practice_name}: Hi {first_name}, reminder of your appointment on {date} at {time}. Call {practice_phone}. Reply STOP to opt out."},
		{OffsetMinutes: domain.ReminderOffsetTwoDays, Channel: domain.NotificationChannelEmail, Enabled: true,
			SubjectTemplate: "Appointment reminder from {practice_name}", BodyTemplate: "Hi {first_name}, see you on {date} at {time}."},
		{OffsetMinutes: domain.ReminderOffsetTwoHours, Channel: domain.NotificationChannelSMS, Enabled: true,
			BodyTemplate: "{practice_name}: Hi {first_name}, see you today at {time}. Reply STOP to opt out."},
	}
}

func (e *reminderEnv) at(month time.Month, day, hour, minute int) time.Time {
	return time.Date(2026, month, day, hour, minute, 0, 0, e.la)
}

func (e *reminderEnv) addPatient(id string, mutate func(p *domain.Patient)) {
	e.t.Helper()
	p := &domain.Patient{
		ID: id, FirstName: "Jane", LastName: id, Email: id + "@example.com", PhonePrimary: "(202) 555-0123",
		ReminderOptIn: true, Status: domain.StatusActive,
	}
	if mutate != nil {
		mutate(p)
	}
	if err := e.patients.Create(context.Background(), p); err != nil {
		e.t.Fatalf("Failed to create patient: %v", err)
	}
}

func (e *reminderEnv) addAppointment(id, patientID string, start time.Time) {
	e.t.Helper()
	if err := e.appointments.Create(context.Background(), &domain.Appointment{
		ID: id, PatientID: patientID, ProviderID: "prov_1", OperatoryID: "op_1",
		StartTime: start, EndTime: start.Add(time.Hour), Status: domain.AppointmentStatusScheduled,
	}); err != nil {
		e.t.Fatalf("Failed to create appointment: %v", err)
	}
}

func (e *reminderEnv) runAt(now time.Time) {
	e.t.Helper()
	e.service.scheduler.now = func() time.Time { return now }
	e.service.scheduler.runPass(context.Background())
}

func (e *reminderEnv) history(appointmentID string) []*domain.NotificationLog {
	e.t.Helper()
	entries, err := e.logs.ListByAppointment(context.Background(), appointmentID)
	if err != nil {
		e.t.Fatalf("ListByAppointment failed: %v", err)
	}
	return entries
}

func (e *reminderEnv) sentCount() (sms, email int) {
	e.sms.mu.Lock()
	defer e.sms.mu.Unlock()
	e.email.mu.Lock()
	defer e.email.mu.Unlock()
	return len(e.sms.sent), len(e.email.sent)
}

func TestReminderScheduler_SendsEachReminderOnce(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 10, 0)) // Thursday 10:00

	e.runAt(e.at(10, 13, 9, 55))
	if sms, email := e.sentCount(); sms != 0 || email != 0 {
		t.Fatalf("Nothing should be sent before the 2-day reminder is due, got %d texts, %d emails", sms, email)
	}

	e.runAt(e.at(10, 13, 10, 0))
	if sms, email := e.sentCount(); sms != 1 || email != 1 {
		t.Fatalf("Expected the 2-day text and email, got %d texts, %d emails", sms, email)
	}
	text := e.sms.sent[0]
	if text.To != "+12025550123" ||
		text.Body != "Smile Dental: Hi Jane, reminder of your appointment on 10/15/2026 at 10:00 AM. Call (206) 555-0100. Reply STOP to opt out." {
		t.Errorf("Unexpected text: %+v", text)
	}
	if mail := e.email.sent[0]; mail.To != "pat_1@example.com" || mail.Subject != "Appointment reminder from Smile Dental" {
		t.Errorf("Unexpected email: %+v", mail)
	}

	// Later passes, and a restarted app (a fresh scheduler on the same database), send nothing more.
	e.runAt(e.at(10, 13, 10, 5))
	e.service = NewReminderService(e.reminders, e.patients, e.appointments, e.practice, e.logs, e.service.notifications, e.audit)
	e.runAt(e.at(10, 14, 9, 0))
	if sms, email := e.sentCount(); sms != 1 || email != 1 {
		t.Fatalf("Expected no repeats, got %d texts, %d emails", sms, email)
	}

	e.runAt(e.at(10, 15, 8, 0))
	if sms, _ := e.sentCount(); sms != 2 {
		t.Fatalf("Expected the 2-hour text at 8:00, got %d texts in total", sms)
	}
	if got := e.sms.sent[1].Body; got != "Smile Dental: Hi Jane, see you today at 10:00 AM. Reply STOP to opt out." {
		t.Errorf("Unexpected 2-hour text: %q", got)
	}

	history := e.history("appt_1")
	if len(history) != 3 {
		t.Fatalf("Expected 3 log entries, got %d", len(history))
	}
	for _, h := range history {
		if h.Status != domain.NotificationStatusSent || h.ExternalMessageID != "ext_1" || h.ReminderKind == "" || h.AppointmentStart == nil {
			t.Errorf("Unexpected log entry: %+v", h)
		}
	}

	status, err := e.service.GetReminderJobStatus(e.token)
	if err != nil || status.LastRunAt == nil || status.Sent != 1 || status.LastError != "" {
		t.Errorf("Unexpected job status after the last pass: %+v, %v", status, err)
	}
}

func TestReminderScheduler_AuditsAsSystemActor(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addPatient("pat_2", func(p *domain.Patient) { p.PhonePrimary = "+1 555 555 0100" })
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 10, 0))
	e.addAppointment("appt_2", "pat_2", e.at(10, 15, 10, 0))
	e.email.sendErr = errors.New("mailbox unavailable")

	e.runAt(e.at(10, 13, 10, 0))

	logs, err := e.audit.GetAuditLogs(e.token, "", 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	byPatient := map[string][]*domain.AuditLogEntry{}
	for _, l := range logs {
		if l.UserID == domain.SystemActorReminders {
			byPatient[l.PatientID] = append(byPatient[l.PatientID], l)
		}
	}
	// pat_1: text sent and email failed. pat_2: email failed; the text was skipped for an
	// invalid number, and skips aren't audited because nothing left the machine.
	if len(byPatient["pat_1"]) != 2 || len(byPatient["pat_2"]) != 1 {
		t.Fatalf("Expected 2 and 1 system audit entries, got %d and %d", len(byPatient["pat_1"]), len(byPatient["pat_2"]))
	}
	logIDs := map[string]bool{}
	for _, h := range e.history("appt_1") {
		logIDs[h.ID] = true
	}
	var sent, failed bool
	for _, l := range byPatient["pat_1"] {
		if l.UserName != domain.SystemActorName || l.Resource != "notification" || !logIDs[l.ResourceID] {
			t.Errorf("Audit entry not linked to its log row: %+v", l)
		}
		sent = sent || strings.HasPrefix(l.Details, "Sent automatic sms reminder (2 days before the appointment)")
		failed = failed || (strings.HasPrefix(l.Details, "Failed to send automatic email reminder") && strings.Contains(l.Details, "mailbox unavailable"))
	}
	if !sent || !failed {
		t.Errorf("Expected a sent and a failed audit entry, got %+v", byPatient["pat_1"])
	}
}

// After a reschedule, the appointment's new time gets its own reminders.
func TestReminderScheduler_RescheduledAppointment(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 10, 0))
	e.runAt(e.at(10, 13, 10, 0))

	appt, err := e.appointments.GetByID(context.Background(), "appt_1")
	if err != nil {
		t.Fatal(err)
	}
	appt.StartTime, appt.EndTime = e.at(10, 20, 14, 0), e.at(10, 20, 15, 0)
	if err := e.appointments.Update(context.Background(), appt); err != nil {
		t.Fatal(err)
	}

	e.runAt(e.at(10, 18, 14, 0))
	if sms, email := e.sentCount(); sms != 2 || email != 2 {
		t.Fatalf("Expected reminders for both times, got %d texts and %d emails", sms, email)
	}
	if got := e.sms.sent[1].Body; !strings.Contains(got, "10/20/2026 at 2:00 PM") {
		t.Errorf("Expected the new time in the reminder, got %q", got)
	}
}

func TestReminderScheduler_SkipsWithoutRecording(t *testing.T) {
	e := newReminderEnv(t)
	start := e.at(10, 15, 10, 0)
	cases := map[string]func(){
		"cancelled": func() {
			e.addPatient("pat_cancelled", nil)
			e.addAppointment("appt_cancelled", "pat_cancelled", start)
			a, _ := e.appointments.GetByID(context.Background(), "appt_cancelled")
			a.Status = domain.AppointmentStatusCancelled
			_ = e.appointments.Update(context.Background(), a)
		},
		"no_show": func() {
			e.addPatient("pat_noshow", nil)
			e.addAppointment("appt_noshow", "pat_noshow", start)
			a, _ := e.appointments.GetByID(context.Background(), "appt_noshow")
			a.Status = domain.AppointmentStatusNoShow
			_ = e.appointments.Update(context.Background(), a)
		},
		"archived": func() {
			e.addPatient("pat_archived", func(p *domain.Patient) { p.Status = domain.StatusArchived })
			e.addAppointment("appt_archived", "pat_archived", start)
		},
		"opted out": func() {
			e.addPatient("pat_optout", func(p *domain.Patient) { p.ReminderOptIn = false })
			e.addAppointment("appt_optout", "pat_optout", start)
		},
		"no contact details": func() {
			e.addPatient("pat_nocontact", func(p *domain.Patient) { p.Email, p.PhonePrimary = "", "" })
			e.addAppointment("appt_nocontact", "pat_nocontact", start)
		},
		"deleted": func() {
			e.addPatient("pat_deleted", nil)
			e.addAppointment("appt_deleted", "pat_deleted", start)
			_ = e.appointments.Delete(context.Background(), "appt_deleted")
		},
	}
	for _, setup := range cases {
		setup()
	}

	e.runAt(e.at(10, 13, 10, 0))
	if sms, email := e.sentCount(); sms != 0 || email != 0 {
		t.Errorf("Expected nothing sent, got %d texts and %d emails", sms, email)
	}
	for _, id := range []string{"appt_cancelled", "appt_noshow", "appt_archived", "appt_optout", "appt_nocontact", "appt_deleted"} {
		if h := e.history(id); len(h) != 0 {
			t.Errorf("%s: expected nothing recorded, got %d entries", id, len(h))
		}
	}

	// Opting in later, while the reminder is still due, sends it.
	p, _ := e.patients.GetByID(context.Background(), "pat_optout")
	p.ReminderOptIn = true
	if err := e.patients.Update(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	e.runAt(e.at(10, 13, 11, 0))
	if sms, email := e.sentCount(); sms != 1 || email != 1 {
		t.Errorf("Expected reminders once the patient opted in, got %d texts and %d emails", sms, email)
	}
}

func TestReminderScheduler_RecordsSkips(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_badphone", func(p *domain.Patient) { p.PhonePrimary = "555-0123" })
	e.addPatient("pat_longname", func(p *domain.Patient) { p.FirstName = strings.Repeat("Maximiliano", 4) })
	e.addAppointment("appt_badphone", "pat_badphone", e.at(10, 15, 10, 0))
	e.addAppointment("appt_longname", "pat_longname", e.at(10, 15, 10, 0))

	e.runAt(e.at(10, 13, 10, 0))
	e.runAt(e.at(10, 13, 10, 5))

	for id, reason := range map[string]string{"appt_badphone": "not a valid phone number", "appt_longname": "over the 160-character limit"} {
		var skipped []*domain.NotificationLog
		for _, h := range e.history(id) {
			if h.Channel == domain.NotificationChannelSMS {
				skipped = append(skipped, h)
			}
		}
		if len(skipped) != 1 || skipped[0].Status != domain.NotificationStatusSkipped || !strings.Contains(skipped[0].ErrorMessage, reason) {
			t.Errorf("%s: expected one skipped text with %q, got %+v", id, reason, skipped)
		}
	}
	// Both patients still got their emails.
	if sms, email := e.sentCount(); sms != 0 || email != 2 {
		t.Errorf("Expected 0 texts and 2 emails, got %d and %d", sms, email)
	}
}

func TestReminderScheduler_FailedAndUnknownAreNotRetried(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 10, 0))
	e.sms.sendErr = fmt.Errorf("%w: no response from AWS", domain.ErrDeliveryUnknown)
	e.email.sendErr = errors.New("recipient rejected")

	e.runAt(e.at(10, 13, 10, 0))
	e.sms.sendErr, e.email.sendErr = nil, nil
	e.runAt(e.at(10, 13, 10, 5))

	if sms, email := e.sentCount(); sms != 1 || email != 1 {
		t.Fatalf("Expected exactly one attempt per reminder, got %d texts and %d emails", sms, email)
	}
	statuses := map[domain.NotificationChannel]domain.NotificationStatus{}
	for _, h := range e.history("appt_1") {
		statuses[h.Channel] = h.Status
	}
	if statuses[domain.NotificationChannelSMS] != domain.NotificationStatusUnknown || statuses[domain.NotificationChannelEmail] != domain.NotificationStatusFailed {
		t.Errorf("Unexpected statuses: %v", statuses)
	}
	status, _ := e.service.GetReminderJobStatus(e.token)
	if status.Sent != 0 || status.Unknown != 0 || status.Failed != 0 {
		t.Errorf("Expected the second pass to do nothing, got %+v", status)
	}
}

// A reminder left pending by a crash becomes unknown and is never resent.
func TestReminderScheduler_InterruptedSendIsNotResent(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 10, 0))
	start := e.at(10, 15, 10, 0).UTC()
	crashed := &domain.NotificationLog{
		ID: "notif_crashed", PatientID: "pat_1", AppointmentID: "appt_1", Channel: domain.NotificationChannelSMS,
		ProviderName: "mock_sms", Recipient: "+12025550123", Body: "Reminder",
		SentAt: e.at(10, 13, 10, 0).UTC(), ReminderKind: "2880m", AppointmentStart: &start,
	}
	if got, err := e.logs.ClaimReminder(context.Background(), crashed, nil); err != nil || got != domain.ReminderClaimed {
		t.Fatalf("ClaimReminder = %s, %v", got, err)
	}

	e.runAt(e.at(10, 13, 10, 30))
	if sms, _ := e.sentCount(); sms != 0 {
		t.Errorf("Expected the interrupted text not to be resent, got %d texts", sms)
	}
	for _, h := range e.history("appt_1") {
		if h.ID == "notif_crashed" && h.Status != domain.NotificationStatusUnknown {
			t.Errorf("Expected the interrupted reminder to become unknown, got %s", h.Status)
		}
	}
}

func TestReminderScheduler_DoesNothingWhenOffOrMisconfigured(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 10, 0))

	if _, err := e.service.DisableReminders(e.token); err != nil {
		t.Fatal(err)
	}
	e.runAt(e.at(10, 13, 10, 0))
	if sms, email := e.sentCount(); sms != 0 || email != 0 {
		t.Fatalf("Expected nothing sent while reminders are off")
	}

	if _, err := e.service.EnableReminders(e.token, nil); err != nil {
		t.Fatalf("Re-enabling should reuse the existing rules, got %v", err)
	}
	cfg, _ := e.practice.Get(context.Background())
	cfg.Timezone = ""
	if err := e.practice.Save(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	e.runAt(e.at(10, 13, 10, 0))
	status, _ := e.service.GetReminderJobStatus(e.token)
	if sms, email := e.sentCount(); sms != 0 || email != 0 || !strings.Contains(status.LastError, "timezone is not set") {
		t.Errorf("Expected nothing sent and a timezone error, got %d/%d and %q", sms, email, status.LastError)
	}
}

// Catch-up: when the app was closed while a reminder came due, it is sent late only until its
// latest send.
func TestReminderScheduler_CatchUpRespectsLatestSend(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addPatient("pat_2", nil)
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 10, 0))
	e.addAppointment("appt_2", "pat_2", e.at(10, 15, 19, 0))

	// First run since Monday: appt_1's 2-day reminders (latest 14 Oct 22:00) can still go,
	// appt_2's too (latest 15 Oct 07:00), and appt_2's 2-hour text isn't due yet.
	e.runAt(e.at(10, 14, 19, 0))
	if sms, email := e.sentCount(); sms != 2 || email != 2 {
		t.Fatalf("Expected late 2-day reminders for both, got %d texts and %d emails", sms, email)
	}

	// Next opened at 9:31 on the day: appt_1's 2-hour text (latest send 9:30) is not sent late.
	e.runAt(e.at(10, 15, 9, 31))
	if sms, _ := e.sentCount(); sms != 2 {
		t.Errorf("Expected appt_1's 2-hour text to be dropped after its latest send, got %d texts", sms)
	}
	// appt_2's 2-hour text (due 17:00, latest 18:30) goes when it comes due.
	e.runAt(e.at(10, 15, 17, 0))
	if sms, _ := e.sentCount(); sms != 3 {
		t.Errorf("Expected appt_2's 2-hour text, got %d texts", sms)
	}
}

// Opened exactly at a reminder's latest send, it still goes.
func TestReminderScheduler_SendsAtLatestSend(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 10, 0))
	e.runAt(e.at(10, 15, 9, 30))
	if sms, _ := e.sentCount(); sms != 1 {
		t.Errorf("Expected the 2-hour text at its latest send, got %d texts", sms)
	}
}

func TestReminderScheduler_DailyLimitDefers(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 10, 0))
	e.addAppointment("appt_2", "pat_1", e.at(10, 16, 14, 0))

	// Tuesday 14:00: appt_1's 2-day text is due (since 10:00), and so is nothing for appt_2.
	e.runAt(e.at(10, 13, 14, 0))
	// Wednesday 14:00: appt_2's 2-day text is due, and today is a new day.
	e.runAt(e.at(10, 14, 14, 0))
	if sms, _ := e.sentCount(); sms != 2 {
		t.Fatalf("Expected one text each day, got %d", sms)
	}
	// Thursday 8:00: appt_1's 2-hour text is due; one text today so far is none, so it goes.
	e.runAt(e.at(10, 15, 8, 0))
	if sms, _ := e.sentCount(); sms != 3 {
		t.Fatalf("Expected the 2-hour text, got %d texts", sms)
	}
	// Friday 12:00: appt_2's 2-hour text is due, but the patient has had 3 texts this week.
	e.runAt(e.at(10, 16, 12, 0))
	if sms, _ := e.sentCount(); sms != 3 {
		t.Errorf("Expected the weekly limit to hold the fourth text back, got %d texts", sms)
	}
	for _, h := range e.history("appt_2") {
		if h.ReminderKind == "120m" && h.Channel == domain.NotificationChannelSMS {
			t.Errorf("A limit hit should defer, not record anything; got %+v", h)
		}
	}
}

// Two schedulers on one database (the desktop app beside the LAN server) run passes at the
// same moment: each reminder still goes out exactly once.
func TestReminderScheduler_TwoSchedulersSendOnce(t *testing.T) {
	e := newReminderEnv(t)
	for i := range 6 {
		id := fmt.Sprintf("pat_%d", i)
		e.addPatient(id, nil)
		e.addAppointment("appt_"+id, id, e.at(10, 15, 10, 0))
	}
	other := NewReminderService(e.reminders, e.patients, e.appointments, e.practice, e.logs, e.service.notifications, e.audit)
	now := e.at(10, 13, 10, 0)
	e.service.scheduler.now = func() time.Time { return now }
	other.scheduler.now = func() time.Time { return now }

	var wg sync.WaitGroup
	for _, s := range []*reminderScheduler{e.service.scheduler, other.scheduler} {
		wg.Add(1)
		go func(s *reminderScheduler) {
			defer wg.Done()
			s.runPass(context.Background())
		}(s)
	}
	wg.Wait()

	if sms, email := e.sentCount(); sms != 6 || email != 6 {
		t.Errorf("Expected 6 texts and 6 emails, got %d and %d", sms, email)
	}
}

func TestReminderScheduler_ReportsProviderProblems(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 10, 0))
	rules, _ := e.reminders.ListRules(context.Background())
	for _, r := range rules {
		if r.Channel == domain.NotificationChannelEmail {
			r.ProviderName = "removed_email"
			if err := e.reminders.SaveRule(context.Background(), r); err != nil {
				t.Fatal(err)
			}
		}
	}

	e.runAt(e.at(10, 13, 10, 0))
	status, _ := e.service.GetReminderJobStatus(e.token)
	if sms, email := e.sentCount(); sms != 1 || email != 0 || !strings.Contains(status.LastError, `"removed_email" is not available`) {
		t.Errorf("Expected the text sent and the email reported, got %d/%d and %q", sms, email, status.LastError)
	}
	if h := e.history("appt_1"); len(h) != 1 {
		t.Errorf("Expected only the text recorded, so the email can still go once fixed; got %d entries", len(h))
	}
}

func TestReminderScheduler_OutsideSendingHours(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_1", nil)
	e.addAppointment("appt_1", "pat_1", e.at(10, 15, 21, 30))

	e.runAt(e.at(10, 13, 21, 30)) // due, but after 20:00
	if sms, email := e.sentCount(); sms != 0 || email != 0 {
		t.Fatalf("Expected nothing sent outside sending hours")
	}
	e.runAt(e.at(10, 14, 8, 0))
	if sms, email := e.sentCount(); sms != 1 || email != 1 {
		t.Errorf("Expected the reminders when sending hours open, got %d and %d", sms, email)
	}
}

func TestReminderService_JobLifecycle(t *testing.T) {
	e := newReminderEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	e.service.startJob(ctx, time.Hour)
	e.service.startJob(ctx, time.Hour) // a second start is ignored
	deadline := time.Now().Add(5 * time.Second)
	for {
		status, _ := e.service.GetReminderJobStatus(e.token)
		if status.Running && status.LastRunAt != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("The job didn't run its startup pass: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	done := make(chan struct{})
	go func() {
		StopReminderJob(e.service)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("StopReminderJob didn't return")
	}
	if status, _ := e.service.GetReminderJobStatus(e.token); status.Running {
		t.Errorf("Expected the job to be stopped")
	}
	StopReminderJob(e.service) // stopping twice is harmless
}
