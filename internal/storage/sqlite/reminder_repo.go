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

// ReminderRepository implements storage.ReminderRepository for SQLite. Timestamps are stored
// as RFC 3339 UTC text, like appointment times.
type ReminderRepository struct {
	db *DB
}

func NewReminderRepository(db *DB) *ReminderRepository {
	return &ReminderRepository{db: db}
}

func (r *ReminderRepository) GetSettings(ctx context.Context) (*domain.ReminderSettings, error) {
	var (
		settings  domain.ReminderSettings
		enabled   int
		enabledAt sql.NullString
		enabledBy sql.NullString
		updatedAt string
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT enabled, sending_hours_start, sending_hours_end, enabled_at, enabled_by, updated_at
		FROM reminder_settings WHERE id = 1`,
	).Scan(&enabled, &settings.SendingHoursStart, &settings.SendingHoursEnd, &enabledAt, &enabledBy, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reminder settings: %w", err)
	}
	settings.Enabled = enabled != 0
	settings.EnabledBy = enabledBy.String
	if enabledAt.Valid {
		if t, err := time.Parse(time.RFC3339, enabledAt.String); err == nil {
			settings.EnabledAt = &t
		}
	}
	settings.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &settings, nil
}

func (r *ReminderRepository) SaveSettings(ctx context.Context, settings *domain.ReminderSettings) error {
	settings.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	var enabledAt any
	if settings.EnabledAt != nil {
		enabledAt = settings.EnabledAt.UTC().Format(time.RFC3339)
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reminder_settings (id, enabled, sending_hours_start, sending_hours_end, enabled_at, enabled_by, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			sending_hours_start = excluded.sending_hours_start,
			sending_hours_end = excluded.sending_hours_end,
			enabled_at = excluded.enabled_at,
			enabled_by = excluded.enabled_by,
			updated_at = excluded.updated_at`,
		boolToInt(settings.Enabled), settings.SendingHoursStart, settings.SendingHoursEnd,
		enabledAt, nullIfEmpty(settings.EnabledBy), settings.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to save reminder settings: %w", err)
	}
	return nil
}

func (r *ReminderRepository) ListRules(ctx context.Context) ([]*domain.ReminderRule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, offset_minutes, channel, provider_name, subject_template, body_template, enabled, created_at, updated_at
		FROM reminder_rules ORDER BY offset_minutes DESC, channel ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list reminder rules: %w", err)
	}
	defer rows.Close()

	rules := []*domain.ReminderRule{}
	for rows.Next() {
		var (
			rule                 domain.ReminderRule
			channel              string
			enabled              int
			createdAt, updatedAt string
		)
		if err := rows.Scan(&rule.ID, &rule.OffsetMinutes, &channel, &rule.ProviderName,
			&rule.SubjectTemplate, &rule.BodyTemplate, &enabled, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan reminder rule: %w", err)
		}
		rule.Channel = domain.NotificationChannel(channel)
		rule.Enabled = enabled != 0
		rule.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		rule.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

func (r *ReminderRepository) SaveRule(ctx context.Context, rule *domain.ReminderRule) error {
	if rule.ID == "" || rule.OffsetMinutes <= 0 || rule.Channel == "" || rule.ProviderName == "" || rule.BodyTemplate == "" {
		return fmt.Errorf("%w: reminder rule needs an ID, a positive offset, a channel, a provider, and a body", storage.ErrInvalidInput)
	}
	now := time.Now().UTC().Truncate(time.Second)
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reminder_rules (id, offset_minutes, channel, provider_name, subject_template, body_template, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider_name = excluded.provider_name,
			subject_template = excluded.subject_template,
			body_template = excluded.body_template,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at`,
		rule.ID, rule.OffsetMinutes, string(rule.Channel), rule.ProviderName,
		rule.SubjectTemplate, rule.BodyTemplate, boolToInt(rule.Enabled),
		rule.CreatedAt.UTC().Format(time.RFC3339), rule.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to save reminder rule: %w", err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
