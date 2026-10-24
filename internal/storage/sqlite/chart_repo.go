package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type ChartRepository struct {
	db *DB
}

func NewChartRepository(db *DB) *ChartRepository {
	return &ChartRepository{db: db}
}

// scanToothCondition scans a single dental_conditions row (as selected by the
// column list used in GetChart/GetConditionByID) into a domain.ToothCondition.
func scanToothCondition(scanner rowScanner) (*domain.ToothCondition, error) {
	var c domain.ToothCondition
	var surfacesJSON sql.NullString
	var adaCode sql.NullString
	var description sql.NullString
	var statusStr string
	var fee sql.NullInt64
	var pricing sql.NullString

	if err := scanner.Scan(
		&c.ID, &c.PatientID, &c.ToothNumber, &surfacesJSON,
		&adaCode, &description, &statusStr, &fee,
		&c.CreatedAt, &c.UpdatedAt, &pricing,
	); err != nil {
		return nil, err
	}

	if pricing.Valid {
		if err := json.Unmarshal([]byte(pricing.String), &c.Pricing); err != nil {
			return nil, err
		}
	}
	c.ADACode = adaCode.String
	c.Description = description.String
	c.Fee = fee.Int64
	c.Status = domain.ToothStatus(statusStr)
	if surfacesJSON.Valid && len(surfacesJSON.String) > 0 {
		if err := json.Unmarshal([]byte(surfacesJSON.String), &c.Surfaces); err != nil {
			return nil, fmt.Errorf("failed to decode tooth condition surfaces: %w", err)
		}
	}
	if c.Surfaces == nil {
		c.Surfaces = []domain.ToothSurface{}
	}

	return &c, nil
}

func (r *ChartRepository) GetChart(ctx context.Context, patientID string) (*domain.DentalChart, error) {
	if patientID == "" {
		return nil, fmt.Errorf("%w: patient ID is required", storage.ErrInvalidInput)
	}

	query := `
	SELECT id, patient_id, tooth_number, surfaces, ada_code, description, status, fee, created_at, updated_at, pricing_source
	FROM dental_conditions
	WHERE patient_id = ?
	ORDER BY tooth_number ASC, created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to query dental conditions: %w", err)
	}
	defer rows.Close()

	var conditions []domain.ToothCondition
	var latestUpdate time.Time

	for rows.Next() {
		c, err := scanToothCondition(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tooth condition: %w", err)
		}

		if c.UpdatedAt.After(latestUpdate) {
			latestUpdate = c.UpdatedAt
		}

		conditions = append(conditions, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if conditions == nil {
		conditions = []domain.ToothCondition{}
	}

	return &domain.DentalChart{
		PatientID:  patientID,
		Conditions: conditions,
		UpdatedAt:  latestUpdate,
	}, nil
}

func (r *ChartRepository) SaveCondition(ctx context.Context, c *domain.ToothCondition) (bool, error) {
	if c.ID == "" || c.PatientID == "" || c.ToothNumber <= 0 {
		return false, fmt.Errorf("%w: ID, PatientID, and valid ToothNumber are required", storage.ErrInvalidInput)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	inserted, err := saveConditionOn(ctx, tx, c)
	if err != nil {
		return false, err
	}
	return inserted, tx.Commit()
}

func saveConditionOn(ctx context.Context, q queryExecutor, c *domain.ToothCondition) (bool, error) {
	if c.ID == "" || c.PatientID == "" || c.ToothNumber <= 0 {
		return false, storage.ErrInvalidInput
	}
	var exists bool
	if err := q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM dental_conditions WHERE id=?)", c.ID).Scan(&exists); err != nil {
		return false, err
	}
	surfaces, err := json.Marshal(c.Surfaces)
	if err != nil {
		return false, err
	}
	var pricing any
	var plan any
	if c.Pricing != nil {
		b, e := json.Marshal(c.Pricing)
		if e != nil {
			return false, e
		}
		pricing = string(b)
		plan = nullIfEmpty(c.Pricing.ProviderID)
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = domain.ToothStatusExisting
	}
	_, err = q.ExecContext(ctx, `INSERT INTO dental_conditions(id,patient_id,tooth_number,surfaces,ada_code,description,status,fee,created_at,updated_at,insurance_provider_id,pricing_source) VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET tooth_number=excluded.tooth_number,surfaces=excluded.surfaces,ada_code=excluded.ada_code,description=excluded.description,status=excluded.status,fee=excluded.fee,updated_at=excluded.updated_at,insurance_provider_id=excluded.insurance_provider_id,pricing_source=excluded.pricing_source`, c.ID, c.PatientID, c.ToothNumber, string(surfaces), c.ADACode, c.Description, c.Status, c.Fee, c.CreatedAt, c.UpdatedAt, plan, pricing)
	return !exists, err
}

func (r *ChartRepository) GetConditionByID(ctx context.Context, id string) (*domain.ToothCondition, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}

	query := `
	SELECT id, patient_id, tooth_number, surfaces, ada_code, description, status, fee, created_at, updated_at, pricing_source
	FROM dental_conditions
	WHERE id = ?`

	c, err := scanToothCondition(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query tooth condition: %w", err)
	}

	return c, nil
}

// DeleteCondition deletes the tooth condition with the given ID and returns the
// condition as it existed at deletion time, read and removed within the same
// transaction so callers get an accurate record (e.g. PatientID) to audit-log even
// if a concurrent request deletes/recreates the same ID for a different patient.
func (r *ChartRepository) DeleteCondition(ctx context.Context, id string) (*domain.ToothCondition, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
	SELECT id, patient_id, tooth_number, surfaces, ada_code, description, status, fee, created_at, updated_at, pricing_source
	FROM dental_conditions
	WHERE id = ?`

	c, err := scanToothCondition(tx.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query tooth condition: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM dental_conditions WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("failed to delete tooth condition: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return c, nil
}
