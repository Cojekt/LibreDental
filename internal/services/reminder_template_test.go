package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

func TestValidateReminderTemplate(t *testing.T) {
	ok := "{practice_name}: Hi {first_name}, see you {date} at {time}. Call {practice_phone}. Reply STOP to opt out."
	if err := validateReminderTemplate(ok); err != nil {
		t.Errorf("Expected all allowed placeholders to be accepted, got %v", err)
	}
	if err := validateReminderTemplate("No placeholders at all {"); err != nil {
		t.Errorf("Expected plain text to be accepted, got %v", err)
	}
	err := validateReminderTemplate("Hi {first_name}, your {reason} visit with {provider} and {reason}")
	if !errors.Is(err, storage.ErrInvalidInput) {
		t.Fatalf("Expected unknown placeholders to be rejected, got %v", err)
	}
	if !strings.Contains(err.Error(), "{reason}, {provider}") || strings.Count(err.Error(), "{reason}") != 1 {
		t.Errorf("Expected each unknown placeholder listed once, got %q", err)
	}
	if err := validateReminderTemplate("{First_Name}"); err == nil {
		t.Errorf("Expected placeholders to be case-sensitive")
	}
}

func TestReminderValuesAndRendering(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	start := time.Date(2026, 10, 15, 17, 5, 0, 0, time.UTC) // 10:05 in Los Angeles
	patient := &domain.Patient{FirstName: "Jonathan", PreferredName: " Jon "}
	us := &domain.PracticeConfig{ClinicName: " Smile Dental ", Phone: "(206) 555-0100", CountryCode: domain.CountryUS, DateFormat: "MM/DD/YYYY"}

	values := reminderValues(patient, start, us, la)
	want := map[string]string{
		"first_name": "Jon", "date": "10/15/2026", "time": "10:05 AM",
		"practice_name": "Smile Dental", "practice_phone": "(206) 555-0100",
	}
	for k, v := range want {
		if values[k] != v {
			t.Errorf("%s = %q; want %q", k, values[k], v)
		}
	}
	got := renderReminderTemplate("{practice_name}: Hi {first_name}, {date} at {time}. {unknown}", values)
	if got != "Smile Dental: Hi Jon, 10/15/2026 at 10:05 AM. {unknown}" {
		t.Errorf("Rendered %q", got)
	}

	values = reminderValues(&domain.Patient{FirstName: "Jane"}, start, us, la)
	if values["first_name"] != "Jane" {
		t.Errorf("Expected the first name when there's no preferred name, got %q", values["first_name"])
	}

	london := mustLocation(t, "Europe/London")
	for _, tt := range []struct {
		country  domain.CountryCode
		format   string
		loc      *time.Location
		date, tm string
	}{
		{domain.CountryGB, "DD/MM/YYYY", london, "15/10/2026", "18:05"},
		{domain.CountryDE, "DD.MM.YYYY", mustLocation(t, "Europe/Berlin"), "15.10.2026", "19:05"},
		{domain.CountryCA, "YYYY-MM-DD", mustLocation(t, "America/Toronto"), "2026-10-15", "1:05 PM"},
		{domain.CountryAU, "DD/MM/YYYY", mustLocation(t, "Australia/Sydney"), "16/10/2026", "4:05 AM"},
		{domain.CountryFR, "", mustLocation(t, "Europe/Paris"), "2026-10-15", "19:05"},
	} {
		v := reminderValues(patient, start, &domain.PracticeConfig{CountryCode: tt.country, DateFormat: tt.format}, tt.loc)
		if v["date"] != tt.date || v["time"] != tt.tm {
			t.Errorf("%s: date %q time %q; want %q %q", tt.country, v["date"], v["time"], tt.date, tt.tm)
		}
	}
}

func TestReminderSMSLengthAndPreview(t *testing.T) {
	if err := checkReminderSMSLength(strings.Repeat("a", 160)); err != nil {
		t.Errorf("Expected 160 characters to be allowed, got %v", err)
	}
	if err := checkReminderSMSLength(strings.Repeat("é", 161)); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected 161 characters to be rejected, got %v", err)
	}

	practice := &domain.PracticeConfig{ClinicName: "Smile Dental", Phone: "(206) 555-0100", CountryCode: domain.CountryUS, DateFormat: "MM/DD/YYYY"}
	body := "{practice_name}: Hi {first_name}, reminder of your appointment on {date} at {time}. Questions? Call {practice_phone}. Reply STOP to opt out."
	p, err := previewReminder(domain.NotificationChannelSMS, "", body, practice)
	if err != nil {
		t.Fatalf("previewReminder failed: %v", err)
	}
	want := "Smile Dental: Hi Christopher, reminder of your appointment on 12/30/2026 at 10:30 AM. Questions? Call (206) 555-0100. Reply STOP to opt out."
	if p.Body != want || p.Characters != len(want) || p.Encoding != smsEncodingGSM7 || p.TooLong {
		t.Errorf("Unexpected preview: %+v", p)
	}

	long := strings.Repeat("x", 140) + " {first_name} {practice_name}"
	if p, _ := previewReminder(domain.NotificationChannelSMS, "", long, practice); !p.TooLong || p.Parts != 2 {
		t.Errorf("Expected a long text to be flagged, got %+v", p)
	}
	// Email has no 160-character limit.
	if p, _ := previewReminder(domain.NotificationChannelEmail, "Reminder for {first_name}", long, practice); p.TooLong || p.Subject != "Reminder for Christopher" {
		t.Errorf("Unexpected email preview: %+v", p)
	}
	if _, err := previewReminder(domain.NotificationChannelSMS, "", "Your {reason} visit", practice); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected unknown placeholders to be rejected, got %v", err)
	}
}
