package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

// TestMigrations_UpgradePreservesPatientData upgrades a database holding patient, claim, and
// chart data from the initial schema to the latest one. Migrations that rebuild a table
// referenced by foreign keys can fail or lose rows on real installs while passing on the empty
// databases the other tests use.
func TestMigrations_UpgradePreservesPatientData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "upgrade.db")
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", dbPath)

	old, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	gooseMu.Lock()
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		gooseMu.Unlock()
		t.Fatal(err)
	}
	err = goose.UpTo(old, "migrations", 20261002224920)
	gooseMu.Unlock()
	if err != nil {
		t.Fatalf("migrate to initial schema: %v", err)
	}

	for _, stmt := range []string{
		`INSERT INTO patients (id, first_name, last_name, date_of_birth, sex, insurance_carrier,
			insurance_policy_number, insurance_is_subscriber, created_at, updated_at)
		 VALUES ('pat_1', 'Jane', 'Doe', '1985-06-15T00:00:00Z', 'female', 'Delta Dental',
			'POL-1', 1, '2026-01-01', '2026-01-01')`,
		`INSERT INTO claims (id, patient_id, date_of_service, status, line_items, created_at, updated_at)
		 VALUES ('claim_1', 'pat_1', '2026-02-01', 'submitted', '[]', '2026-02-01', '2026-02-01')`,
		`INSERT INTO dental_conditions (id, patient_id, tooth_number, ada_code, status, created_at, updated_at)
		 VALUES ('cond_1', 'pat_1', 3, 'D2392', 'treatment_planned', '2026-02-01', '2026-02-01')`,
		`INSERT INTO practice_config (id, clinic_name, country_code, currency, created_at, updated_at)
		 VALUES (1, 'Smile Dental', 'US', 'USD', '2026-01-01', '2026-01-01')`,
		`INSERT INTO notification_log (id, patient_id, channel, provider_name, recipient, body, status, sent_at)
		 VALUES ('notif_1', 'pat_1', 'sms', 'aws_sms', '+12025550123', 'Reminder', 'sent', '2026-02-01')`,
	} {
		if _, err := old.Exec(stmt); err != nil {
			t.Fatalf("seed old schema: %v\n%s", err, stmt)
		}
	}
	old.Close()

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("upgrade to latest schema: %v", err)
	}
	defer db.Close()

	var name, carrier, policy string
	if err := db.QueryRow(`SELECT first_name, insurance_carrier, insurance_policy_number FROM patients WHERE id = 'pat_1'`).
		Scan(&name, &carrier, &policy); err != nil {
		t.Fatalf("patient lost in upgrade: %v", err)
	}
	if name != "Jane" || carrier != "Delta Dental" || policy != "POL-1" {
		t.Errorf("patient data changed in upgrade: %s / %s / %s", name, carrier, policy)
	}
	assertLinkedRowsIntact(t, db.DB, "upgrade")

	// The reminder migration adds columns to tables that already hold rows: existing
	// notifications have no reminder key, and the practice has no timezone until one is set.
	var kind, apptStart sql.NullString
	var status string
	if err := db.QueryRow(`SELECT status, reminder_kind, appointment_start FROM notification_log WHERE id = 'notif_1'`).
		Scan(&status, &kind, &apptStart); err != nil {
		t.Fatalf("notification lost in upgrade: %v", err)
	}
	if status != "sent" || kind.Valid || apptStart.Valid {
		t.Errorf("existing notification changed in upgrade: %s / %v / %v", status, kind, apptStart)
	}
	var clinic, timezone string
	if err := db.QueryRow(`SELECT clinic_name, timezone FROM practice_config WHERE id = 1`).Scan(&clinic, &timezone); err != nil {
		t.Fatalf("practice config lost in upgrade: %v", err)
	}
	if clinic != "Smile Dental" || timezone != "" {
		t.Errorf("practice config changed in upgrade: %q / timezone %q", clinic, timezone)
	}

	// Rolling back and re-applying must also work on a populated database.
	gooseMu.Lock()
	defer gooseMu.Unlock()
	goose.SetBaseFS(embedMigrations)
	if err := goose.DownTo(db.DB, "migrations", 20261002224920); err != nil {
		t.Fatalf("roll back to initial schema: %v", err)
	}
	if err := goose.Up(db.DB, "migrations"); err != nil {
		t.Fatalf("re-apply after rollback: %v", err)
	}
	if err := db.QueryRow(`SELECT first_name FROM patients WHERE id = 'pat_1'`).Scan(&name); err != nil || name != "Jane" {
		t.Errorf("patient lost across rollback: %q (err %v)", name, err)
	}
	if err := db.QueryRow(`SELECT status FROM notification_log WHERE id = 'notif_1'`).Scan(&status); err != nil || status != "sent" {
		t.Errorf("notification lost across rollback: %q (err %v)", status, err)
	}
	assertLinkedRowsIntact(t, db.DB, "rollback and re-apply")
}

// assertLinkedRowsIntact checks the claim and chart rows referencing pat_1 survived and that no
// foreign key is left dangling.
func assertLinkedRowsIntact(t *testing.T, db *sql.DB, stage string) {
	t.Helper()
	for table, id := range map[string]string{"claims": "claim_1", "dental_conditions": "cond_1"} {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE id = ? AND patient_id = 'pat_1'", id).Scan(&n); err != nil || n != 1 {
			t.Errorf("%s: %s row lost (count %d, err %v)", stage, table, n, err)
		}
	}
	var violations int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations); err != nil || violations != 0 {
		t.Errorf("%s: foreign key violations: %d (err %v)", stage, violations, err)
	}
}
