package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// The reminder timing rules, kept free of I/O so they can be tested exhaustively. Every
// decision is made from wall-clock times (stored appointment starts against time.Now), never
// from elapsed ticks, because timers pause while a computer sleeps.

// Per-patient limits on text reminders, from the TCPA healthcare exemption.
const (
	smsRemindersPerDay  = 1
	smsRemindersPerWeek = 3
)

// sendingHours is the daily window, in minutes after local midnight, in which reminders may
// be sent. The end is exclusive.
type sendingHours struct {
	start, end int
}

func parseSendingHours(start, end string) (sendingHours, error) {
	s, err := parseClock(start)
	if err != nil {
		return sendingHours{}, err
	}
	e, err := parseClock(end)
	if err != nil {
		return sendingHours{}, err
	}
	if s >= e {
		return sendingHours{}, fmt.Errorf("%w: sending hours must start before they end", storage.ErrInvalidInput)
	}
	return sendingHours{start: s, end: e}, nil
}

// parseClock parses "HH:MM" (24-hour) into minutes after midnight.
func parseClock(v string) (int, error) {
	h, m, ok := strings.Cut(strings.TrimSpace(v), ":")
	hour, errH := strconv.Atoi(h)
	minute, errM := strconv.Atoi(m)
	if !ok || len(h) != 2 || len(m) != 2 || errH != nil || errM != nil || hour > 23 || minute > 59 || hour < 0 || minute < 0 {
		return 0, fmt.Errorf("%w: %q is not a time of day (HH:MM)", storage.ErrInvalidInput, v)
	}
	return hour*60 + minute, nil
}

// within reports whether t falls inside the sending hours in loc.
func (h sendingHours) within(t time.Time, loc *time.Location) bool {
	local := t.In(loc)
	minutes := local.Hour()*60 + local.Minute()
	return minutes >= h.start && minutes < h.end
}

// reminderWindow returns when a reminder comes due and the latest it may still be sent, for an
// appointment starting at start.
func reminderWindow(rule *domain.ReminderRule, start time.Time) (due, latest time.Time) {
	return start.Add(-rule.Offset()), start.Add(-rule.LatestSend())
}

// reminderDueNow reports whether the reminder should be sent at now: it has come due, it isn't
// past its latest send, and now is within sending hours.
func reminderDueNow(rule *domain.ReminderRule, start, now time.Time, hours sendingHours, loc *time.Location) bool {
	due, latest := reminderWindow(rule, start)
	return !now.Before(due) && !now.After(latest) && hours.within(now, loc)
}

// startOfDay returns local midnight of the day containing t, in loc.
func startOfDay(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

// smsReminderLimit is the TCPA limit as of now: one text per day in the practice's timezone and
// three in any seven days.
func smsReminderLimit(now time.Time, loc *time.Location) *domain.ReminderLimit {
	return &domain.ReminderLimit{
		DayStart:  startOfDay(now, loc).UTC(),
		PerDay:    smsRemindersPerDay,
		WeekStart: now.Add(-7 * 24 * time.Hour).UTC(),
		PerWeek:   smsRemindersPerWeek,
	}
}
