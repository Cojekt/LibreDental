package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type TimecardRepository struct {
	db *DB
}

func NewTimecardRepository(db *DB) *TimecardRepository {
	return &TimecardRepository{db: db}
}

func (r *TimecardRepository) ListTimecards(ctx context.Context, providerID string, startDate *time.Time, endDate *time.Time) ([]*domain.Timecard, error) {
	query := `
	SELECT id, provider_id, clock_in, clock_out, hourly_rate, total_hours, total_pay, paid_at, is_manual, created_at, updated_at
	FROM timecards
	WHERE 1=1
	`
	args := []interface{}{}

	if providerID != "" && providerID != "all" {
		query += ` AND provider_id = ?`
		args = append(args, providerID)
	}

	if startDate != nil {
		query += ` AND clock_in >= ?`
		args = append(args, *startDate)
	}

	if endDate != nil {
		query += ` AND clock_in <= ?`
		args = append(args, *endDate)
	}

	query += ` ORDER BY clock_in DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list timecards: %w", err)
	}
	defer rows.Close()

	var timecards []*domain.Timecard
	for rows.Next() {
		var t domain.Timecard
		var clockOut sql.NullTime
		var totalHours sql.NullFloat64
		var totalPay sql.NullFloat64
		var paidAt sql.NullTime
		var isManualInt int

		err := rows.Scan(
			&t.ID, &t.ProviderID, &t.ClockIn, &clockOut, &t.HourlyRate,
			&totalHours, &totalPay, &paidAt, &isManualInt, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timecard: %w", err)
		}

		if clockOut.Valid {
			t.ClockOut = &clockOut.Time
		}
		if totalHours.Valid {
			t.TotalHours = totalHours.Float64
		}
		if totalPay.Valid {
			t.TotalPay = totalPay.Float64
		}
		if paidAt.Valid {
			t.PaidAt = &paidAt.Time
		}
		t.IsManual = isManualInt == 1

		timecards = append(timecards, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timecards: %w", err)
	}

	return timecards, nil
}

func (r *TimecardRepository) GetActiveTimecard(ctx context.Context, providerID string) (*domain.Timecard, error) {
	query := `
	SELECT id, provider_id, clock_in, clock_out, hourly_rate, total_hours, total_pay, paid_at, is_manual, created_at, updated_at
	FROM timecards
	WHERE provider_id = ? AND clock_out IS NULL
	ORDER BY clock_in DESC LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, providerID)

	var t domain.Timecard
	var clockOut sql.NullTime
	var totalHours sql.NullFloat64
	var totalPay sql.NullFloat64
	var paidAt sql.NullTime
	var isManualInt int

	err := row.Scan(
		&t.ID, &t.ProviderID, &t.ClockIn, &clockOut, &t.HourlyRate,
		&totalHours, &totalPay, &paidAt, &isManualInt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get active timecard: %w", err)
	}

	if clockOut.Valid {
		t.ClockOut = &clockOut.Time
	}
	if totalHours.Valid {
		t.TotalHours = totalHours.Float64
	}
	if totalPay.Valid {
		t.TotalPay = totalPay.Float64
	}
	if paidAt.Valid {
		t.PaidAt = &paidAt.Time
	}
	t.IsManual = isManualInt == 1

	return &t, nil
}

func (r *TimecardRepository) SaveTimecard(ctx context.Context, t *domain.Timecard) error {
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	isManualInt := 0
	if t.IsManual {
		isManualInt = 1
	}

	query := `
	INSERT INTO timecards (
		id, provider_id, clock_in, clock_out, hourly_rate, total_hours, total_pay, paid_at, is_manual, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		provider_id = excluded.provider_id,
		clock_in = excluded.clock_in,
		clock_out = excluded.clock_out,
		hourly_rate = excluded.hourly_rate,
		total_hours = excluded.total_hours,
		total_pay = excluded.total_pay,
		paid_at = excluded.paid_at,
		is_manual = excluded.is_manual,
		updated_at = excluded.updated_at
	`

	var clockOut sql.NullTime
	if t.ClockOut != nil {
		clockOut.Valid = true
		clockOut.Time = *t.ClockOut
	}

	var totalHours sql.NullFloat64
	if t.TotalHours > 0 {
		totalHours.Valid = true
		totalHours.Float64 = t.TotalHours
	}

	var totalPay sql.NullFloat64
	if t.TotalPay > 0 {
		totalPay.Valid = true
		totalPay.Float64 = t.TotalPay
	}

	var paidAt sql.NullTime
	if t.PaidAt != nil {
		paidAt.Valid = true
		paidAt.Time = *t.PaidAt
	}

	_, err := r.db.ExecContext(
		ctx, query,
		t.ID, t.ProviderID, t.ClockIn, clockOut, t.HourlyRate,
		totalHours, totalPay, paidAt, isManualInt, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save timecard: %w", err)
	}

	return nil
}

func (r *TimecardRepository) GetTotalOwed(ctx context.Context, providerID string) (float64, error) {
	query := `
	SELECT SUM(total_pay) FROM timecards
	WHERE provider_id = ? AND paid_at IS NULL AND (clock_out IS NOT NULL OR is_manual = 1)
	`
	var total sql.NullFloat64
	err := r.db.QueryRowContext(ctx, query, providerID).Scan(&total)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to calculate total owed: %w", err)
	}

	if total.Valid {
		return total.Float64, nil
	}
	return 0.0, nil
}

func (r *TimecardRepository) MarkTimecardsPaid(ctx context.Context, providerID string) error {
	query := `
	UPDATE timecards SET paid_at = ?, updated_at = ?
	WHERE provider_id = ? AND paid_at IS NULL AND (clock_out IS NOT NULL OR is_manual = 1)
	`
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query, now, now, providerID)
	if err != nil {
		return fmt.Errorf("failed to mark timecards paid: %w", err)
	}
	return nil
}

func (r *TimecardRepository) DeleteTimecard(ctx context.Context, id string) error {
	query := `DELETE FROM timecards WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete timecard: %w", err)
	}
	return nil
}
