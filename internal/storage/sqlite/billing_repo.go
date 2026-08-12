package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// ClaimRepository implements storage.ClaimRepository for SQLite.
type ClaimRepository struct {
	db *DB
}

func NewClaimRepository(db *DB) *ClaimRepository {
	return &ClaimRepository{db: db}
}

func (r *ClaimRepository) Create(ctx context.Context, c *domain.Claim) error {
	if c.ID == "" || c.PatientID == "" || c.DateOfService == "" {
		return fmt.Errorf("%w: ID, PatientID, and DateOfService are required", storage.ErrInvalidInput)
	}

	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	if c.Status == "" {
		c.Status = domain.ClaimStatusDraft
	}
	if c.LineItems == nil {
		c.LineItems = []domain.ClaimLineItem{}
	}

	lineItemsJSON, err := json.Marshal(c.LineItems)
	if err != nil {
		lineItemsJSON = []byte("[]")
	}

	_, err = r.db.ExecContext(
		ctx, `
		INSERT INTO claims (
			id, patient_id, provider_id, appointment_id,
			insurance_carrier, policy_number, group_number,
			date_of_service, status, notes, line_items,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.PatientID, c.ProviderID, c.AppointmentID,
		c.InsuranceCarrier, c.PolicyNumber, c.GroupNumber,
		c.DateOfService, string(c.Status), c.Notes, string(lineItemsJSON),
		c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create claim: %w", err)
	}
	return nil
}

func (r *ClaimRepository) GetByID(ctx context.Context, id string) (*domain.Claim, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT id, patient_id, provider_id, appointment_id,
			insurance_carrier, policy_number, group_number,
			date_of_service, status, notes, line_items, created_at, updated_at
		FROM claims WHERE id = ?`, id)

	c, err := scanClaim(row)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *ClaimRepository) Update(ctx context.Context, c *domain.Claim) error {
	if c.ID == "" {
		return fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}

	c.UpdatedAt = time.Now().UTC()

	if c.LineItems == nil {
		c.LineItems = []domain.ClaimLineItem{}
	}
	lineItemsJSON, err := json.Marshal(c.LineItems)
	if err != nil {
		lineItemsJSON = []byte("[]")
	}

	res, err := r.db.ExecContext(
		ctx, `
		UPDATE claims SET
			patient_id = ?, provider_id = ?, appointment_id = ?,
			insurance_carrier = ?, policy_number = ?, group_number = ?,
			date_of_service = ?, status = ?, notes = ?, line_items = ?,
			updated_at = ?
		WHERE id = ?`,
		c.PatientID, c.ProviderID, c.AppointmentID,
		c.InsuranceCarrier, c.PolicyNumber, c.GroupNumber,
		c.DateOfService, string(c.Status), c.Notes, string(lineItemsJSON),
		c.UpdatedAt, c.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update claim: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *ClaimRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}
	res, err := r.db.ExecContext(ctx, "DELETE FROM claims WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete claim: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *ClaimRepository) List(ctx context.Context, patientID string) ([]*domain.Claim, error) {
	query := `
		SELECT id, patient_id, provider_id, appointment_id,
			insurance_carrier, policy_number, group_number,
			date_of_service, status, notes, line_items, created_at, updated_at
		FROM claims`

	var args []any
	if patientID != "" {
		query += " WHERE patient_id = ?"
		args = append(args, patientID)
	}
	query += " ORDER BY date_of_service DESC, created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list claims: %w", err)
	}
	defer rows.Close()

	var claims []*domain.Claim
	for rows.Next() {
		c, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		claims = append(claims, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if claims == nil {
		claims = []*domain.Claim{}
	}
	return claims, nil
}

// GetTotalBilled computes the sum of all line item fees across all claims for a patient.
// Line items are stored as JSON, so we use a JSON aggregation approach: we decode in-app.
func (r *ClaimRepository) GetTotalBilled(ctx context.Context, patientID string) (float64, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT line_items FROM claims WHERE patient_id = ?", patientID)
	if err != nil {
		return 0, fmt.Errorf("failed to query claim line items: %w", err)
	}
	defer rows.Close()

	var total float64
	for rows.Next() {
		var lineItemsJSON string
		if err := rows.Scan(&lineItemsJSON); err != nil {
			return 0, err
		}
		var items []domain.ClaimLineItem
		if err := json.Unmarshal([]byte(lineItemsJSON), &items); err != nil {
			continue
		}
		for _, item := range items {
			total += item.Fee
		}
	}
	return total, rows.Err()
}

func scanClaim(row rowScanner) (*domain.Claim, error) {
	var c domain.Claim
	var statusStr, lineItemsJSON string

	err := row.Scan(
		&c.ID, &c.PatientID, &c.ProviderID, &c.AppointmentID,
		&c.InsuranceCarrier, &c.PolicyNumber, &c.GroupNumber,
		&c.DateOfService, &statusStr, &c.Notes, &lineItemsJSON,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w", storage.ErrNotFound)
	}

	c.Status = domain.ClaimStatus(statusStr)
	if err := json.Unmarshal([]byte(lineItemsJSON), &c.LineItems); err != nil {
		c.LineItems = []domain.ClaimLineItem{}
	}
	if c.LineItems == nil {
		c.LineItems = []domain.ClaimLineItem{}
	}
	return &c, nil
}
