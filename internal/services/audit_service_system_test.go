package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

// System logging lives in an internal test because it is unexported, so Wails can't bind it.
func TestAuditService_LogSystemPatientAction(t *testing.T) {
	auditDb, err := sqlite.OpenAudit(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatalf("Failed to open audit sqlite db: %v", err)
	}
	defer auditDb.Close()

	auditRepo := sqlite.NewAuditRepository(auditDb)
	// No provider repository and no session: system actions happen with nobody logged in.
	service := NewAuditService(auditRepo, nil)

	for _, id := range []string{"", "prov_1", "system:", "System:reminders", " system:reminders"} {
		err := service.logSystemPatientAction(id, domain.AuditActionCreate, "pat_1", "notification", "notif_1", "Sent reminder")
		if !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected actor ID %q to be rejected, got %v", id, err)
		}
	}

	ctx := context.Background()
	if logs, err := auditRepo.Query(ctx, "", 10, 0); err != nil || len(logs) != 0 {
		t.Fatalf("Expected rejected calls to write nothing, got %d entries, err %v", len(logs), err)
	}

	if err := service.logSystemPatientAction(domain.SystemActorReminders, domain.AuditActionCreate,
		"pat_1", "notification", "notif_1", "Sent 48-hour SMS reminder via aws_sms"); err != nil {
		t.Fatalf("logSystemPatientAction failed: %v", err)
	}

	logs, err := auditRepo.Query(ctx, "pat_1", 10, 0)
	if err != nil {
		t.Fatalf("Failed to query audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 audit entry, got %d", len(logs))
	}
	got := logs[0]
	if got.UserID != domain.SystemActorReminders || got.UserName != domain.SystemActorName {
		t.Errorf("Expected entry attributed to %q/%q, got %q/%q",
			domain.SystemActorReminders, domain.SystemActorName, got.UserID, got.UserName)
	}
	if got.PatientID != "pat_1" || got.Action != domain.AuditActionCreate || got.Resource != "notification" ||
		got.ResourceID != "notif_1" || got.Details != "Sent 48-hour SMS reminder via aws_sms" {
		t.Errorf("Unexpected entry contents: %+v", got)
	}
	if got.ID == "" || got.Timestamp.IsZero() {
		t.Errorf("Expected ID and timestamp to be set, got %+v", got)
	}
}
