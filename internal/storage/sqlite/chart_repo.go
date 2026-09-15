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

	if err := scanner.Scan(
		&c.ID, &c.PatientID, &c.ToothNumber, &surfacesJSON,
		&adaCode, &description, &statusStr, &fee,
		&c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return nil, err
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
	SELECT id, patient_id, tooth_number, surfaces, ada_code, description, status, fee, created_at, updated_at
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

	surfacesJSON, err := json.Marshal(c.Surfaces)
	if err != nil {
		surfacesJSON = []byte("[]")
	}

	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	if c.Status == "" {
		c.Status = domain.ToothStatusExisting
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM dental_conditions WHERE id = ?)", c.ID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	query := `
	INSERT INTO dental_conditions (
		id, patient_id, tooth_number, surfaces, ada_code, description, status, fee, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		tooth_number = excluded.tooth_number,
		surfaces = excluded.surfaces,
		ada_code = excluded.ada_code,
		description = excluded.description,
		status = excluded.status,
		fee = excluded.fee,
		updated_at = excluded.updated_at`

	_, err = tx.ExecContext(ctx, query,
		c.ID, c.PatientID, c.ToothNumber, string(surfacesJSON),
		c.ADACode, c.Description, string(c.Status), c.Fee,
		c.CreatedAt, c.UpdatedAt,
	)

	if err != nil {
		return false, fmt.Errorf("failed to save tooth condition: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return !exists, nil
}

func (r *ChartRepository) GetConditionByID(ctx context.Context, id string) (*domain.ToothCondition, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}

	query := `
	SELECT id, patient_id, tooth_number, surfaces, ada_code, description, status, fee, created_at, updated_at
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
	SELECT id, patient_id, tooth_number, surfaces, ada_code, description, status, fee, created_at, updated_at
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
