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

// PaymentRepository implements storage.PaymentRepository for SQLite.
type PaymentRepository struct {
	db *DB
}

func NewPaymentRepository(db *DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	if p.ID == "" || p.PatientID == "" || p.Amount <= 0 || p.Date == "" {
		return fmt.Errorf("%w: ID, PatientID, Amount > 0, and Date are required", storage.ErrInvalidInput)
	}

	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.Method == "" {
		p.Method = domain.PaymentMethodCash
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payments (id, patient_id, claim_id, amount, method, date, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.PatientID, p.ClaimID, p.Amount,
		string(p.Method), p.Date, p.Notes, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}

	var p domain.Payment
	var methodStr string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, patient_id, claim_id, amount, method, date, notes, created_at
		FROM payments WHERE id = ?`, id,
	).Scan(&p.ID, &p.PatientID, &p.ClaimID, &p.Amount, &methodStr, &p.Date, &p.Notes, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan payment: %w", err)
	}

	p.Method = domain.PaymentMethod(methodStr)
	return &p, nil
}

func (r *PaymentRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}
	res, err := r.db.ExecContext(ctx, "DELETE FROM payments WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete payment: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *PaymentRepository) List(ctx context.Context, patientID string) ([]*domain.Payment, error) {
	query := `
		SELECT id, patient_id, claim_id, amount, method, date, notes, created_at
		FROM payments`

	var args []any
	if patientID != "" {
		query += " WHERE patient_id = ?"
		args = append(args, patientID)
	}
	query += " ORDER BY date DESC, created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		var methodStr string
		if err := rows.Scan(
			&p.ID, &p.PatientID, &p.ClaimID, &p.Amount,
			&methodStr, &p.Date, &p.Notes, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		p.Method = domain.PaymentMethod(methodStr)
		payments = append(payments, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if payments == nil {
		payments = []*domain.Payment{}
	}
	return payments, nil
}

// GetTotalPaid sums all payment amounts for a patient.
func (r *PaymentRepository) GetTotalPaid(ctx context.Context, patientID string) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE patient_id = ?",
		patientID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to compute total paid: %w", err)
	}
	return total, nil
}

// ListByDateRange returns payments within a specific date range (inclusive).
func (r *PaymentRepository) ListByDateRange(ctx context.Context, startDate, endDate string) ([]*domain.Payment, error) {
	query := `
		SELECT id, patient_id, claim_id, amount, method, date, notes, created_at
		FROM payments
		WHERE date >= ? AND date <= ?
		ORDER BY date DESC, created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments by date range: %w", err)
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		var methodStr string
		if err := rows.Scan(
			&p.ID, &p.PatientID, &p.ClaimID, &p.Amount,
			&methodStr, &p.Date, &p.Notes, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		p.Method = domain.PaymentMethod(methodStr)
		payments = append(payments, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if payments == nil {
		payments = []*domain.Payment{}
	}
	return payments, nil
}
