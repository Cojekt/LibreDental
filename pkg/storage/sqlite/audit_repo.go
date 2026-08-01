package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
)

type AuditRepository struct {
	db *DB
}

func NewAuditRepository(db *DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Log(ctx context.Context, entry *domain.AuditLogEntry) error {
	if entry.ID == "" {
		return fmt.Errorf("audit log ID is required")
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	query := `
	INSERT INTO audit_logs (
		id, timestamp, user_id, user_name, patient_id, action, resource, resource_id, details, ip_address
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.Timestamp, entry.UserID, entry.UserName, entry.PatientID,
		entry.Action, entry.Resource, entry.ResourceID, entry.Details, entry.IPAddress,
	)

	if err != nil {
		return fmt.Errorf("failed to log HIPAA audit entry: %w", err)
	}
	return nil
}

func (r *AuditRepository) Query(ctx context.Context, patientID string, limit int, offset int) ([]*domain.AuditLogEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	var query string
	var args []interface{}

	if patientID != "" {
		query = `SELECT id, timestamp, user_id, user_name, patient_id, action, resource, resource_id, details, ip_address
		         FROM audit_logs WHERE patient_id = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?`
		args = append(args, patientID, limit, offset)
	} else {
		query = `SELECT id, timestamp, user_id, user_name, patient_id, action, resource, resource_id, details, ip_address
		         FROM audit_logs ORDER BY timestamp DESC LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*domain.AuditLogEntry
	for rows.Next() {
		var e domain.AuditLogEntry
		var actionStr string
		err := rows.Scan(
			&e.ID, &e.Timestamp, &e.UserID, &e.UserName, &e.PatientID,
			&actionStr, &e.Resource, &e.ResourceID, &e.Details, &e.IPAddress,
		)
		if err != nil {
			return nil, err
		}
		e.Action = domain.AuditAction(actionStr)
		entries = append(entries, &e)
	}

	return entries, nil
}
