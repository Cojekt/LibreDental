package services

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

type bridgeTestEnv struct {
	svc       *BridgeService
	docs      *DocumentService
	audit     *AuditService
	token     string
	launches  []*domain.BridgeLaunch
	startErr  error
	exePath   string
	patientID string
}

func newTestBridgeService(t *testing.T) *bridgeTestEnv {
	t.Helper()
	tmpDir := t.TempDir()
	// Keep exported document copies inside the test's own directory.
	t.Setenv("TMPDIR", filepath.Join(tmpDir, "tmp"))
	if err := os.MkdirAll(filepath.Join(tmpDir, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := sqlite.Open(filepath.Join(tmpDir, "test_bridge.db"))
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	auditDb, err := sqlite.OpenAudit(filepath.Join(tmpDir, "test_bridge_audit.db"))
	if err != nil {
		t.Fatalf("Failed to open audit db: %v", err)
	}
	t.Cleanup(func() { auditDb.Close() })

	ctx := context.Background()
	configRepo := sqlite.NewPracticeConfigRepository(db)
	if err := configRepo.SaveProvider(ctx, &domain.Provider{ID: "prov_1", Name: "Test Prov", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	auditSvc := NewAuditService(sqlite.NewAuditRepository(auditDb), configRepo)
	token, err := auditSvc.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	patientRepo := sqlite.NewPatientRepository(db)
	for _, id := range []string{"pat_1", "pat_2"} {
		if err := patientRepo.Create(ctx, &domain.Patient{ID: id, FirstName: "Test", LastName: id}); err != nil {
			t.Fatalf("Failed to create patient: %v", err)
		}
	}

	docs := NewDocumentService(sqlite.NewDocumentRepository(db), tmpDir, auditSvc)
	env := &bridgeTestEnv{
		svc:       NewBridgeService(tmpDir, patientRepo, docs, auditSvc),
		docs:      docs,
		audit:     auditSvc,
		token:     token,
		exePath:   filepath.Join(tmpDir, "bin", "viewer"),
		patientID: "pat_1",
	}
	env.svc.serverMode = false
	env.svc.start = func(l *domain.BridgeLaunch) error {
		env.launches = append(env.launches, l)
		return env.startErr
	}
	for _, b := range DefaultProgramBridges() {
		RegisterProgramBridge(env.svc, b)
	}
	return env
}

func (e *bridgeTestEnv) saveDoc(t *testing.T, patientID, name string) *domain.Document {
	t.Helper()
	doc, err := e.docs.SaveDocumentBase64(e.token, patientID, name, "", string(domain.DocumentTypeXRay), "application/dicom",
		base64.StdEncoding.EncodeToString([]byte("DICM data for "+name)))
	if err != nil {
		t.Fatalf("Failed to save document: %v", err)
	}
	return doc
}

func (e *bridgeTestEnv) enable(t *testing.T, name, args string) {
	t.Helper()
	cfg := map[string]string{domain.BridgeConfigEnabled: "true", domain.BridgeConfigPath: e.exePath}
	if args != "" {
		cfg[domain.BridgeConfigArgs] = args
	}
	if err := e.svc.SetBridgeConfig(e.token, name, cfg); err != nil {
		t.Fatalf("SetBridgeConfig: %v", err)
	}
}

func (e *bridgeTestEnv) bridgeAuditDetails(t *testing.T) []string {
	t.Helper()
	logs, err := e.audit.GetAuditLogs(e.token, e.patientID, 100, 0)
	if err != nil {
		t.Fatalf("GetAuditLogs: %v", err)
	}
	var details []string
	for _, l := range logs {
		if l.Resource == "program_bridge" {
			if l.Action != domain.AuditActionExport {
				t.Errorf("bridge audit action = %s, want EXPORT", l.Action)
			}
			details = append(details, l.Details)
		}
	}
	return details
}

func TestBridgeServiceRequiresSession(t *testing.T) {
	env := newTestBridgeService(t)
	if _, err := env.svc.ListBridges("bad"); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("ListBridges: expected ErrUnauthorized, got %v", err)
	}
	if err := env.svc.SetBridgeConfig("bad", "weasis", nil); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("SetBridgeConfig: expected ErrUnauthorized, got %v", err)
	}
	if err := env.svc.LaunchBridge("bad", "weasis", "pat_1", nil); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("LaunchBridge: expected ErrUnauthorized, got %v", err)
	}
}

func TestBridgeServiceServerMode(t *testing.T) {
	env := newTestBridgeService(t)
	env.svc.serverMode = true

	infos, err := env.svc.ListBridges(env.token)
	if err != nil || len(infos) != 0 {
		t.Errorf("ListBridges in server mode = %v, %v; want empty", infos, err)
	}
	if err := env.svc.SetBridgeConfig(env.token, "weasis", nil); !errors.Is(err, ErrBridgesUnavailable) {
		t.Errorf("SetBridgeConfig: expected ErrBridgesUnavailable, got %v", err)
	}
	if err := env.svc.LaunchBridge(env.token, "weasis", "pat_1", nil); !errors.Is(err, ErrBridgesUnavailable) {
		t.Errorf("LaunchBridge: expected ErrBridgesUnavailable, got %v", err)
	}
}

func TestBridgeServiceConfig(t *testing.T) {
	env := newTestBridgeService(t)

	infos, err := env.svc.ListBridges(env.token)
	if err != nil {
		t.Fatalf("ListBridges: %v", err)
	}
	if len(infos) != len(DefaultProgramBridges()) {
		t.Fatalf("ListBridges returned %d bridges, want %d", len(infos), len(DefaultProgramBridges()))
	}
	for _, info := range infos {
		if info.Config[domain.BridgeConfigEnabled] != "false" {
			t.Errorf("bridge %s enabled before configuration", info.Name)
		}
	}

	if err := env.svc.SetBridgeConfig(env.token, "weasis", map[string]string{
		domain.BridgeConfigEnabled: "true", domain.BridgeConfigPath: "relative/weasis",
	}); !errors.Is(err, ErrBridgeNotConfigured) {
		t.Errorf("expected ErrBridgeNotConfigured for relative path, got %v", err)
	}
	if err := env.svc.SetBridgeConfig(env.token, "weasis", map[string]string{
		domain.BridgeConfigArgs: `-d "unterminated`,
	}); !errors.Is(err, errBridgeTemplate) {
		t.Errorf("expected template error, got %v", err)
	}
	if err := env.svc.SetBridgeConfig(env.token, "nope", nil); err == nil {
		t.Error("expected error for unregistered bridge")
	}

	if err := env.svc.SetBridgeConfig(env.token, "weasis", map[string]string{
		domain.BridgeConfigEnabled: "true", domain.BridgeConfigPath: env.exePath, "unknown_key": "x",
	}); err != nil {
		t.Fatalf("SetBridgeConfig: %v", err)
	}

	// A fresh service on the same app dir sees the saved config, as after a restart. Saved
	// args win over a later change to the bridge's default.
	reloaded := NewBridgeService(env.svc.appDir, nil, nil, env.audit)
	RegisterProgramBridge(reloaded, NewCommandLineBridge("weasis", "default", domain.BridgeCapabilityDocuments))
	infos, err = reloaded.ListBridges(env.token)
	if err != nil {
		t.Fatalf("ListBridges after reload: %v", err)
	}
	cfg := infos[0].Config
	if cfg[domain.BridgeConfigEnabled] != "true" || cfg[domain.BridgeConfigPath] != env.exePath || cfg[domain.BridgeConfigArgs] != `'$dicom:get -l "{dir}"'` {
		t.Errorf("unexpected reloaded config: %v", cfg)
	}
	if _, ok := cfg["unknown_key"]; ok {
		t.Error("unknown config key was persisted")
	}

	logs, err := env.audit.GetAuditLogs(env.token, "", 100, 0)
	if err != nil {
		t.Fatalf("GetAuditLogs: %v", err)
	}
	found := false
	for _, l := range logs {
		found = found || (l.Resource == "program_bridge_config" && l.Action == domain.AuditActionUpdate)
	}
	if !found {
		t.Error("config change was not audited")
	}
}

func TestBridgeServiceLaunchDocuments(t *testing.T) {
	env := newTestBridgeService(t)
	doc1 := env.saveDoc(t, "pat_1", `pano "front".dcm`)
	doc2 := env.saveDoc(t, "pat_1", "bitewing")
	env.enable(t, "weasis", "")

	if err := env.svc.LaunchBridge(env.token, "weasis", "pat_1", []string{doc1.ID, doc2.ID}); err != nil {
		t.Fatalf("LaunchBridge: %v", err)
	}
	if len(env.launches) != 1 {
		t.Fatalf("expected 1 launch, got %d", len(env.launches))
	}
	launch := env.launches[0]
	if launch.Executable != env.exePath || len(launch.Args) != 1 || !strings.HasPrefix(launch.Args[0], `$dicom:get -l "`) {
		t.Fatalf("unexpected launch: %+v", launch)
	}

	dir := strings.TrimSuffix(strings.TrimPrefix(launch.Args[0], `$dicom:get -l "`), `"`)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("export dir not readable: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if strings.Join(names, ",") != "01_pano__front_.dcm,02_bitewing.dcm" {
		t.Errorf("exported files = %v", names)
	}
	data, _ := os.ReadFile(filepath.Join(dir, names[0]))
	if !strings.Contains(string(data), "pano") {
		t.Errorf("exported content mismatch: %q", data)
	}

	details := env.bridgeAuditDetails(t)
	if len(details) != 1 || !strings.Contains(details[0], doc1.ID) || !strings.Contains(details[0], "weasis") {
		t.Errorf("unexpected bridge audit entries: %v", details)
	}
}

func TestBridgeServiceLaunchPatient(t *testing.T) {
	env := newTestBridgeService(t)
	env.enable(t, "custom", "--patient {patient_id} --last {last_name}")

	if err := env.svc.LaunchBridge(env.token, "custom", "pat_1", nil); err != nil {
		t.Fatalf("LaunchBridge: %v", err)
	}
	if got := strings.Join(env.launches[0].Args, "|"); got != "--patient|pat_1|--last|pat_1" {
		t.Errorf("args = %s", got)
	}
	if len(env.bridgeAuditDetails(t)) != 1 {
		t.Error("patient launch was not audited")
	}
}

func TestBridgeServiceLaunchRejections(t *testing.T) {
	env := newTestBridgeService(t)
	otherDoc := env.saveDoc(t, "pat_2", "other.dcm")

	if err := env.svc.LaunchBridge(env.token, "weasis", "pat_1", nil); !errors.Is(err, ErrBridgeNotConfigured) {
		t.Errorf("disabled bridge: expected ErrBridgeNotConfigured, got %v", err)
	}

	env.enable(t, "weasis", "")
	if err := env.svc.LaunchBridge(env.token, "weasis", "pat_1", nil); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("patient-only launch on documents bridge: expected ErrInvalidInput, got %v", err)
	}
	if err := env.svc.LaunchBridge(env.token, "weasis", "pat_1", []string{otherDoc.ID}); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("other patient's document: expected ErrInvalidInput, got %v", err)
	}
	if err := env.svc.LaunchBridge(env.token, "weasis", "", []string{otherDoc.ID}); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("missing patient: expected ErrInvalidInput, got %v", err)
	}
	if len(env.launches) != 0 {
		t.Errorf("rejected launches started a program: %+v", env.launches)
	}
	if details := env.bridgeAuditDetails(t); len(details) != 0 {
		t.Errorf("rejected launches were audited as exports: %v", details)
	}
}

func TestBridgeServiceLaunchStartFailure(t *testing.T) {
	env := newTestBridgeService(t)
	doc := env.saveDoc(t, "pat_1", "pano.dcm")
	env.enable(t, "radiant", "")
	env.startErr = errors.New("exec format error")

	if err := env.svc.LaunchBridge(env.token, "radiant", "pat_1", []string{doc.ID}); err == nil {
		t.Fatal("expected start failure")
	}
	dir := env.launches[0].Args[1]
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("export dir not cleaned up after failed start: %v", err)
	}
	details := env.bridgeAuditDetails(t)
	if len(details) != 2 || !strings.Contains(strings.Join(details, "\n"), "failed to start") {
		t.Errorf("start failure not audited: %v", details)
	}
}

func TestRemoveStaleBridgeExports(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	stale := filepath.Join(tmp, bridgeExportPrefix+"stale")
	fresh := filepath.Join(tmp, bridgeExportPrefix+"fresh")
	unrelated := filepath.Join(tmp, "other_stale")
	for _, d := range []string{stale, fresh, unrelated} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-48 * time.Hour)
	for _, d := range []string{stale, unrelated} {
		if err := os.Chtimes(d, old, old); err != nil {
			t.Fatal(err)
		}
	}

	removeStaleBridgeExports(24 * time.Hour)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale export dir was not removed")
	}
	for _, d := range []string{fresh, unrelated} {
		if _, err := os.Stat(d); err != nil {
			t.Errorf("%s should remain: %v", d, err)
		}
	}
}

func TestStartBridgeProcess(t *testing.T) {
	exe, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no 'true' executable on this platform")
	}
	if err := startBridgeProcess(&domain.BridgeLaunch{Executable: exe, Args: []string{"a b", `"c"`}}); err != nil {
		t.Fatalf("startBridgeProcess: %v", err)
	}
	if err := startBridgeProcess(&domain.BridgeLaunch{Executable: filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Error("expected error starting a missing executable")
	}
}
