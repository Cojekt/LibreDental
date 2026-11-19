package sqlite

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Instants are stored as INTEGER Unix nanoseconds (UTC) so that ORDER BY and range comparisons
// are numeric and identical on every platform. Calendar dates are stored as TEXT YYYY-MM-DD.
// Never pass a time.Time to SQL directly: the driver would write Time.String() text, which the
// schema's typeof CHECK constraints reject.

const dateLayout = "2006-01-02"

var (
	minInstant = time.Unix(0, -1<<63).UTC()
	maxInstant = time.Unix(0, 1<<63-1).UTC()
)

func instantNanos(t time.Time) (int64, error) {
	if t.IsZero() {
		return 0, fmt.Errorf("sqlite: cannot store zero time as an instant")
	}
	if t.Before(minInstant) || t.After(maxInstant) {
		return 0, fmt.Errorf("sqlite: time %s is outside the storable instant range", t.UTC().Format(time.RFC3339))
	}
	return t.UnixNano(), nil
}

// instant converts t for use as a SQL argument; the zero time and out-of-range values fail the
// statement instead of being stored.
func instant(t time.Time) driver.Valuer { return instantArg{t} }

// nullInstant stores nil as NULL.
func nullInstant(t *time.Time) driver.Valuer {
	if t == nil {
		return nullArg{}
	}
	return instantArg{*t}
}

type instantArg struct{ t time.Time }

func (a instantArg) Value() (driver.Value, error) { return instantNanos(a.t) }

type nullArg struct{}

func (nullArg) Value() (driver.Value, error) { return nil, nil }

// scanInstant reads a NOT NULL instant column into dst.
func scanInstant(dst *time.Time) *instantScanner { return &instantScanner{dst: dst} }

type instantScanner struct{ dst *time.Time }

func (s *instantScanner) Scan(src any) error {
	ns, ok := src.(int64)
	if !ok {
		return fmt.Errorf("sqlite: instant column holds %T, want int64", src)
	}
	*s.dst = time.Unix(0, ns).UTC()
	return nil
}

// scanNullInstant reads a nullable instant column; NULL sets dst to nil.
func scanNullInstant(dst **time.Time) *nullInstantScanner { return &nullInstantScanner{dst: dst} }

type nullInstantScanner struct{ dst **time.Time }

func (s *nullInstantScanner) Scan(src any) error {
	if src == nil {
		*s.dst = nil
		return nil
	}
	var t time.Time
	if err := (&instantScanner{dst: &t}).Scan(src); err != nil {
		return err
	}
	*s.dst = &t
	return nil
}

// dateValue formats the calendar date of t as YYYY-MM-DD, ignoring its clock and zone.
func dateValue(t time.Time) string { return t.Format(dateLayout) }

// scanDate reads a YYYY-MM-DD column into dst as midnight UTC.
func scanDate(dst *time.Time) *dateScanner { return &dateScanner{dst: dst} }

type dateScanner struct{ dst *time.Time }

func (s *dateScanner) Scan(src any) error {
	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("sqlite: date column holds %T, want text", src)
	}
	t, err := time.Parse(dateLayout, str)
	if err != nil {
		return fmt.Errorf("sqlite: invalid date %q: %w", str, err)
	}
	*s.dst = t
	return nil
}
