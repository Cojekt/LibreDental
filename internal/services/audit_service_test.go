package services_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestAuditService(t *testing.T) {
	tempDir := t.TempDir()
	mainDbPath := filepath.Join(tempDir, "main.db")
	auditDbPath := filepath.Join(tempDir, "audit.db")

	auditDb, err := sqlite.OpenAudit(auditDbPath)
	if err != nil {
		t.Fatalf("Failed to open audit sqlite db: %v", err)
	}
	defer auditDb.Close()

	mainDb, err := sqlite.Open(mainDbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer mainDb.Close()

	auditRepo := sqlite.NewAuditRepository(auditDb)
	configRepo := sqlite.NewPracticeConfigRepository(mainDb)

	ctx := context.Background()
	err = configRepo.SaveProvider(ctx, &domain.Provider{
		ID:       "prov_1",
		Name:     "Test Prov",
		Pin:      "1234",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}

	service := services.NewAuditService(auditRepo, configRepo)

	// Session Tests
	token, err := service.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	user := service.GetSessionUser(token)
	if user == nil || user.ID != "prov_1" {
		t.Fatalf("Failed to get session user")
	}

	_, err = service.CreateSession("prov_1", "9999")
	if err == nil {
		t.Fatalf("Expected error for incorrect PIN")
	}

	service.DestroySession(token)
	if service.GetSessionUser(token) != nil {
		t.Fatalf("Expected session to be destroyed")
	}

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
		ID:        "audit_2",
		Timestamp: time.Now().UTC(),
		UserID:    "user_101",
		UserName:  "Dr. Smith",
		PatientID: "pat_123",
		Action:    domain.AuditActionUpdate,
		Resource:  "patient_demographics",
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
