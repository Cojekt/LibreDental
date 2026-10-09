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
			external_message_id, recipient, subject, body, status, error_message, sent_at,
			reminder_kind, appointment_start
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.PatientID, nullIfEmpty(entry.AppointmentID), string(entry.Channel), entry.ProviderName,
		entry.ExternalMessageID, entry.Recipient, entry.Subject, entry.Body, string(entry.Status), entry.ErrorMessage, entry.SentAt,
		nullIfEmpty(entry.ReminderKind), appointmentStartValue(entry.AppointmentStart),
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
			external_message_id, recipient, subject, body, status, error_message, sent_at,
			reminder_kind, appointment_start
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
			external_message_id, recipient, subject, body, status, error_message, sent_at,
			reminder_kind, appointment_start
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
			reminderKind      sql.NullString
			appointmentStart  sql.NullString
			channel           string
			status            string
		)
		if err := rows.Scan(
			&entry.ID, &entry.PatientID, &appointmentID, &channel, &entry.ProviderName,
			&externalMessageID, &entry.Recipient, &subject, &entry.Body, &status, &errorMessage, &entry.SentAt,
			&reminderKind, &appointmentStart,
		); err != nil {
			return nil, fmt.Errorf("failed to scan notification log entry: %w", err)
		}
		entry.AppointmentID = appointmentID.String
		entry.ExternalMessageID = externalMessageID.String
		entry.Subject = subject.String
		entry.ErrorMessage = errorMessage.String
		entry.ReminderKind = reminderKind.String
		if appointmentStart.Valid {
			if t, err := time.Parse(time.RFC3339, appointmentStart.String); err == nil {
				entry.AppointmentStart = &t
			}
		}
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

// appointmentStartValue stores an appointment start the same way the appointments table does
// (RFC 3339, UTC), so the reminder key matches across reads and writes.
func appointmentStartValue(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func validateReminderEntry(entry *domain.NotificationLog) error {
	if entry.ID == "" || entry.PatientID == "" || entry.AppointmentID == "" || entry.Recipient == "" {
		return fmt.Errorf("%w: ID, PatientID, AppointmentID, and Recipient are required", storage.ErrInvalidInput)
	}
	if entry.Channel == "" || entry.ProviderName == "" || entry.ReminderKind == "" || entry.AppointmentStart == nil {
		return fmt.Errorf("%w: Channel, ProviderName, ReminderKind, and AppointmentStart are required", storage.ErrInvalidInput)
	}
	if entry.SentAt.IsZero() {
		entry.SentAt = time.Now().UTC()
	}
	entry.SentAt = entry.SentAt.UTC()
	return nil
}

// Rows that count toward a patient's reminder limit: anything that went out, or may have.
const countedStatuses = "('sent', 'pending', 'unknown')"

func (r *NotificationRepository) ClaimReminder(ctx context.Context, entry *domain.NotificationLog, limit *domain.ReminderLimit) (domain.ReminderClaim, error) {
	if err := validateReminderEntry(entry); err != nil {
		return "", err
	}
	entry.Status = domain.NotificationStatusPending
	entry.ErrorMessage = ""

	// The driver writes times with time.Time.String, so sent_at only sorts correctly as text
	// when every value is UTC with no monotonic reading. UTC() guarantees both (a time.Now()
	// passed directly would be written with a trailing "m=+..." and break the comparison).
	checkLimit := 0
	var dayStart, weekStart time.Time
	var perDay, perWeek int
	if limit != nil {
		checkLimit = 1
		dayStart, weekStart = limit.DayStart.UTC(), limit.WeekStart.UTC()
		perDay, perWeek = limit.PerDay, limit.PerWeek
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO notification_log (
			id, patient_id, appointment_id, channel, provider_name,
			external_message_id, recipient, subject, body, status, error_message, sent_at,
			reminder_kind, appointment_start
		)
		SELECT ?, ?, ?, ?, ?, '', ?, ?, ?, ?, '', ?, ?, ?
		WHERE ? = 0 OR (
			(SELECT COUNT(*) FROM notification_log
			 WHERE patient_id = ? AND channel = ? AND status IN `+countedStatuses+` AND sent_at >= ?) < ?
			AND
			(SELECT COUNT(*) FROM notification_log
			 WHERE patient_id = ? AND channel = ? AND status IN `+countedStatuses+` AND sent_at >= ?) < ?
		)
		ON CONFLICT (appointment_id, appointment_start, reminder_kind, channel) WHERE reminder_kind IS NOT NULL
		DO NOTHING`,
		entry.ID, entry.PatientID, entry.AppointmentID, string(entry.Channel), entry.ProviderName,
		entry.Recipient, entry.Subject, entry.Body, string(entry.Status), entry.SentAt,
		entry.ReminderKind, appointmentStartValue(entry.AppointmentStart),
		checkLimit,
		entry.PatientID, string(entry.Channel), dayStart, perDay,
		entry.PatientID, string(entry.Channel), weekStart, perWeek,
	)
	if err != nil {
		return "", fmt.Errorf("failed to claim reminder: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return "", err
	}
	if n == 1 {
		return domain.ReminderClaimed, nil
	}

	exists, err := r.reminderExists(ctx, entry)
	if err != nil {
		return "", err
	}
	if exists {
		return domain.ReminderAlreadyClaimed, nil
	}
	return domain.ReminderLimitReached, nil
}

func (r *NotificationRepository) reminderExists(ctx context.Context, entry *domain.NotificationLog) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notification_log
		WHERE appointment_id = ? AND appointment_start = ? AND reminder_kind = ? AND channel = ?`,
		entry.AppointmentID, appointmentStartValue(entry.AppointmentStart), entry.ReminderKind, string(entry.Channel),
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("failed to check reminder: %w", err)
	}
	return n > 0, nil
}

func (r *NotificationRepository) RecordSkipped(ctx context.Context, entry *domain.NotificationLog) error {
	if err := validateReminderEntry(entry); err != nil {
		return err
	}
	entry.Status = domain.NotificationStatusSkipped
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notification_log (
			id, patient_id, appointment_id, channel, provider_name,
			external_message_id, recipient, subject, body, status, error_message, sent_at,
			reminder_kind, appointment_start
		) VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (appointment_id, appointment_start, reminder_kind, channel) WHERE reminder_kind IS NOT NULL
		DO NOTHING`,
		entry.ID, entry.PatientID, entry.AppointmentID, string(entry.Channel), entry.ProviderName,
		entry.Recipient, entry.Subject, entry.Body, string(entry.Status), entry.ErrorMessage, entry.SentAt,
		entry.ReminderKind, appointmentStartValue(entry.AppointmentStart),
	)
	if err != nil {
		return fmt.Errorf("failed to record skipped reminder: %w", err)
	}
	return nil
}

func (r *NotificationRepository) UpdateResult(ctx context.Context, id string, status domain.NotificationStatus, externalMessageID, errorMessage string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE notification_log SET status = ?, external_message_id = ?, error_message = ?
		WHERE id = ?`,
		string(status), externalMessageID, errorMessage, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update notification result: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkStalePending(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE notification_log
		SET status = 'unknown', error_message = 'Sending was interrupted; the message may or may not have been delivered.'
		WHERE status = 'pending' AND sent_at < ?`,
		cutoff.UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to mark interrupted reminders: %w", err)
	}
	return res.RowsAffected()
}
