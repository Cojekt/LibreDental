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

// ProcedureRepository implements storage.ProcedureCodeRepository and storage.FeeScheduleRepository.
type ProcedureRepository struct {
	db *DB
}

func NewProcedureRepository(db *DB) *ProcedureRepository {
	return &ProcedureRepository{db: db}
}

// ── ProcedureCodeRepository Implementation ──────────────────────────────────

func (r *ProcedureRepository) List(ctx context.Context, countryCode domain.CountryCode) ([]*domain.ProcedureCode, error) {
	query := `
		SELECT country_code, code, category, description, default_fee, is_active
		FROM procedure_codes
		WHERE country_code = ? AND is_active = 1
		ORDER BY category ASC, code ASC`

	rows, err := r.db.QueryContext(ctx, query, string(countryCode))
	if err != nil {
		return nil, fmt.Errorf("failed to list procedure codes: %w", err)
	}
	defer rows.Close()

	var list []*domain.ProcedureCode
	for rows.Next() {
		var pc domain.ProcedureCode
		var isActiveInt int
		if err := rows.Scan(&pc.CountryCode, &pc.Code, &pc.Category, &pc.Description, &pc.DefaultFee, &isActiveInt); err != nil {
			return nil, fmt.Errorf("failed to scan procedure code: %w", err)
		}
		pc.IsActive = isActiveInt == 1
		list = append(list, &pc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if list == nil {
		list = []*domain.ProcedureCode{}
	}
	return list, nil
}

func (r *ProcedureRepository) GetByCode(ctx context.Context, countryCode domain.CountryCode, code string) (*domain.ProcedureCode, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT country_code, code, category, description, default_fee, is_active
		FROM procedure_codes
		WHERE country_code = ? AND code = ?`, string(countryCode), code)

	var pc domain.ProcedureCode
	var isActiveInt int
	err := row.Scan(&pc.CountryCode, &pc.Code, &pc.Category, &pc.Description, &pc.DefaultFee, &isActiveInt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get procedure code: %w", err)
	}
	pc.IsActive = isActiveInt == 1
	return &pc, nil
}

// ── FeeScheduleRepository Implementation ────────────────────────────────────

func (r *ProcedureRepository) Save(ctx context.Context, schedule *domain.FeeSchedule) error {
	if schedule == nil || schedule.CountryCode == "" || schedule.Code == "" {
		return fmt.Errorf("%w: country_code and code are required", storage.ErrInvalidInput)
	}
	if schedule.ID == "" {
		schedule.ID = fmt.Sprintf("fee_%d", time.Now().UnixNano())
	}
	schedule.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fee_schedules (id, country_code, code, provider_id, custom_fee, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(country_code, code, provider_id) DO UPDATE SET
			custom_fee = excluded.custom_fee,
			updated_at = excluded.updated_at`,
		schedule.ID, string(schedule.CountryCode), schedule.Code,
		schedule.ProviderID, schedule.CustomFee, schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save fee schedule: %w", err)
	}
	return nil
}

func (r *ProcedureRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}
	res, err := r.db.ExecContext(ctx, "DELETE FROM fee_schedules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete fee schedule: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *ProcedureRepository) ListFeeSchedules(ctx context.Context, countryCode domain.CountryCode, providerID string) ([]*domain.FeeSchedule, error) {
	query := `
		SELECT id, country_code, code, provider_id, custom_fee, updated_at
		FROM fee_schedules
		WHERE country_code = ? AND (provider_id = ? OR provider_id = '')
		ORDER BY code ASC`

	rows, err := r.db.QueryContext(ctx, query, string(countryCode), providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list fee schedules: %w", err)
	}
	defer rows.Close()

	var list []*domain.FeeSchedule
	for rows.Next() {
		var fs domain.FeeSchedule
		if err := rows.Scan(&fs.ID, &fs.CountryCode, &fs.Code, &fs.ProviderID, &fs.CustomFee, &fs.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan fee schedule: %w", err)
		}
		list = append(list, &fs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if list == nil {
		list = []*domain.FeeSchedule{}
	}
	return list, nil
}

func (r *ProcedureRepository) GetEffectiveFee(ctx context.Context, countryCode domain.CountryCode, code string, providerID string) (int64, error) {
	// 1. Check provider-specific fee override
	if providerID != "" {
		var fee int64
		err := r.db.QueryRowContext(ctx, `
			SELECT custom_fee FROM fee_schedules
			WHERE country_code = ? AND code = ? AND provider_id = ?`,
			string(countryCode), code, providerID,
		).Scan(&fee)
		if err == nil {
			return fee, nil
		}
	}

	// 2. Check practice-wide custom fee override (provider_id = '')
	var practiceFee int64
	err := r.db.QueryRowContext(ctx, `
		SELECT custom_fee FROM fee_schedules
		WHERE country_code = ? AND code = ? AND provider_id = ''`,
		string(countryCode), code,
	).Scan(&practiceFee)
	if err == nil {
		return practiceFee, nil
	}

	// 3. Fallback to procedure_codes catalog default fee
	var defaultFee int64
	err = r.db.QueryRowContext(ctx, `
		SELECT default_fee FROM procedure_codes
		WHERE country_code = ? AND code = ?`,
		string(countryCode), code,
	).Scan(&defaultFee)
	if err == nil {
		return defaultFee, nil
	}

	return 0, nil
}
