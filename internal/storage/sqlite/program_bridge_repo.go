package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// ProgramBridgeRepository implements storage.ProgramBridgeRepository for SQLite.
type ProgramBridgeRepository struct {
	db *DB
}

func NewProgramBridgeRepository(db *DB) *ProgramBridgeRepository {
	return &ProgramBridgeRepository{db: db}
}

const programBridgeColumns = `name, enabled, path, args, created_at, updated_at`

func (r *ProgramBridgeRepository) Get(ctx context.Context, name string) (*domain.BridgeConfig, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+programBridgeColumns+` FROM program_bridges WHERE name = ?`, name)
	cfg, err := scanProgramBridge(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get program bridge: %w", err)
	}
	return cfg, nil
}

func (r *ProgramBridgeRepository) List(ctx context.Context) ([]*domain.BridgeConfig, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+programBridgeColumns+` FROM program_bridges ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list program bridges: %w", err)
	}
	defer rows.Close()

	var configs []*domain.BridgeConfig
	for rows.Next() {
		cfg, err := scanProgramBridge(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan program bridge: %w", err)
		}
		configs = append(configs, cfg)
	}
	return configs, rows.Err()
}

// Save inserts or replaces a bridge's config, keeping its original created_at.
func (r *ProgramBridgeRepository) Save(ctx context.Context, cfg *domain.BridgeConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("%w: bridge name is required", storage.ErrInvalidInput)
	}
	now := time.Now().UTC()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO program_bridges (`+programBridgeColumns+`) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			enabled = excluded.enabled,
			path = excluded.path,
			args = excluded.args,
			updated_at = excluded.updated_at`,
		cfg.Name, cfg.Enabled, cfg.Path, cfg.Args, cfg.CreatedAt, cfg.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save program bridge: %w", err)
	}
	return nil
}

func scanProgramBridge(row rowScanner) (*domain.BridgeConfig, error) {
	var cfg domain.BridgeConfig
	if err := row.Scan(&cfg.Name, &cfg.Enabled, &cfg.Path, &cfg.Args, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
		return nil, err
	}
	return &cfg, nil
}
