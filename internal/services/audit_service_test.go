package services_test

import (
	"context"
	"path/filepath"
	"reflect"
	"slices"
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

	err = auditRepo.Log(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to log audit event: %v", err)
	}

	// Re-authenticate since the earlier session was destroyed above.
	readToken, err := service.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if _, err := service.GetAuditLogs("bogus-token", "pat_123", 10, 0); err != services.ErrUnauthorized {
		t.Fatalf("Expected ErrUnauthorized for unauthenticated call, got %v", err)
	}

	// 2. Fetch Logs
	logs, err := service.GetAuditLogs(readToken, "pat_123", 10, 0)
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
	allLogs, err := service.GetAuditLogs(readToken, "", 10, 0)
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
	err = auditRepo.Log(ctx, entry2)
	if err != nil {
		t.Fatalf("Failed to log second audit event: %v", err)
	}

	paginatedLogs, err := service.GetAuditLogs(readToken, "pat_123", 1, 0) // Limit 1
	if err != nil {
		t.Fatalf("Failed to get paginated logs: %v", err)
	}
	if len(paginatedLogs) != 1 {
		t.Errorf("Expected 1 paginated log entry, got %d", len(paginatedLogs))
	}

	paginatedLogsOffset, err := service.GetAuditLogs(readToken, "pat_123", 1, 1) // Offset 1
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

func TestAuditService_LogPatientActionUsesSessionUser(t *testing.T) {
	tempDir := t.TempDir()
	auditDb, err := sqlite.OpenAudit(filepath.Join(tempDir, "audit.db"))
	if err != nil {
		t.Fatalf("Failed to open audit sqlite db: %v", err)
	}
	defer auditDb.Close()
	mainDb, err := sqlite.Open(filepath.Join(tempDir, "main.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer mainDb.Close()

	configRepo := sqlite.NewPracticeConfigRepository(mainDb)
	if err := configRepo.SaveProvider(context.Background(), &domain.Provider{ID: "prov_1", Name: "Test Prov", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	service := services.NewAuditService(sqlite.NewAuditRepository(auditDb), configRepo)

	if err := service.LogPatientAction("bogus-token", domain.AuditActionRead, "pat_9", "dental_chart", "Viewed chart"); err != services.ErrUnauthorized {
		t.Fatalf("Expected ErrUnauthorized without a session, got %v", err)
	}

	token, err := service.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if err := service.LogPatientAction(token, domain.AuditActionRead, "pat_9", "dental_chart", "Viewed chart"); err != nil {
		t.Fatalf("LogPatientAction failed: %v", err)
	}

	logs, err := service.GetAuditLogs(token, "pat_9", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 audit entry for pat_9, got %d", len(logs))
	}
	if logs[0].UserID != "prov_1" || logs[0].UserName != "Test Prov" {
		t.Errorf("Expected entry attributed to the session user, got %q/%q", logs[0].UserID, logs[0].UserName)
	}
	if logs[0].Resource != "dental_chart" || logs[0].Action != domain.AuditActionRead {
		t.Errorf("Unexpected entry contents: %+v", logs[0])
	}
}

// Wails binds every exported AuditService method for the frontend, and in LAN server mode for
// any client on the network. This list is deliberately fixed: a newly exported method, such as
// one that logs as a system actor, must be reviewed for what it lets a client write.
func TestAuditService_BoundMethods(t *testing.T) {
	want := []string{"CreateSession", "DestroySession", "GetAuditLogs", "GetSessionUser", "LogAction", "LogPatientAction"}

	typ := reflect.TypeOf(&services.AuditService{})
	var got []string
	for i := range typ.NumMethod() {
		got = append(got, typ.Method(i).Name)
	}
	if !slices.Equal(got, want) {
		t.Errorf("AuditService exported methods changed:\n got  %v\n want %v", got, want)
	}
}

// Whether an entry's actor is a system actor is decided by domain.IsSystemActorID and sent with
// each entry, so the audit view never applies its own, possibly different, rule.
func TestAuditService_MarksSystemActors(t *testing.T) {
	tempDir := t.TempDir()
	auditDb, err := sqlite.OpenAudit(filepath.Join(tempDir, "audit.db"))
	if err != nil {
		t.Fatalf("Failed to open audit sqlite db: %v", err)
	}
	defer auditDb.Close()
	mainDb, err := sqlite.Open(filepath.Join(tempDir, "main.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer mainDb.Close()

	configRepo := sqlite.NewPracticeConfigRepository(mainDb)
	if err := configRepo.SaveProvider(context.Background(), &domain.Provider{ID: "prov_1", Name: "Test Prov", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	auditRepo := sqlite.NewAuditRepository(auditDb)
	service := services.NewAuditService(auditRepo, configRepo)
	token, err := service.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	want := map[string]bool{
		domain.SystemActorReminders: true,
		"system:":                   false, // the bare prefix isn't a system actor
		"System:reminders":          false,
		"prov_1":                    false,
	}
	for userID := range want {
		if err := auditRepo.Log(context.Background(), &domain.AuditLogEntry{
			ID: "audit_" + userID, UserID: userID, UserName: "x", Action: domain.AuditActionCreate, Resource: "notification",
		}); err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	logs, err := service.GetAuditLogs(token, "", 10, 0)
	if err != nil {
		t.Fatalf("GetAuditLogs failed: %v", err)
	}
	if len(logs) != len(want) {
		t.Fatalf("Expected %d entries, got %d", len(want), len(logs))
	}
	for _, l := range logs {
		if l.SystemActor != want[l.UserID] {
			t.Errorf("%q: SystemActor = %v; want %v", l.UserID, l.SystemActor, want[l.UserID])
		}
	}
}
