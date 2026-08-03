package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage"
)

type ConfigRepository struct {
	db *DB
}

func NewConfigRepository(db *DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

func (r *ConfigRepository) Get(ctx context.Context) (*domain.PracticeConfig, error) {
	query := `
	SELECT id, country_code, currency, tooth_system, date_format, created_at, updated_at
	FROM practice_config WHERE id = 1`

	row := r.db.QueryRowContext(ctx, query)

	var cfg domain.PracticeConfig
	var countryStr, toothStr string

	err := row.Scan(
		&cfg.ID,
		&countryStr,
		&cfg.Currency,
		&toothStr,
		&cfg.DateFormat,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to fetch practice config: %w", err)
	}

	cfg.CountryCode = domain.CountryCode(countryStr)
	cfg.ToothSystem = domain.ToothSystem(toothStr)

	return &cfg, nil
}

func (r *ConfigRepository) Save(ctx context.Context, cfg *domain.PracticeConfig) error {
	now := time.Now().UTC()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
	cfg.ID = 1

	query := `
	INSERT INTO practice_config (id, country_code, currency, tooth_system, date_format, created_at, updated_at)
	VALUES (1, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		country_code = excluded.country_code,
		currency = excluded.currency,
		tooth_system = excluded.tooth_system,
		date_format = excluded.date_format,
		updated_at = excluded.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		cfg.CountryCode, cfg.Currency, cfg.ToothSystem, cfg.DateFormat, cfg.CreatedAt, cfg.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save practice config: %w", err)
	}

	return nil
}
