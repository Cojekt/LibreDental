package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type AppointmentRepository struct {
	db *DB
}

func NewAppointmentRepository(db *DB) *AppointmentRepository {
	return &AppointmentRepository{db: db}
}

func (r *AppointmentRepository) Create(ctx context.Context, a *domain.Appointment) error {
	if a.ID == "" {
		return fmt.Errorf("%w: appointment ID is required", storage.ErrInvalidInput)
	}
	if a.PatientID == "" {
		return fmt.Errorf("%w: patient ID is required", storage.ErrInvalidInput)
	}

	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	if a.Version == 0 {
		a.Version = 1
	}
	if a.Status == "" {
		a.Status = domain.AppointmentStatusScheduled
	}

	query := `
	INSERT INTO appointments (
		id, patient_id, provider_id, operatory_id,
		start_time, end_time, status, reason, color, notes,
		created_at, updated_at, version
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.PatientID, a.ProviderID, a.OperatoryID,
		a.StartTime.Format(time.RFC3339), a.EndTime.Format(time.RFC3339),
		string(a.Status), a.Reason, a.Color, a.Notes,
		a.CreatedAt, a.UpdatedAt, a.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to insert appointment: %w", err)
	}
	return nil
}

func (r *AppointmentRepository) GetByID(ctx context.Context, id string) (*domain.Appointment, error) {
	query := `
	SELECT id, patient_id, provider_id, operatory_id,
	       start_time, end_time, status, reason, color, notes,
	       created_at, updated_at, version
	FROM appointments WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanAppointment(row)
}

func (r *AppointmentRepository) Update(ctx context.Context, a *domain.Appointment) error {
	now := time.Now().UTC()

	query := `
	UPDATE appointments SET
		patient_id = ?, provider_id = ?, operatory_id = ?,
		start_time = ?, end_time = ?, status = ?,
		reason = ?, color = ?, notes = ?, updated_at = ?, version = version + 1
	WHERE id = ? AND version = ?`

	res, err := r.db.ExecContext(ctx, query,
		a.PatientID, a.ProviderID, a.OperatoryID,
		a.StartTime.Format(time.RFC3339), a.EndTime.Format(time.RFC3339), string(a.Status),
		a.Reason, a.Color, a.Notes, now,
		a.ID, a.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to update appointment: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return storage.ErrConflict
	}

	a.Version++
	a.UpdatedAt = now
	return nil
}

func (r *AppointmentRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM appointments WHERE id = ?", id)
	if err != nil {
		return err
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

func (r *AppointmentRepository) List(ctx context.Context, filter domain.AppointmentFilter) ([]*domain.Appointment, error) {
	var conditions []string
	var args []interface{}

	if filter.PatientID != "" {
		conditions = append(conditions, "patient_id = ?")
		args = append(args, filter.PatientID)
	}
	if filter.ProviderID != "" {
		conditions = append(conditions, "provider_id = ?")
		args = append(args, filter.ProviderID)
	}
	if filter.OperatoryID != "" {
		conditions = append(conditions, "operatory_id = ?")
		args = append(args, filter.OperatoryID)
	}
	if !filter.StartDate.IsZero() {
		conditions = append(conditions, "end_time >= ?")
		args = append(args, filter.StartDate.Format(time.RFC3339))
	}
	if !filter.EndDate.IsZero() {
		conditions = append(conditions, "start_time <= ?")
		args = append(args, filter.EndDate.Format(time.RFC3339))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	selectQuery := fmt.Sprintf(`
	SELECT id, patient_id, provider_id, operatory_id,
	       start_time, end_time, status, reason, color, notes,
	       created_at, updated_at, version
	FROM appointments %s ORDER BY start_time ASC`, whereClause)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query appointments: %w", err)
	}
	defer rows.Close()

	var appointments []*domain.Appointment
	for rows.Next() {
		a, err := scanAppointment(rows)
		if err != nil {
			return nil, err
		}
		appointments = append(appointments, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return appointments, nil
}

func scanAppointment(scanner scannable) (*domain.Appointment, error) {
	var a domain.Appointment
	var startTimeStr, endTimeStr, statusStr string

	err := scanner.Scan(
		&a.ID, &a.PatientID, &a.ProviderID, &a.OperatoryID,
		&startTimeStr, &endTimeStr, &statusStr, &a.Reason, &a.Color, &a.Notes,
		&a.CreatedAt, &a.UpdatedAt, &a.Version,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}

	a.Status = domain.AppointmentStatus(statusStr)
	a.StartTime, _ = time.Parse(time.RFC3339, startTimeStr)
	a.EndTime, _ = time.Parse(time.RFC3339, endTimeStr)

	return &a, nil
}
