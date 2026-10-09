package services

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// Wails binds every exported ReminderService method for the frontend and, in LAN server
// mode, for any client on the network. Starting and stopping the job are package functions
// so they can't be called from there; this list keeps it that way.
func TestReminderService_BoundMethods(t *testing.T) {
	want := []string{
		"DisableReminders", "EnableReminders", "GetRecipientCounts", "GetReminderJobStatus", "GetReminderSettings",
		"ListReminderProviders", "ListReminderRules", "PreviewReminderTemplate", "SaveReminderRule", "SaveSendingHours",
	}
	typ := reflect.TypeOf(&ReminderService{})
	var got []string
	for i := range typ.NumMethod() {
		got = append(got, typ.Method(i).Name)
	}
	if !slices.Equal(got, want) {
		t.Errorf("ReminderService exported methods changed:\n got  %v\n want %v", got, want)
	}
}

func TestReminderService_RequiresSession(t *testing.T) {
	e := newReminderEnv(t)
	s, bad := e.service, "bad_token"
	calls := map[string]error{}
	_, calls["GetReminderSettings"] = s.GetReminderSettings(bad)
	_, calls["SaveSendingHours"] = s.SaveSendingHours(bad, "08:00", "20:00")
	_, calls["GetRecipientCounts"] = s.GetRecipientCounts(bad)
	_, calls["EnableReminders"] = s.EnableReminders(bad, nil)
	_, calls["DisableReminders"] = s.DisableReminders(bad)
	_, calls["ListReminderRules"] = s.ListReminderRules(bad)
	_, calls["SaveReminderRule"] = s.SaveReminderRule(bad, domain.ReminderRule{})
	_, calls["ListReminderProviders"] = s.ListReminderProviders(bad)
	_, calls["PreviewReminderTemplate"] = s.PreviewReminderTemplate(bad, "sms", "", "Hi")
	_, calls["GetReminderJobStatus"] = s.GetReminderJobStatus(bad)
	for name, err := range calls {
		if err != ErrUnauthorized {
			t.Errorf("%s without a session: got %v, want ErrUnauthorized", name, err)
		}
	}
}

func TestReminderService_EnableAndDisable(t *testing.T) {
	e := newReminderEnv(t) // turned on once, with the default rules
	ctx := context.Background()

	rules, err := e.service.ListReminderRules(e.token)
	if err != nil || len(rules) != 3 {
		t.Fatalf("Expected the 3 default rules, got %d, %v", len(rules), err)
	}
	for _, r := range rules {
		want := map[domain.NotificationChannel]string{domain.NotificationChannelSMS: "mock_sms", domain.NotificationChannelEmail: "mock_email"}[r.Channel]
		if r.ProviderName != want || !r.Enabled || !strings.HasPrefix(r.ID, "rule_") {
			t.Errorf("Rule not set up as expected: %+v", r)
		}
	}

	settings, err := e.service.GetReminderSettings(e.token)
	if err != nil || !settings.Enabled || settings.EnabledBy != "prov_1" || settings.EnabledAt == nil ||
		settings.SendingHoursStart != "08:00" || settings.SendingHoursEnd != "20:00" {
		t.Fatalf("Unexpected settings: %+v, %v", settings, err)
	}

	// Turning reminders off keeps the rules; turning them back on doesn't add more.
	if settings, err = e.service.DisableReminders(e.token); err != nil || settings.Enabled || settings.EnabledAt != nil {
		t.Fatalf("DisableReminders: %+v, %v", settings, err)
	}
	if _, err := e.service.EnableReminders(e.token, defaultTestRules()); err != nil {
		t.Fatalf("EnableReminders again: %v", err)
	}
	if rules, _ := e.reminders.ListRules(ctx); len(rules) != 3 {
		t.Errorf("Expected the existing 3 rules to be reused, got %d", len(rules))
	}

	logs, _ := e.audit.GetAuditLogs(e.token, "", 100, 0)
	var on, off int
	for _, l := range logs {
		if l.Resource != "reminder_settings" || l.UserID != "prov_1" {
			continue
		}
		if strings.HasPrefix(l.Details, "Turned on automatic reminders for ") {
			on++
		}
		if l.Details == "Turned off automatic reminders" {
			off++
		}
	}
	if on != 2 || off != 1 {
		t.Errorf("Expected turning on twice and off once to be audited, got %d and %d", on, off)
	}
}

func TestReminderService_EnableRequiresTimezoneAndValidRules(t *testing.T) {
	e := newReminderEnv(t)
	ctx := context.Background()
	cfg, _ := e.practice.Get(ctx)
	cfg.Timezone = ""
	if err := e.practice.Save(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := e.service.EnableReminders(e.token, nil); !errors.Is(err, storage.ErrInvalidInput) || !strings.Contains(err.Error(), "timezone") {
		t.Errorf("Expected turning on without a timezone to be refused, got %v", err)
	}

	// On a practice that never turned reminders on, bad default rules are refused before
	// anything is saved.
	bad := map[string][]domain.ReminderRule{
		"none":            nil,
		"voice":           {{OffsetMinutes: 60, Channel: domain.NotificationChannelVoice, BodyTemplate: "Hi"}},
		"after the visit": {{OffsetMinutes: -60, Channel: domain.NotificationChannelSMS, BodyTemplate: "Hi"}},
		"unknown field":   {{OffsetMinutes: 60, Channel: domain.NotificationChannelSMS, BodyTemplate: "Your {reason} visit"}},
		"email subject":   {{OffsetMinutes: 60, Channel: domain.NotificationChannelEmail, BodyTemplate: "Hi"}},
		"duplicate": {
			{OffsetMinutes: 60, Channel: domain.NotificationChannelSMS, BodyTemplate: "Hi"},
			{OffsetMinutes: 60, Channel: domain.NotificationChannelSMS, BodyTemplate: "Hi again"},
		},
	}
	for name, rules := range bad {
		f := newReminderEnvWithoutRules(t)
		if _, err := f.service.EnableReminders(f.token, rules); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("%s: expected the rules to be refused, got %v", name, err)
		}
		if settings, _ := f.service.GetReminderSettings(f.token); settings.Enabled {
			t.Errorf("%s: reminders should still be off", name)
		}
		if rules, _ := f.reminders.ListRules(ctx); len(rules) != 0 {
			t.Errorf("%s: expected no rules saved, got %d", name, len(rules))
		}
	}
}

// newReminderEnvWithoutRules is a practice that has never turned reminders on.
func newReminderEnvWithoutRules(t *testing.T) *reminderEnv {
	t.Helper()
	e := newReminderEnv(t)
	ctx := context.Background()
	if _, err := e.db.ExecContext(ctx, "DELETE FROM reminder_rules"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.service.DisableReminders(e.token); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestReminderService_RecipientCounts(t *testing.T) {
	e := newReminderEnv(t)
	e.addPatient("pat_both", nil)
	e.addPatient("pat_email_only", func(p *domain.Patient) { p.PhonePrimary = "" })
	e.addPatient("pat_bad_phone", func(p *domain.Patient) { p.PhonePrimary = "555-0123"; p.Email = "" })
	e.addPatient("pat_opted_out", func(p *domain.Patient) { p.ReminderOptIn = false })
	e.addPatient("pat_archived", func(p *domain.Patient) { p.Status = domain.StatusArchived })

	counts, err := e.service.GetRecipientCounts(e.token)
	if err != nil {
		t.Fatal(err)
	}
	if *counts != (domain.ReminderRecipientCounts{OptedIn: 3, WithMobile: 1, WithEmail: 2}) {
		t.Errorf("Unexpected counts: %+v", counts)
	}
}

func TestReminderService_SaveRuleAndHours(t *testing.T) {
	e := newReminderEnv(t)
	rules, _ := e.service.ListReminderRules(e.token)
	var sms, email *domain.ReminderRule
	for _, r := range rules {
		if r.Channel == domain.NotificationChannelEmail {
			email = r
		} else if r.OffsetMinutes == domain.ReminderOffsetTwoHours {
			sms = r
		}
	}

	edit := *sms
	edit.BodyTemplate = "{practice_name}: see you at {time}."
	edit.Enabled = false
	edit.OffsetMinutes, edit.Channel = 30, domain.NotificationChannelEmail // ignored
	saved, err := e.service.SaveReminderRule(e.token, edit)
	if err != nil || saved.BodyTemplate != edit.BodyTemplate || saved.Enabled || saved.OffsetMinutes != domain.ReminderOffsetTwoHours || saved.Channel != domain.NotificationChannelSMS {
		t.Fatalf("SaveReminderRule: %+v, %v", saved, err)
	}

	for name, mutate := range map[string]func(r *domain.ReminderRule){
		"unknown placeholder":   func(r *domain.ReminderRule) { r.BodyTemplate = "Your {procedure} visit" },
		"empty body":            func(r *domain.ReminderRule) { r.BodyTemplate = " " },
		"empty email subject":   func(r *domain.ReminderRule) { r.SubjectTemplate = "" },
		"provider for SMS only": func(r *domain.ReminderRule) { r.ProviderName = "mock_sms" },
		"unregistered provider": func(r *domain.ReminderRule) { r.ProviderName = "nope" },
	} {
		r := *email
		mutate(&r)
		if _, err := e.service.SaveReminderRule(e.token, r); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("%s: expected rejection, got %v", name, err)
		}
	}
	if _, err := e.service.SaveReminderRule(e.token, domain.ReminderRule{ID: "missing"}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Expected ErrNotFound for an unknown rule, got %v", err)
	}

	if settings, err := e.service.SaveSendingHours(e.token, "09:00", "18:30"); err != nil || settings.SendingHoursStart != "09:00" || settings.SendingHoursEnd != "18:30" || !settings.Enabled {
		t.Errorf("SaveSendingHours: %+v, %v", settings, err)
	}
	if _, err := e.service.SaveSendingHours(e.token, "18:00", "09:00"); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected inverted sending hours to be rejected, got %v", err)
	}

	providers, err := e.service.ListReminderProviders(e.token)
	if err != nil || !slices.Equal(providers["sms"], []string{"mock_sms"}) || !slices.Equal(providers["email"], []string{"mock_email"}) {
		t.Errorf("ListReminderProviders: %v, %v", providers, err)
	}

	preview, err := e.service.PreviewReminderTemplate(e.token, "sms", "", "{practice_name}: Hi {first_name}")
	if err != nil || preview.Body != "Smile Dental: Hi Christopher" || preview.TooLong {
		t.Errorf("PreviewReminderTemplate: %+v, %v", preview, err)
	}
}
