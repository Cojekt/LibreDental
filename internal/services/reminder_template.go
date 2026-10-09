package services

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// The values a reminder template may use. Nothing else from the patient's record can appear in
// a reminder: HIPAA's minimum-necessary standard allows the name, the appointment's date and
// time, and the practice's name and phone number, but not the reason for the visit.
var reminderPlaceholders = []string{"first_name", "date", "time", "practice_name", "practice_phone"}

var placeholderPattern = regexp.MustCompile(`\{([A-Za-z_]+)\}`)

// maxReminderSMSLength is the TCPA healthcare exemption's limit for a text message.
const maxReminderSMSLength = 160

// Sample values for previews; the first name is deliberately long so the length check is
// conservative.
const previewFirstName = "Christopher"

var previewAppointment = time.Date(2026, 12, 30, 10, 30, 0, 0, time.UTC)

// validateReminderTemplate rejects templates that use anything but the allowed placeholders.
func validateReminderTemplate(template string) error {
	var unknown []string
	for _, m := range placeholderPattern.FindAllStringSubmatch(template, -1) {
		if !slices.Contains(reminderPlaceholders, m[1]) && !slices.Contains(unknown, m[0]) {
			unknown = append(unknown, m[0])
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("%w: unknown placeholders %s; allowed: {%s}", storage.ErrInvalidInput,
			strings.Join(unknown, ", "), strings.Join(reminderPlaceholders, "}, {"))
	}
	return nil
}

func renderReminderTemplate(template string, values map[string]string) string {
	return placeholderPattern.ReplaceAllStringFunc(template, func(m string) string {
		if v, ok := values[m[1:len(m)-1]]; ok {
			return v
		}
		return m
	})
}

// reminderValues returns the placeholder values for an appointment, with the time shown in the
// practice's timezone and in its country's date and clock conventions.
func reminderValues(patient *domain.Patient, start time.Time, practice *domain.PracticeConfig, loc *time.Location) map[string]string {
	name := strings.TrimSpace(patient.PreferredName)
	if name == "" {
		name = strings.TrimSpace(patient.FirstName)
	}
	local := start.In(loc)
	return map[string]string{
		"first_name":     name,
		"date":           local.Format(dateLayout(practice.DateFormat)),
		"time":           local.Format(clockLayout(practice.CountryCode)),
		"practice_name":  strings.TrimSpace(practice.ClinicName),
		"practice_phone": strings.TrimSpace(practice.Phone),
	}
}

// dateLayout converts the practice's date format (for example "MM/DD/YYYY") to a Go layout.
// Numeric dates avoid translating month and weekday names, which Go has no locale data for.
func dateLayout(format string) string {
	layout := strings.NewReplacer("YYYY", "2006", "MM", "01", "DD", "02").Replace(format)
	if format == "" || !strings.Contains(layout, "2006") || !strings.Contains(layout, "01") || !strings.Contains(layout, "02") {
		return "2006-01-02"
	}
	return layout
}

// clockLayout uses a 12-hour clock where that's the everyday convention, and 24-hour elsewhere.
func clockLayout(country domain.CountryCode) string {
	switch country {
	case domain.CountryUS, domain.CountryCA, domain.CountryAU:
		return "3:04 PM"
	default:
		return "15:04"
	}
}

// checkReminderSMSLength enforces the TCPA exemption's 160-character limit on a rendered text.
func checkReminderSMSLength(body string) error {
	if n := utf8.RuneCountInString(body); n > maxReminderSMSLength {
		return fmt.Errorf("%w: text message is %d characters, over the %d-character limit for reminders",
			storage.ErrInvalidInput, n, maxReminderSMSLength)
	}
	return nil
}

// previewReminder renders templates with sample values for the template editor.
func previewReminder(channel domain.NotificationChannel, subject, body string, practice *domain.PracticeConfig) (*domain.ReminderPreview, error) {
	if err := validateReminderTemplate(subject); err != nil {
		return nil, err
	}
	if err := validateReminderTemplate(body); err != nil {
		return nil, err
	}
	values := reminderValues(&domain.Patient{FirstName: previewFirstName}, previewAppointment, practice, time.UTC)
	preview := &domain.ReminderPreview{
		Subject: renderReminderTemplate(subject, values),
		Body:    renderReminderTemplate(body, values),
	}
	preview.Characters = utf8.RuneCountInString(preview.Body)
	if channel == domain.NotificationChannelSMS {
		preview.Encoding, _, preview.Parts = smsSize(preview.Body)
		preview.TooLong = preview.Characters > maxReminderSMSLength
	}
	return preview, nil
}
