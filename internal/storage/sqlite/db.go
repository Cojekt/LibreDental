package sqlite

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/LibreDental/libredental/internal/storage/seed"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

//go:embed audit_migrations/*.sql
var auditEmbedMigrations embed.FS

// gooseMu serializes migrate/migrateAudit: goose.SetBaseFS is a package-level
// global, so two migrations racing (main + audit) could each run against the
// other's embedded migration set.
var gooseMu sync.Mutex

// ErrIncompatibleDatabase is returned when a database was created by a
// pre-baseline release whose migration chain has since been squashed. Running
// the new baseline over it would fail midway on CREATE TABLE, so refuse early.
var ErrIncompatibleDatabase = errors.New("database was created by an older, incompatible LibreDental release; move it aside and start with a fresh data directory")

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
	gooseMu.Lock()
	defer gooseMu.Unlock()
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := checkBaseline(db.DB, "migrations"); err != nil {
		return err
	}

	if err := goose.Up(db.DB, "migrations"); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	if err := seed.Run(db.DB); err != nil {
		return fmt.Errorf("failed to run seeds: %w", err)
	}

	return nil
}

// OpenAudit opens the SQLite database specifically for auditing.
func OpenAudit(dbPath string) (*DB, error) {
	// Enable WAL mode, foreign keys, and normal sync for high performance local execution
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(1)&_pragma=busy_timeout(5000)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite audit db: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite audit db: %w", err)
	}

	sDB := &DB{DB: db}
	if err := sDB.migrateAudit(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run audit migrations: %w", err)
	}

	return sDB, nil
}

func (db *DB) migrateAudit() error {
	gooseMu.Lock()
	defer gooseMu.Unlock()
	goose.SetBaseFS(auditEmbedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := checkBaseline(db.DB, "audit_migrations"); err != nil {
		return err
	}

	if err := goose.Up(db.DB, "audit_migrations"); err != nil {
		return fmt.Errorf("failed to apply audit migrations: %w", err)
	}

	return nil
}

// checkBaseline rejects databases whose applied migrations all predate the
// oldest embedded migration, i.e. ones built from a chain that no longer exists.
// Callers must hold gooseMu with the matching base FS already set.
func checkBaseline(db *sql.DB, dir string) error {
	var exists int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", goose.TableName()).Scan(&exists); err != nil {
		return fmt.Errorf("failed to inspect migration state: %w", err)
	}
	if exists == 0 {
		return nil
	}

	var current int64
	if err := db.QueryRow(fmt.Sprintf("SELECT COALESCE(MAX(version_id), 0) FROM %s WHERE is_applied", goose.TableName())).Scan(&current); err != nil {
		return fmt.Errorf("failed to read migration version: %w", err)
	}
	if current == 0 {
		return nil
	}

	migrations, err := goose.CollectMigrations(dir, 0, math.MaxInt64)
	if err != nil {
		return fmt.Errorf("failed to collect migrations: %w", err)
	}
	if len(migrations) > 0 && current < migrations[0].Version {
		return fmt.Errorf("%w (found schema version %d, oldest supported is %d)", ErrIncompatibleDatabase, current, migrations[0].Version)
	}
	return nil
}

// rowScanner covers both *sql.Row and *sql.Rows for shared scan helpers.
type rowScanner interface {
	Scan(dest ...any) error
}

// nullIfEmpty maps the domain's empty-string "no reference" to SQL NULL so
// optional foreign keys stay valid.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
