package sqlite

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestOpen_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_open.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("Expected non-nil *DB, got nil")
	}

	// Verify the DB is usable
	if err := db.Ping(); err != nil {
		t.Fatalf("Ping failed on newly opened DB: %v", err)
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	// A path with a NUL byte is rejected by the OS before SQLite even opens it.
	_, err := Open("/tmp/\x00invalid")
	if err == nil {
		t.Error("Expected error when opening DB with NUL-byte path, got nil")
	}
}

func TestOpen_MigrationsApplied(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_migrations.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Verify at least one expected table exists after migration (e.g. patients)
	var tableName string
	row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='patients'")
	if err := row.Scan(&tableName); err != nil {
		t.Errorf("Expected 'patients' table to exist after migration, got error: %v", err)
	}
}

func TestOpenAudit_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_audit.db")

	db, err := OpenAudit(dbPath)
	if err != nil {
		t.Fatalf("OpenAudit failed: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("Expected non-nil *DB, got nil")
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping failed on newly opened audit DB: %v", err)
	}
}

func TestOpenAudit_AuditMigrationsApplied(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_audit_migrations.db")

	db, err := OpenAudit(dbPath)
	if err != nil {
		t.Fatalf("OpenAudit failed: %v", err)
	}
	defer db.Close()

	// Verify audit_logs table exists after migration
	var tableName string
	row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='audit_logs'")
	if err := row.Scan(&tableName); err != nil {
		t.Errorf("Expected 'audit_logs' table to exist after audit migration, got error: %v", err)
	}
}

func TestOpen_Reopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_reopen.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open failed: %v", err)
	}
	db.Close()

	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("reopening a current database failed: %v", err)
	}
	db.Close()
}

func TestOpen_RejectsPreBaselineDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_old.db")

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	// Mimic a database left behind by the squashed pre-baseline migration chain.
	for _, stmt := range []string{
		"CREATE TABLE goose_db_version (id INTEGER PRIMARY KEY AUTOINCREMENT, version_id INTEGER NOT NULL, is_applied INTEGER NOT NULL, tstamp TIMESTAMP DEFAULT (datetime('now')))",
		"INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, 1), (20260920024046, 1)",
		"CREATE TABLE patients (id TEXT NOT NULL PRIMARY KEY)",
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("setup %q failed: %v", stmt, err)
		}
	}
	raw.Close()

	_, err = Open(dbPath)
	if !errors.Is(err, ErrIncompatibleDatabase) {
		t.Fatalf("expected ErrIncompatibleDatabase, got %v", err)
	}
}
