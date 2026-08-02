package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB wraps the *sql.DB handle for LibreDental SQLite storage.
type DB struct {
	*sql.DB
}

// Open opens a SQLite database at the given path and runs initial migrations.
func Open(dbPath string) (*DB, error) {
	// Enable WAL mode, foreign keys, and normal sync for high performance local execution
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(1)&_pragma=busy_timeout(5000)", dbPath)
	
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite db: %w", err)
	}

	sDB := &DB{DB: db}
	if err := sDB.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return sDB, nil
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS patients (
		id TEXT PRIMARY KEY,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		middle_name TEXT DEFAULT '',
		preferred_name TEXT DEFAULT '',
		date_of_birth TEXT NOT NULL,
		gender TEXT NOT NULL,
		email TEXT DEFAULT '',
		phone_primary TEXT DEFAULT '',
		phone_secondary TEXT DEFAULT '',
		address_line1 TEXT DEFAULT '',
		address_line2 TEXT DEFAULT '',
		city TEXT DEFAULT '',
		state TEXT DEFAULT '',
		zip_code TEXT DEFAULT '',
		medical_alerts TEXT DEFAULT '[]',
		allergies TEXT DEFAULT '[]',
		notes TEXT DEFAULT '',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		status TEXT NOT NULL DEFAULT 'active'
	);

	CREATE INDEX IF NOT EXISTS idx_patients_name ON patients(last_name, first_name);
	CREATE INDEX IF NOT EXISTS idx_patients_dob ON patients(date_of_birth);

	CREATE TABLE IF NOT EXISTS appointments (
		id TEXT PRIMARY KEY,
		patient_id TEXT NOT NULL,
		provider_id TEXT NOT NULL,
		operatory_id TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		status TEXT NOT NULL,
		reason TEXT DEFAULT '',
		color TEXT DEFAULT '',
		notes TEXT DEFAULT '',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		FOREIGN KEY (patient_id) REFERENCES patients(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_appointments_patient ON appointments(patient_id);
	CREATE INDEX IF NOT EXISTS idx_appointments_date ON appointments(start_time, end_time);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id TEXT PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		user_id TEXT NOT NULL,
		user_name TEXT NOT NULL,
		patient_id TEXT DEFAULT '',
		action TEXT NOT NULL,
		resource TEXT NOT NULL,
		resource_id TEXT DEFAULT '',
		details TEXT DEFAULT '',
		ip_address TEXT DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_audit_patient ON audit_logs(patient_id);
	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
	`

	_, err := db.Exec(schema)
	return err
}
