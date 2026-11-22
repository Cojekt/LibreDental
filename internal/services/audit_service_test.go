package services_test

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage"
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

	if _, err := service.GetAuditLogs("bogus-token", "pat_123", 10, ""); err != services.ErrUnauthorized {
		t.Fatalf("Expected ErrUnauthorized for unauthenticated call, got %v", err)
	}

	// 2. Fetch Logs
	logs, err := service.GetAuditLogs(readToken, "pat_123", 10, "")
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
	allLogs, err := service.GetAuditLogs(readToken, "", 10, "")
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

	paginatedLogs, err := service.GetAuditLogs(readToken, "pat_123", 1, "") // Limit 1
	if err != nil {
		t.Fatalf("Failed to get paginated logs: %v", err)
	}
	if len(paginatedLogs) != 1 {
		t.Errorf("Expected 1 paginated log entry, got %d", len(paginatedLogs))
	}

	nextPage, err := service.GetAuditLogs(readToken, "pat_123", 1, paginatedLogs[0].ID)
	if err != nil {
		t.Fatalf("Failed to get next page of logs: %v", err)
	}
	if len(nextPage) != 1 {
		t.Fatalf("Expected 1 paginated log entry, got %d", len(nextPage))
	}

	if paginatedLogs[0].ID == nextPage[0].ID {
		t.Errorf("Next page returned the same row")
	}

	if _, err := service.GetAuditLogs(readToken, "pat_123", 1, "audit_missing"); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for an unknown cursor, got %v", err)
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

	logs, err := service.GetAuditLogs(token, "pat_9", 10, "")
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

	logs, err := service.GetAuditLogs(token, "", 10, "")
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

func TestAuditService_DeactivatingProviderRevokesSessions(t *testing.T) {
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	adb, err := sqlite.OpenAudit(filepath.Join(dir, "a.db"))
	if err != nil {
		t.Fatalf("open audit db: %v", err)
	}
	defer adb.Close()

	cfgRepo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()
	for _, id := range []string{"keep", "remove"} {
		if err := cfgRepo.SaveProvider(ctx, &domain.Provider{ID: id, Name: id, Pin: "4729", IsActive: true}); err != nil {
			t.Fatalf("save provider %s: %v", id, err)
		}
	}
	audit := services.NewAuditService(sqlite.NewAuditRepository(adb), cfgRepo)
	cfg := services.NewPracticeConfigService(cfgRepo, audit)

	adminTok, err := audit.CreateSession("keep", "4729")
	if err != nil {
		t.Fatalf("admin session: %v", err)
	}
	victimTok, err := audit.CreateSession("remove", "4729")
	if err != nil {
		t.Fatalf("victim session: %v", err)
	}
	if audit.GetSessionUser(victimTok) == nil {
		t.Fatal("precondition: victim session should be valid before deactivation")
	}

	if err := cfg.DeleteProvider(adminTok, "remove"); err != nil {
		t.Fatalf("deactivate provider: %v", err)
	}

	if audit.GetSessionUser(victimTok) != nil {
		t.Error("deactivated provider's session is still valid; it must be revoked")
	}
	if audit.GetSessionUser(adminTok) == nil {
		t.Error("unrelated provider's session was revoked")
	}
}

func TestAuditService_SaveProviderInactiveRevokesSessions(t *testing.T) {
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	adb, err := sqlite.OpenAudit(filepath.Join(dir, "a.db"))
	if err != nil {
		t.Fatalf("open audit db: %v", err)
	}
	defer adb.Close()

	cfgRepo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()
	for _, id := range []string{"keep", "edit"} {
		if err := cfgRepo.SaveProvider(ctx, &domain.Provider{ID: id, Name: id, Pin: "4729", IsActive: true}); err != nil {
			t.Fatalf("save provider %s: %v", id, err)
		}
	}
	audit := services.NewAuditService(sqlite.NewAuditRepository(adb), cfgRepo)
	cfg := services.NewPracticeConfigService(cfgRepo, audit)

	adminTok, err := audit.CreateSession("keep", "4729")
	if err != nil {
		t.Fatalf("admin session: %v", err)
	}
	victimTok, err := audit.CreateSession("edit", "4729")
	if err != nil {
		t.Fatalf("victim session: %v", err)
	}

	// Edit the provider to inactive (the separate SaveProvider deactivation path).
	if _, err := cfg.SaveProvider(adminTok, domain.Provider{ID: "edit", Name: "edit", Pin: "****", IsActive: false}); err != nil {
		t.Fatalf("deactivate via SaveProvider: %v", err)
	}
	if audit.GetSessionUser(victimTok) != nil {
		t.Error("session of a provider edited inactive is still valid; it must be revoked")
	}
}

func TestAuditService_SelfDeleteIsAudited(t *testing.T) {
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	adb, err := sqlite.OpenAudit(filepath.Join(dir, "a.db"))
	if err != nil {
		t.Fatalf("open audit db: %v", err)
	}
	defer adb.Close()

	cfgRepo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()
	for _, id := range []string{"keep", "self"} {
		if err := cfgRepo.SaveProvider(ctx, &domain.Provider{ID: id, Name: id, Pin: "4729", IsActive: true}); err != nil {
			t.Fatalf("save provider %s: %v", id, err)
		}
	}
	audit := services.NewAuditService(sqlite.NewAuditRepository(adb), cfgRepo)
	cfg := services.NewPracticeConfigService(cfgRepo, audit)

	selfTok, err := audit.CreateSession("self", "4729")
	if err != nil {
		t.Fatalf("self session: %v", err)
	}

	// A provider deactivates themselves: the deletion must still be audited
	// (revocation must not drop the token before the action is logged).
	if err := cfg.DeleteProvider(selfTok, "self"); err != nil {
		t.Fatalf("self delete: %v", err)
	}
	if audit.GetSessionUser(selfTok) != nil {
		t.Error("self-deleted provider's session should be revoked")
	}
	var count int
	if err := adb.QueryRow(
		"SELECT COUNT(*) FROM audit_logs WHERE user_id = 'self' AND resource = 'provider' AND action = 'DELETE'",
	).Scan(&count); err != nil {
		t.Fatalf("query audit logs: %v", err)
	}
	if count != 1 {
		t.Errorf("self-deletion audit entries = %d, want 1 (the successful deletion must be logged)", count)
	}
}
