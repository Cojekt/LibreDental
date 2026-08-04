package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/pkg/storage"
)

// SystemSettingsRepository handles SQLite persistence for application & system settings.
type SystemSettingsRepository struct {
	db *DB
}

// NewSystemSettingsRepository returns a new SystemSettingsRepository instance.
func NewSystemSettingsRepository(db *DB) *SystemSettingsRepository {
	return &SystemSettingsRepository{db: db}
}

// GetSetting fetches a system setting value by key.
func (r *SystemSettingsRepository) GetSetting(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM system_settings WHERE key = ?`
	row := r.db.QueryRowContext(ctx, query, key)

	var value string
	err := row.Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", storage.ErrNotFound
		}
		return "", fmt.Errorf("failed to fetch setting (%s): %w", key, err)
	}

	return value, nil
}

// SetSetting inserts or updates a system setting value by key.
func (r *SystemSettingsRepository) SetSetting(ctx context.Context, key string, value string) error {
	now := time.Now().UTC()
	query := `
	INSERT INTO system_settings (key, value, updated_at)
	VALUES (?, ?, ?)
	ON CONFLICT(key) DO UPDATE SET
		value = excluded.value,
		updated_at = excluded.updated_at`

	_, err := r.db.ExecContext(ctx, query, key, value, now)
	if err != nil {
		return fmt.Errorf("failed to save setting (%s): %w", key, err)
	}

	return nil
}
