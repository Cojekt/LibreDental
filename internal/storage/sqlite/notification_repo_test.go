package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestNotificationRepository(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_notification_repo.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	patientRepo := sqlite.NewPatientRepository(db)
	patient := &domain.Patient{ID: "pat_notif_1", FirstName: "Jane", LastName: "Doe"}
	if err := patientRepo.Create(ctx, patient); err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	configRepo := sqlite.NewPracticeConfigRepository(db)
	if err := configRepo.SaveProvider(ctx, &domain.Provider{ID: "prov_notif_1", Name: "Dr Test", IsActive: true}); err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	if err := configRepo.SaveOperatory(ctx, &domain.Operatory{ID: "op_notif_1", Name: "Chair 1", IsActive: true}); err != nil {
		t.Fatalf("Failed to create operatory: %v", err)
	}

	appointmentRepo := sqlite.NewAppointmentRepository(db)
	appt := &domain.Appointment{
		ID:          "appt_notif_1",
		PatientID:   "pat_notif_1",
		ProviderID:  "prov_notif_1",
		OperatoryID: "op_notif_1",
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(time.Hour),
		Status:      domain.AppointmentStatusScheduled,
	}
	if err := appointmentRepo.Create(ctx, appt); err != nil {
		t.Fatalf("Failed to create appointment: %v", err)
	}

	repo := sqlite.NewNotificationRepository(db)

	// Input validation
	valid := domain.NotificationLog{
		ID:           "notif_invalid",
		PatientID:    "pat_notif_1",
		Channel:      domain.NotificationChannelEmail,
		ProviderName: "mock_email",
		Recipient:    "x@example.com",
		Status:       domain.NotificationStatusSent,
	}
	invalidCases := map[string]func(e *domain.NotificationLog){
		"ID":           func(e *domain.NotificationLog) { e.ID = "" },
		"PatientID":    func(e *domain.NotificationLog) { e.PatientID = "" },
		"Recipient":    func(e *domain.NotificationLog) { e.Recipient = "" },
		"Channel":      func(e *domain.NotificationLog) { e.Channel = "" },
		"Status":       func(e *domain.NotificationLog) { e.Status = "" },
		"ProviderName": func(e *domain.NotificationLog) { e.ProviderName = "" },
	}
	for field, mutate := range invalidCases {
		e := valid
		mutate(&e)
		if err := repo.Create(ctx, &e); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected ErrInvalidInput when %s is empty, got: %v", field, err)
		}
	}

	now := time.Now().UTC().Truncate(time.Second)

	entry1 := &domain.NotificationLog{
		ID:                "notif_1",
		PatientID:         "pat_notif_1",
		AppointmentID:     "appt_notif_1",
		Channel:           domain.NotificationChannelEmail,
		ProviderName:      "mock_email",
		Recipient:         "jane@example.com",
		Subject:           "Appointment reminder",
		Body:              "See you tomorrow at 9am",
		Status:            domain.NotificationStatusSent,
		ExternalMessageID: "ext_msg_1",
		SentAt:            now.Add(-time.Minute),
	}
	if err := repo.Create(ctx, entry1); err != nil {
		t.Fatalf("Failed to create notification log entry: %v", err)
	}

	entry2 := &domain.NotificationLog{
		ID:           "notif_2",
		PatientID:    "pat_notif_1",
		Channel:      domain.NotificationChannelSMS,
		ProviderName: "mock_sms",
		Recipient:    "+15555550123",
		Body:         "Reminder: appointment tomorrow",
		Status:       domain.NotificationStatusFailed,
		ErrorMessage: "carrier rejected message",
		SentAt:       now,
	}
	if err := repo.Create(ctx, entry2); err != nil {
		t.Fatalf("Failed to create notification log entry without appointment: %v", err)
	}

	list, err := repo.List(ctx, "pat_notif_1", 0, 0)
	if err != nil {
		t.Fatalf("Failed to list notification log: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Expected 2 notification log entries, got %d", len(list))
	}
	if got := list[0]; got.ID != "notif_2" || got.Channel != domain.NotificationChannelSMS ||
		got.Status != domain.NotificationStatusFailed || got.ErrorMessage != "carrier rejected message" ||
		got.AppointmentID != "" || got.ExternalMessageID != "" {
		t.Errorf("Unexpected most recent entry: %+v", got)
	}
	if got := list[1]; got.ID != "notif_1" || got.Channel != domain.NotificationChannelEmail ||
		got.Status != domain.NotificationStatusSent || got.Subject != "Appointment reminder" ||
		got.AppointmentID != "appt_notif_1" || got.ExternalMessageID != "ext_msg_1" || got.ProviderName != "mock_email" {
		t.Errorf("Unexpected oldest entry: %+v", got)
	}

	byAppointment, err := repo.ListByAppointment(ctx, "appt_notif_1")
	if err != nil {
		t.Fatalf("Failed to list notification log by appointment: %v", err)
	}
	if len(byAppointment) != 1 || byAppointment[0].ID != "notif_1" {
		t.Fatalf("Expected 1 entry for appointment, got %+v", byAppointment)
	}

	// Nullable columns written as NULL (e.g. by an older client or manual repair) must still scan.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO notification_log (id, patient_id, channel, provider_name, external_message_id,
			recipient, subject, body, status, error_message, sent_at)
		VALUES ('notif_null', 'pat_notif_1', 'sms', 'mock_sms', NULL, '+15555550123', NULL, 'Hi', 'sent', NULL, ?)`,
		now.Add(-time.Hour)); err != nil {
		t.Fatalf("Failed to insert notification log entry with NULL columns: %v", err)
	}
	withNulls, err := repo.List(ctx, "pat_notif_1", 0, 0)
	if err != nil {
		t.Fatalf("Failed to list notification log with NULL columns: %v", err)
	}
	if len(withNulls) != 3 || withNulls[2].ID != "notif_null" || withNulls[2].Subject != "" || withNulls[2].ErrorMessage != "" {
		t.Errorf("Unexpected entries after NULL insert: %+v", withNulls)
	}

	if _, err := repo.ListByAppointment(ctx, ""); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for empty appointment ID, got: %v", err)
	}

	unrelated, err := repo.List(ctx, "pat_no_notifications", 0, 0)
	if err != nil {
		t.Fatalf("Failed to list notification log for unrelated patient: %v", err)
	}
	if len(unrelated) != 0 {
		t.Errorf("Expected 0 notification log entries for unrelated patient, got %d", len(unrelated))
	}
}
