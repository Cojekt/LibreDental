package services_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestAuditService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_audit_service.db")

	auditDb, err := sqlite.OpenAudit(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite audit db: %v", err)
	}
	defer auditDb.Close()

	auditRepo := sqlite.NewAuditRepository(auditDb)
	service := services.NewAuditService(auditRepo)

	// 1. Log an Event
	entry := &domain.AuditLogEntry{
		ID:         "audit_1",
		Timestamp:  time.Now().UTC(),
		UserID:     "user_101",
		UserName:   "Dr. Smith",
		PatientID:  "pat_123",
		Action:     domain.AuditActionRead,
		Resource:   "dental_chart",
		ResourceID: "chart_123",
		Details:    "Viewed patient chart",
		IPAddress:  "127.0.0.1",
	}

	err = service.LogEvent(entry)
	if err != nil {
		t.Fatalf("Failed to log audit event: %v", err)
	}

	// 2. Fetch Logs
	logs, err := service.GetAuditLogs("pat_123", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	fetched := logs[0]
	if fetched.ID != "audit_1" {
		t.Errorf("Expected ID 'audit_1', got '%s'", fetched.ID)
	}
	if fetched.UserName != "Dr. Smith" {
		t.Errorf("Expected UserName 'Dr. Smith', got '%s'", fetched.UserName)
	}

	// 3. Fetch Logs with no Patient ID (should return all)
	allLogs, err := service.GetAuditLogs("", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get all audit logs: %v", err)
	}
	if len(allLogs) != 1 {
		t.Errorf("Expected 1 total log entry, got %d", len(allLogs))
	}

	// 4. Test Pagination
	// Insert a second event
	entry2 := &domain.AuditLogEntry{
		ID:         "audit_2",
		Timestamp:  time.Now().UTC(),
		UserID:     "user_101",
		UserName:   "Dr. Smith",
		PatientID:  "pat_123",
		Action:     domain.AuditActionUpdate,
		Resource:   "patient_demographics",
	}
	err = service.LogEvent(entry2)
	if err != nil {
		t.Fatalf("Failed to log second audit event: %v", err)
	}

	paginatedLogs, err := service.GetAuditLogs("pat_123", 1, 0) // Limit 1
	if err != nil {
		t.Fatalf("Failed to get paginated logs: %v", err)
	}
	if len(paginatedLogs) != 1 {
		t.Errorf("Expected 1 paginated log entry, got %d", len(paginatedLogs))
	}

	paginatedLogsOffset, err := service.GetAuditLogs("pat_123", 1, 1) // Offset 1
	if err != nil {
		t.Fatalf("Failed to get paginated offset logs: %v", err)
	}
	if len(paginatedLogsOffset) != 1 {
		t.Errorf("Expected 1 paginated log entry, got %d", len(paginatedLogsOffset))
	}

	if paginatedLogs[0].ID == paginatedLogsOffset[0].ID {
		t.Errorf("Pagination offset returned the same row")
	}
}
