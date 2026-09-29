package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// NotificationRepository implements storage.NotificationLogRepository for SQLite.
type NotificationRepository struct {
	db *DB
}

func NewNotificationRepository(db *DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, entry *domain.NotificationLog) error {
	if entry.ID == "" || entry.PatientID == "" || entry.Recipient == "" {
		return fmt.Errorf("%w: ID, PatientID, and Recipient are required", storage.ErrInvalidInput)
	}
	if entry.Channel == "" || entry.Status == "" || entry.ProviderName == "" {
		return fmt.Errorf("%w: Channel, Status, and ProviderName are required", storage.ErrInvalidInput)
	}

	if entry.SentAt.IsZero() {
		entry.SentAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(
		ctx, `
		INSERT INTO notification_log (
			id, patient_id, appointment_id, channel, provider_name,
			external_message_id, recipient, subject, body, status, error_message, sent_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.PatientID, nullIfEmpty(entry.AppointmentID), string(entry.Channel), entry.ProviderName,
		entry.ExternalMessageID, entry.Recipient, entry.Subject, entry.Body, string(entry.Status), entry.ErrorMessage, entry.SentAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification log entry: %w", err)
	}
	return nil
}

func (r *NotificationRepository) List(ctx context.Context, patientID string, limit, offset int) ([]*domain.NotificationLog, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, patient_id, appointment_id, channel, provider_name,
			external_message_id, recipient, subject, body, status, error_message, sent_at
		FROM notification_log`

	var args []any
	if patientID != "" {
		query += " WHERE patient_id = ?"
		args = append(args, patientID)
	}
	query += " ORDER BY sent_at DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list notification log: %w", err)
	}
	defer rows.Close()

	entries, err := scanNotificationLogs(rows)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *NotificationRepository) ListByAppointment(ctx context.Context, appointmentID string) ([]*domain.NotificationLog, error) {
	if appointmentID == "" {
		return nil, fmt.Errorf("%w: appointment ID is required", storage.ErrInvalidInput)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, patient_id, appointment_id, channel, provider_name,
			external_message_id, recipient, subject, body, status, error_message, sent_at
		FROM notification_log WHERE appointment_id = ? ORDER BY sent_at DESC, id DESC`, appointmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list notification log for appointment: %w", err)
	}
	defer rows.Close()

	entries, err := scanNotificationLogs(rows)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func scanNotificationLogs(rows *sql.Rows) ([]*domain.NotificationLog, error) {
	var entries []*domain.NotificationLog
	for rows.Next() {
		var (
			entry             domain.NotificationLog
			appointmentID     sql.NullString
			externalMessageID sql.NullString
			subject           sql.NullString
			errorMessage      sql.NullString
			channel           string
			status            string
		)
		if err := rows.Scan(
			&entry.ID, &entry.PatientID, &appointmentID, &channel, &entry.ProviderName,
			&externalMessageID, &entry.Recipient, &subject, &entry.Body, &status, &errorMessage, &entry.SentAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan notification log entry: %w", err)
		}
		entry.AppointmentID = appointmentID.String
		entry.ExternalMessageID = externalMessageID.String
		entry.Subject = subject.String
		entry.ErrorMessage = errorMessage.String
		entry.Channel = domain.NotificationChannel(channel)
		entry.Status = domain.NotificationStatus(status)
		entries = append(entries, &entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []*domain.NotificationLog{}
	}
	return entries, nil
}
