package sqlite

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/LibreDental/libredental/internal/storage/seed"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

//go:embed audit_migrations/*.sql
var auditEmbedMigrations embed.FS

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
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
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
	goose.SetBaseFS(auditEmbedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db.DB, "audit_migrations"); err != nil {
		return fmt.Errorf("failed to apply audit migrations: %w", err)
	}

	return nil
}

// rowScanner covers both *sql.Row and *sql.Rows for shared scan helpers.
type rowScanner interface {
	Scan(dest ...any) error
}
