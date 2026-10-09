package services

import (
	"errors"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

func mustLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%q): %v", name, err)
	}
	return loc
}

func TestParseSendingHours(t *testing.T) {
	h, err := parseSendingHours("08:00", "20:00")
	if err != nil || h.start != 480 || h.end != 1200 {
		t.Fatalf("parseSendingHours(08:00, 20:00) = %+v, %v", h, err)
	}
	for _, bad := range [][2]string{
		{"20:00", "08:00"}, {"08:00", "08:00"}, {"8:00", "20:00"}, {"08:00", "24:00"},
		{"08:60", "20:00"}, {"", "20:00"}, {"ab:cd", "20:00"}, {"-1:00", "20:00"},
	} {
		if _, err := parseSendingHours(bad[0], bad[1]); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("parseSendingHours(%q, %q) should be rejected, got %v", bad[0], bad[1], err)
		}
	}
}

func TestReminderDueNow(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	hours, _ := parseSendingHours("08:00", "20:00")
	twoDays := &domain.ReminderRule{OffsetMinutes: domain.ReminderOffsetTwoDays}
	twoHours := &domain.ReminderRule{OffsetMinutes: domain.ReminderOffsetTwoHours}
	at := func(day, hour, minute int) time.Time { return time.Date(2026, 10, day, hour, minute, 0, 0, la) }

	tests := []struct {
		name  string
		rule  *domain.ReminderRule
		start time.Time
		now   time.Time
		want  bool
	}{
		{"2 days: just before due", twoDays, at(15, 10, 0), at(13, 9, 59), false},
		{"2 days: exactly due", twoDays, at(15, 10, 0), at(13, 10, 0), true},
		{"2 days: late but before latest send", twoDays, at(15, 10, 0), at(14, 19, 0), true},
		{"2 days: exactly the latest send", twoDays, at(16, 20, 0), at(16, 8, 0), true},
		{"2 days: just past the latest send", twoDays, at(16, 20, 0), at(16, 8, 0).Add(time.Second), false},
		{"2 days: past the latest send (12 hours before)", twoDays, at(15, 10, 0), at(14, 22, 1), false},
		{"2 days: due at night waits for morning", twoDays, at(15, 6, 0), at(13, 6, 0), false},
		{"2 days: ...and goes at 8:00", twoDays, at(15, 6, 0), at(13, 8, 0), true},
		{"2 hours: due on time", twoHours, at(15, 14, 0), at(15, 12, 0), true},
		{"2 hours: 9:00 appointment waits until 8:00", twoHours, at(15, 9, 0), at(15, 7, 0), false},
		{"2 hours: 9:00 appointment sends at 8:00", twoHours, at(15, 9, 0), at(15, 8, 0), true},
		{"2 hours: 9:00 appointment, latest send 8:30", twoHours, at(15, 9, 0), at(15, 8, 31), false},
		{"2 hours: 8:15 appointment is never in hours", twoHours, at(15, 8, 15), at(15, 7, 45), false},
		{"2 hours: 8:15 appointment at 8:00 is past latest send", twoHours, at(15, 8, 15), at(15, 8, 0), false},
		{"2 hours: evening appointment, sending hours end 20:00", twoHours, at(15, 21, 30), at(15, 19, 59), true},
		{"2 hours: 20:00 is outside sending hours", twoHours, at(15, 21, 30), at(15, 20, 0), false},
	}
	for _, tt := range tests {
		if got := reminderDueNow(tt.rule, tt.start, tt.now, hours, la); got != tt.want {
			due, latest := reminderWindow(tt.rule, tt.start)
			t.Errorf("%s: reminderDueNow = %v; want %v (due %v, latest %v)", tt.name, got, tt.want, due.In(la), latest.In(la))
		}
	}
}

// Rules count elapsed time, so across a daylight-saving change "2 days before" is 48 real
// hours, and sending hours follow the local clock.
func TestReminderDueNowAcrossDST(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	hours, _ := parseSendingHours("08:00", "20:00")
	twoDays := &domain.ReminderRule{OffsetMinutes: domain.ReminderOffsetTwoDays}

	// Clocks go back at 2:00 on 2026-11-01. A Monday 10:00 appointment's reminder is due 48
	// hours earlier, which is Saturday 11:00 local time.
	start := time.Date(2026, 11, 2, 10, 0, 0, 0, la)
	due, _ := reminderWindow(twoDays, start)
	if got := due.In(la); got.Day() != 31 || got.Hour() != 11 {
		t.Errorf("Due = %v; want Saturday 11:00 local", got)
	}
	if !reminderDueNow(twoDays, start, due, hours, la) {
		t.Errorf("Expected the reminder to be due at %v", due.In(la))
	}

	// Clocks go forward at 2:00 on 2026-03-08: 8:00 that morning is still the opening time.
	springMorning := time.Date(2026, 3, 8, 8, 0, 0, 0, la)
	if !hours.within(springMorning, la) || hours.within(springMorning.Add(-time.Minute), la) {
		t.Errorf("Sending hours should open at 8:00 local on the day clocks go forward")
	}
}

func TestStartOfDayAndLimit(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	// 23:30 in Los Angeles is already the next day in UTC; the limit's day is the local one.
	now := time.Date(2026, 10, 14, 23, 30, 0, 0, la)
	got := startOfDay(now, la)
	if want := time.Date(2026, 10, 14, 0, 0, 0, 0, la); !got.Equal(want) {
		t.Errorf("startOfDay = %v; want %v", got, want)
	}
	limit := smsReminderLimit(now, la)
	if !limit.DayStart.Equal(got) || limit.DayStart.Location() != time.UTC || limit.PerDay != 1 || limit.PerWeek != 3 ||
		!limit.WeekStart.Equal(now.Add(-7*24*time.Hour)) {
		t.Errorf("Unexpected limit: %+v", limit)
	}
}
