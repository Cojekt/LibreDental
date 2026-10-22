package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
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

// Query returns audit entries newest first. beforeID is a keyset cursor (the ID
// of the last entry of the previous page): when set, only entries older than it
// are returned, so entries logged while someone pages through the log cannot
// shift later pages and cause duplicated or skipped rows.
func (r *AuditRepository) Query(ctx context.Context, patientID string, limit int, beforeID string) ([]*domain.AuditLogEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	var conditions []string
	var args []interface{}

	if patientID != "" {
		conditions = append(conditions, "patient_id = ?")
		args = append(args, patientID)
	}

	if beforeID != "" {
		// A cursor is the last entry of the previous page, so when filtering by
		// patient it must belong to that patient's log.
		cursorQuery := `SELECT 1 FROM audit_logs WHERE id = ?`
		cursorArgs := []interface{}{beforeID}
		if patientID != "" {
			cursorQuery += ` AND patient_id = ?`
			cursorArgs = append(cursorArgs, patientID)
		}
		var exists int
		err := r.db.QueryRowContext(ctx, cursorQuery, cursorArgs...).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: unknown audit log cursor %q", storage.ErrInvalidInput, beforeID)
		}
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, "(timestamp, rowid) < (SELECT timestamp, rowid FROM audit_logs WHERE id = ?)")
		args = append(args, beforeID)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// rowid breaks ties between entries logged within the same clock tick.
	query := fmt.Sprintf(`SELECT id, timestamp, user_id, user_name, patient_id, action, resource, resource_id, details, ip_address
		FROM audit_logs %s ORDER BY timestamp DESC, rowid DESC LIMIT ?`, where)
	args = append(args, limit)

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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
