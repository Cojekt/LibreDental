package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestAuditRepository_LogAndQuery(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_audit.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewAuditRepository(db)
	ctx := context.Background()

	entry := &domain.AuditLogEntry{
		ID:         "audit_001",
		Timestamp:  time.Now().UTC(),
		UserID:     "usr_dr_smith",
		UserName:   "Dr. Jane Smith",
		PatientID:  "pat_123",
		Action:     domain.AuditActionRead,
		Resource:   "patient_demographics",
		ResourceID: "pat_123",
		Details:    "Viewed patient medical alerts",
		IPAddress:  "127.0.0.1",
	}

	err = repo.Log(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to log audit entry: %v", err)
	}

	logs, err := repo.Query(ctx, "pat_123", 10, 0)
	if err != nil {
		t.Fatalf("Failed to query audit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 audit record, got %d", len(logs))
	}
	if logs[0].UserID != "usr_dr_smith" || logs[0].Action != domain.AuditActionRead {
		t.Errorf("Unexpected log content: %+v", logs[0])
	}
}
