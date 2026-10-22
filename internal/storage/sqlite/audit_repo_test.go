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

	db, err := sqlite.OpenAudit(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite audit db: %v", err)
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

// Low-resolution clocks (notably Windows) can give consecutive entries the
// same timestamp; those must still come back newest-first and page stably.
func TestAuditRepository_QueryOrdersSameTimestampByInsertion(t *testing.T) {
	db, err := sqlite.OpenAudit(filepath.Join(t.TempDir(), "test_audit.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite audit db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewAuditRepository(db)
	ctx := context.Background()

	ts := time.Date(2026, 10, 22, 6, 24, 41, 67099600, time.UTC)
	ids := []string{"audit_a", "audit_b", "audit_c", "audit_d"}
	for _, id := range ids {
		err := repo.Log(ctx, &domain.AuditLogEntry{
			ID:        id,
			Timestamp: ts,
			UserID:    "usr_dr_smith",
			UserName:  "Dr. Jane Smith",
			PatientID: "pat_123",
			Action:    domain.AuditActionRead,
			Resource:  "patient_demographics",
		})
		if err != nil {
			t.Fatalf("Failed to log audit entry %s: %v", id, err)
		}
	}

	for _, patientID := range []string{"pat_123", ""} {
		var got []string
		for offset := 0; offset < len(ids); offset++ {
			page, err := repo.Query(ctx, patientID, 1, offset)
			if err != nil {
				t.Fatalf("Query(%q, offset %d) failed: %v", patientID, offset, err)
			}
			if len(page) != 1 {
				t.Fatalf("Query(%q, offset %d) returned %d entries, want 1", patientID, offset, len(page))
			}
			got = append(got, page[0].ID)
		}

		want := []string{"audit_d", "audit_c", "audit_b", "audit_a"}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("Query(%q) paged order = %v, want %v", patientID, got, want)
				break
			}
		}
	}
}
