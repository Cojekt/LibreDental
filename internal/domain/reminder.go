package domain

import (
	"fmt"
	"time"
)

// Default reminder timings, in minutes before the appointment.
const (
	ReminderOffsetTwoDays  = 2 * 24 * 60
	ReminderOffsetTwoHours = 2 * 60
)

// Default sending hours, in the practice's timezone.
const (
	DefaultSendingHoursStart = "08:00"
	DefaultSendingHoursEnd   = "20:00"
)

// ReminderSettings is the practice-wide switch and sending hours for automatic reminders.
// Reminders are off until staff turn them on.
type ReminderSettings struct {
	Enabled bool `json:"enabled"`
	// SendingHoursStart and SendingHoursEnd ("HH:MM", practice timezone) bound when
	// reminders may be sent; the end is exclusive.
	SendingHoursStart string     `json:"sending_hours_start"`
	SendingHoursEnd   string     `json:"sending_hours_end"`
	EnabledAt         *time.Time `json:"enabled_at,omitempty"`
	EnabledBy         string     `json:"enabled_by,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// DefaultReminderSettings returns the settings used before anything has been saved.
func DefaultReminderSettings() ReminderSettings {
	return ReminderSettings{
		SendingHoursStart: DefaultSendingHoursStart,
		SendingHoursEnd:   DefaultSendingHoursEnd,
	}
}

// ReminderRule sends one automatic reminder per appointment on one channel, OffsetMinutes
// before the appointment starts.
type ReminderRule struct {
	ID              string              `json:"id"`
	OffsetMinutes   int                 `json:"offset_minutes"`
	Channel         NotificationChannel `json:"channel"`
	ProviderName    string              `json:"provider_name"`
	SubjectTemplate string              `json:"subject_template"`
	BodyTemplate    string              `json:"body_template"`
	Enabled         bool                `json:"enabled"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

// Kind identifies the rule's timing in the notification log, for example "2880m". It is
// derived from the offset rather than the rule ID, so recreating a rule can't resend
// reminders that already went out.
func (r *ReminderRule) Kind() string {
	return fmt.Sprintf("%dm", r.OffsetMinutes)
}

// Offset is how long before the appointment the reminder comes due.
func (r *ReminderRule) Offset() time.Duration {
	return time.Duration(r.OffsetMinutes) * time.Minute
}

// LatestSend is how long before the appointment the reminder can still be sent: a quarter of
// its offset. A reminder held back past this, by sending hours or because the app wasn't
// running, is not sent at all rather than arriving too close to the appointment.
func (r *ReminderRule) LatestSend() time.Duration {
	return r.Offset() / 4
}

// ReminderRecipientCounts is shown before turning reminders on: how many active patients are
// opted in, and how many of those can be reached by text and by email.
type ReminderRecipientCounts struct {
	OptedIn    int `json:"opted_in"`
	WithMobile int `json:"with_mobile"`
	WithEmail  int `json:"with_email"`
}

// ReminderPreview is a template rendered with sample values, for the template editor.
type ReminderPreview struct {
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	Characters int    `json:"characters"`
	Parts      int    `json:"parts"`
	Encoding   string `json:"encoding"`
	// TooLong is set for text messages over the 160-character limit; they would be skipped.
	TooLong bool `json:"too_long"`
}

// ReminderJobStatus describes the reminder job's most recent pass in this process.
type ReminderJobStatus struct {
	Running   bool       `json:"running"`
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
	Sent      int        `json:"sent"`
	Failed    int        `json:"failed"`
	Unknown   int        `json:"unknown"`
	Skipped   int        `json:"skipped"`
	LastError string     `json:"last_error,omitempty"`
}

// ReminderLimit caps how many reminders a patient gets on a channel: at most PerDay since
// DayStart (midnight in the practice's timezone) and PerWeek since WeekStart. It implements
// the TCPA healthcare exemption's one text per day and three per week.
type ReminderLimit struct {
	DayStart  time.Time
	PerDay    int
	WeekStart time.Time
	PerWeek   int
}

// ReminderClaim is the outcome of trying to claim a reminder for sending.
type ReminderClaim string

const (
	ReminderClaimed        ReminderClaim = "claimed"
	ReminderAlreadyClaimed ReminderClaim = "already_claimed"
	ReminderLimitReached   ReminderClaim = "limit_reached"
)
