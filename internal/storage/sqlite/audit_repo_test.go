package sqlite_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
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

	logs, err := repo.Query(ctx, "pat_123", 10, "")
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
		got := pageAllAuditIDs(t, repo, patientID, nil)
		want := []string{"audit_d", "audit_c", "audit_b", "audit_a"}
		if !slices.Equal(got, want) {
			t.Errorf("Query(%q) paged order = %v, want %v", patientID, got, want)
		}
	}
}

// Entries logged while someone is paging must not shift later pages; with
// OFFSET paging each new entry repeated the last row of the previous page.
func TestAuditRepository_QueryCursorIsStableUnderNewEntries(t *testing.T) {
	db, err := sqlite.OpenAudit(filepath.Join(t.TempDir(), "test_audit.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite audit db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewAuditRepository(db)
	ctx := context.Background()

	base := time.Date(2026, 10, 22, 6, 0, 0, 0, time.UTC)
	logEntry := func(id string, ts time.Time) {
		t.Helper()
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
	for i, id := range []string{"audit_1", "audit_2", "audit_3", "audit_4"} {
		logEntry(id, base.Add(time.Duration(i)*time.Second))
	}

	newEntries := 0
	got := pageAllAuditIDs(t, repo, "pat_123", func() {
		newEntries++
		logEntry(fmt.Sprintf("audit_new_%d", newEntries), base.Add(time.Hour+time.Duration(newEntries)*time.Second))
	})
	want := []string{"audit_4", "audit_3", "audit_2", "audit_1"}
	if !slices.Equal(got, want) {
		t.Errorf("paged IDs with entries logged between pages = %v, want %v", got, want)
	}

	if _, err := repo.Query(ctx, "pat_123", 1, "audit_missing"); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Query with unknown cursor: got %v, want ErrInvalidInput", err)
	}
}

// pageAllAuditIDs walks the log one entry per page using the keyset cursor,
// calling betweenPages (if set) after each page.
func pageAllAuditIDs(t *testing.T, repo *sqlite.AuditRepository, patientID string, betweenPages func()) []string {
	t.Helper()
	var ids []string
	cursor := ""
	for {
		page, err := repo.Query(context.Background(), patientID, 1, cursor)
		if err != nil {
			t.Fatalf("Query(%q, before %q) failed: %v", patientID, cursor, err)
		}
		if len(page) == 0 {
			return ids
		}
		ids = append(ids, page[0].ID)
		cursor = page[0].ID
		if betweenPages != nil {
			betweenPages()
		}
	}
}
