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

// A cursor is the last entry of the previous page, so a cursor from another
// patient's log can never be valid for a patient-filtered query.
func TestAuditRepository_QueryRejectsCursorFromOtherPatient(t *testing.T) {
	db, err := sqlite.OpenAudit(filepath.Join(t.TempDir(), "test_audit.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite audit db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewAuditRepository(db)
	ctx := context.Background()

	base := time.Date(2026, 10, 22, 6, 0, 0, 0, time.UTC)
	for i, e := range []struct{ id, patientID string }{
		{"audit_a1", "pat_a"}, {"audit_b1", "pat_b"}, {"audit_a2", "pat_a"},
	} {
		err := repo.Log(ctx, &domain.AuditLogEntry{
			ID:        e.id,
			Timestamp: base.Add(time.Duration(i) * time.Second),
			UserID:    "usr_dr_smith",
			UserName:  "Dr. Jane Smith",
			PatientID: e.patientID,
			Action:    domain.AuditActionRead,
			Resource:  "patient_demographics",
		})
		if err != nil {
			t.Fatalf("Failed to log audit entry %s: %v", e.id, err)
		}
	}

	if _, err := repo.Query(ctx, "pat_a", 10, "audit_b1"); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Query(pat_a) with pat_b cursor: got %v, want ErrInvalidInput", err)
	}

	got, err := repo.Query(ctx, "pat_a", 10, "audit_a2")
	if err != nil {
		t.Fatalf("Query(pat_a) with own cursor failed: %v", err)
	}
	if len(got) != 1 || got[0].ID != "audit_a1" {
		t.Errorf("Query(pat_a, before audit_a2) = %v, want [audit_a1]", auditIDs(got))
	}

	// Unfiltered queries page the whole log, so any entry is a valid cursor.
	got, err = repo.Query(ctx, "", 10, "audit_b1")
	if err != nil {
		t.Fatalf("Query(all) with cursor failed: %v", err)
	}
	if len(got) != 1 || got[0].ID != "audit_a1" {
		t.Errorf("Query(all, before audit_b1) = %v, want [audit_a1]", auditIDs(got))
	}
}

func auditIDs(entries []*domain.AuditLogEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ID)
	}
	return ids
}

// pageAllAuditIDs walks the log one entry per page using the keyset cursor,
// calling betweenPages (if set) after each page.
func pageAllAuditIDs(t *testing.T, repo *sqlite.AuditRepository, patientID string, betweenPages func()) []string {
	t.Helper()
	// A cursor regression could otherwise loop forever: betweenPages keeps
	// logging fresh entries, so the log never runs out.
	const maxPages = 100
	var ids []string
	cursor := ""
	for range maxPages {
		page, err := repo.Query(context.Background(), patientID, 1, cursor)
		if err != nil {
			t.Fatalf("Query(%q, before %q) failed: %v", patientID, cursor, err)
		}
		if len(page) == 0 {
			return ids
		}
		if slices.Contains(ids, page[0].ID) {
			t.Fatalf("Query(%q, before %q) returned %q again; paged so far: %v", patientID, cursor, page[0].ID, ids)
		}
		ids = append(ids, page[0].ID)
		cursor = page[0].ID
		if betweenPages != nil {
			betweenPages()
		}
	}
	t.Fatalf("Query(%q) did not run out of entries after %d pages; paged: %v", patientID, maxPages, ids)
	return nil
}
