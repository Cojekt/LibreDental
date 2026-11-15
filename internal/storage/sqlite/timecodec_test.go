package sqlite

import (
	"database/sql"
	"testing"
	"time"
)

func TestInstant_RoundTripsNanosecondsAsUTC(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE t (at INTEGER NOT NULL CHECK (typeof(at) = 'integer'), maybe INTEGER NULL)`); err != nil {
		t.Fatal(err)
	}

	zone := time.FixedZone("UTC+5", 5*3600)
	// 100 ns steps match the Windows clock resolution.
	want := time.Date(2026, 10, 22, 11, 24, 41, 66594100, zone)
	if _, err := db.Exec(`INSERT INTO t (at, maybe) VALUES (?, ?)`, instant(want), nullInstant(nil)); err != nil {
		t.Fatal(err)
	}

	var got time.Time
	var maybe *time.Time
	if err := db.QueryRow(`SELECT at, maybe FROM t`).Scan(scanInstant(&got), scanNullInstant(&maybe)); err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("got %v, want %v in UTC", got, want)
	}
	if maybe != nil {
		t.Errorf("NULL scanned as %v", maybe)
	}

	if _, err := db.Exec(`UPDATE t SET maybe = ?`, nullInstant(&want)); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT maybe FROM t`).Scan(scanNullInstant(&maybe)); err != nil {
		t.Fatal(err)
	}
	if maybe == nil || !maybe.Equal(want) {
		t.Errorf("nullable instant = %v, want %v", maybe, want)
	}
}

func TestInstant_OrdersNumerically(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE t (id TEXT, at INTEGER NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	// As trimmed text these sort in the wrong order ("...00.06Z" > "...00.0601Z").
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for id, at := range map[string]time.Time{
		"60ms":   base.Add(60 * time.Millisecond),
		"60.1ms": base.Add(60100 * time.Microsecond),
	} {
		if _, err := db.Exec(`INSERT INTO t VALUES (?, ?)`, id, instant(at)); err != nil {
			t.Fatal(err)
		}
	}
	var first string
	if err := db.QueryRow(`SELECT id FROM t ORDER BY at DESC LIMIT 1`).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if first != "60.1ms" {
		t.Errorf("newest = %s, want 60.1ms", first)
	}
}

func TestInstant_RejectsUnstorableTimes(t *testing.T) {
	for name, v := range map[string]time.Time{
		"zero":       {},
		"before1678": time.Date(1600, 1, 1, 0, 0, 0, 0, time.UTC),
		"after2262":  time.Date(2300, 1, 1, 0, 0, 0, 0, time.UTC),
	} {
		if _, err := instant(v).Value(); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestInstantScanner_RejectsText(t *testing.T) {
	var got time.Time
	if err := scanInstant(&got).Scan("2026-10-22 06:24:41 +0000 UTC"); err == nil {
		t.Error("expected error scanning text into an instant")
	}
}

func TestDate_RoundTrip(t *testing.T) {
	dob := time.Date(1985, 6, 15, 0, 0, 0, 0, time.UTC)
	if got := dateValue(dob); got != "1985-06-15" {
		t.Errorf("dateValue = %q", got)
	}
	var got time.Time
	if err := scanDate(&got).Scan("1985-06-15"); err != nil {
		t.Fatal(err)
	}
	if !got.Equal(dob) {
		t.Errorf("scanDate = %v, want %v", got, dob)
	}
	if err := scanDate(&got).Scan("1985-06-15T00:00:00Z"); err == nil {
		t.Error("expected error for non-canonical date")
	}
}
