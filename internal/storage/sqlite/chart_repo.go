package sqlite

import (
	"context"
	"encoding/json"
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
		var c domain.ToothCondition
		var surfacesJSON string
		var statusStr string

		err := rows.Scan(
			&c.ID, &c.PatientID, &c.ToothNumber, &surfacesJSON,
			&c.ADACode, &c.Description, &statusStr, &c.Fee,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tooth condition: %w", err)
		}

		c.Status = domain.ToothStatus(statusStr)
		if len(surfacesJSON) > 0 {
			json.Unmarshal([]byte(surfacesJSON), &c.Surfaces)
		}
		if c.Surfaces == nil {
			c.Surfaces = []domain.ToothSurface{}
		}

		if c.UpdatedAt.After(latestUpdate) {
			latestUpdate = c.UpdatedAt
		}

		conditions = append(conditions, c)
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

func (r *ChartRepository) SaveCondition(ctx context.Context, c *domain.ToothCondition) error {
	if c.ID == "" || c.PatientID == "" || c.ToothNumber <= 0 {
		return fmt.Errorf("%w: ID, PatientID, and valid ToothNumber are required", storage.ErrInvalidInput)
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

	_, err = r.db.ExecContext(ctx, query,
		c.ID, c.PatientID, c.ToothNumber, string(surfacesJSON),
		c.ADACode, c.Description, string(c.Status), c.Fee,
		c.CreatedAt, c.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save tooth condition: %w", err)
	}

	return nil
}

func (r *ChartRepository) DeleteCondition(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}

	res, err := r.db.ExecContext(ctx, "DELETE FROM dental_conditions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete tooth condition: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return storage.ErrNotFound
	}

	return nil
}
