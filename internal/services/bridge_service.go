package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LibreDental/libredental/internal/app"
	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// ErrBridgesUnavailable is returned in server builds: a bridge starts a program on the machine
// running the backend, which in server mode is not the workstation the user is sitting at.
var ErrBridgesUnavailable = errors.New("program bridges are only available in the desktop app")

// bridgeExportMaxAge is how long exported documents are left for a bridged program to read
// before a later launch removes them; they are copies of patient records in the OS temp dir.
const bridgeExportMaxAge = 24 * time.Hour

// BridgeService exposes local program bridges to the Wails frontend. It follows the same
// registry pattern as the claim and notification integrations, but bridge config is stored
// per-workstation in bridges.json (an executable path only makes sense on the machine it was
// set on) rather than in the shared database or the keychain, since none of it is secret.
type BridgeService struct {
	appDir       string
	patientRepo  storage.PatientRepository
	documents    *DocumentService
	auditService *AuditService
	serverMode   bool
	start        func(*domain.BridgeLaunch) error

	mu      sync.Mutex
	bridges map[string]domain.ProgramBridge
}

func NewBridgeService(appDir string, patientRepo storage.PatientRepository, documents *DocumentService, auditService *AuditService) *BridgeService {
	return &BridgeService{
		appDir:       appDir,
		patientRepo:  patientRepo,
		documents:    documents,
		auditService: auditService,
		serverMode:   app.IsServerBuild(),
		start:        startBridgeProcess,
		bridges:      make(map[string]domain.ProgramBridge),
	}
}

// RegisterProgramBridge registers a bridge for use.
// Exposed as a function rather than a method so Wails does not bind it.
func RegisterProgramBridge(s *BridgeService, b domain.ProgramBridge) {
	if b == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bridges[b.Name()] = b
}

// ListBridges returns the registered bridges with this workstation's config.
func (s *BridgeService) ListBridges(token string) ([]domain.BridgeInfo, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	if s.serverMode {
		return []domain.BridgeInfo{}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	stored, err := s.loadConfigsLocked()
	if err != nil {
		return nil, err
	}
	infos := make([]domain.BridgeInfo, 0, len(s.bridges))
	for name, b := range s.bridges {
		infos = append(infos, domain.BridgeInfo{
			Name:         name,
			Capabilities: b.Capabilities(),
			Config:       mergeBridgeConfig(b, stored[name]),
		})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
	return infos, nil
}

// SetBridgeConfig saves this workstation's config for a bridge. Unknown keys are dropped.
func (s *BridgeService) SetBridgeConfig(token string, name string, config map[string]string) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}
	if s.serverMode {
		return ErrBridgesUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.bridges[name]
	if !ok {
		return fmt.Errorf("program bridge %q not registered", name)
	}
	merged := mergeBridgeConfig(b, config)
	if merged[domain.BridgeConfigEnabled] == "true" {
		if err := validateBridgeExecutable(merged[domain.BridgeConfigPath]); err != nil {
			return err
		}
	}
	if _, err := splitBridgeArgs(merged[domain.BridgeConfigArgs]); err != nil {
		return err
	}

	stored, err := s.loadConfigsLocked()
	if err != nil {
		return err
	}
	stored[name] = merged
	if err := s.saveConfigsLocked(stored); err != nil {
		return err
	}
	if err := s.auditService.LogAction(token, domain.AuditActionUpdate, "program_bridge_config",
		"Updated configuration for program bridge "+name); err != nil {
		return fmt.Errorf("bridge config saved but failed to log audit: %w", err)
	}
	return nil
}

// LaunchBridge opens a patient, and optionally some of their documents, in a bridged program.
// Handing patient data to another program is a disclosure, so it is audited as an export
// before the program is started, and not started if the audit entry cannot be written.
func (s *BridgeService) LaunchBridge(token string, name string, patientID string, documentIDs []string) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}
	if s.serverMode {
		return ErrBridgesUnavailable
	}
	if patientID == "" {
		return fmt.Errorf("%w: patient ID is required", storage.ErrInvalidInput)
	}

	s.mu.Lock()
	b, ok := s.bridges[name]
	stored, err := s.loadConfigsLocked()
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("program bridge %q not registered", name)
	}
	if err != nil {
		return err
	}
	config := mergeBridgeConfig(b, stored[name])
	if config[domain.BridgeConfigEnabled] != "true" {
		return fmt.Errorf("%w: %s is not enabled on this workstation", ErrBridgeNotConfigured, name)
	}

	needed := domain.BridgeCapabilityPatient
	if len(documentIDs) > 0 {
		needed = domain.BridgeCapabilityDocuments
	}
	if !slices.Contains(b.Capabilities(), needed) {
		return fmt.Errorf("%w: program bridge %q does not support %s", storage.ErrInvalidInput, name, needed)
	}

	ctx := context.Background()
	patient, err := s.patientRepo.GetByID(ctx, patientID)
	if err != nil {
		return fmt.Errorf("failed to get patient for program bridge: %w", err)
	}

	removeStaleBridgeExports(bridgeExportMaxAge)

	req := &domain.BridgeRequest{Patient: patient}
	if len(documentIDs) > 0 {
		dir, files, err := s.documents.exportPatientDocuments(patientID, documentIDs)
		if err != nil {
			return err
		}
		req.Dir, req.Files = dir, files
	}
	cleanup := func() {
		if req.Dir != "" {
			_ = os.RemoveAll(req.Dir)
		}
	}

	launch, err := b.BuildLaunch(ctx, req, config)
	if err != nil {
		cleanup()
		return err
	}

	detail := fmt.Sprintf("Sent patient to program bridge %s", name)
	if len(documentIDs) > 0 {
		detail = fmt.Sprintf("Sent patient and %d document(s) to program bridge %s: %s",
			len(documentIDs), name, strings.Join(documentIDs, ", "))
	}
	if err := s.auditService.LogPatientAction(token, domain.AuditActionExport, patientID, "program_bridge", detail); err != nil {
		cleanup()
		return fmt.Errorf("failed to log audit, program bridge not started: %w", err)
	}

	if err := s.start(launch); err != nil {
		cleanup()
		_ = s.auditService.LogPatientAction(token, domain.AuditActionExport, patientID, "program_bridge",
			fmt.Sprintf("Program bridge %s failed to start: %v", name, err))
		return fmt.Errorf("failed to start program bridge %q: %w", name, err)
	}
	return nil
}

func mergeBridgeConfig(b domain.ProgramBridge, config map[string]string) map[string]string {
	merged := b.DefaultConfig()
	for k := range merged {
		if v, ok := config[k]; ok {
			merged[k] = v
		}
	}
	return merged
}

func (s *BridgeService) configPath() string {
	return filepath.Join(s.appDir, "bridges.json")
}

func (s *BridgeService) loadConfigsLocked() (map[string]map[string]string, error) {
	configs := make(map[string]map[string]string)
	data, err := os.ReadFile(s.configPath())
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(data) == 0) {
		return configs, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read bridge config: %w", err)
	}
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse bridge config: %w", err)
	}
	return configs, nil
}

func (s *BridgeService) saveConfigsLocked(configs map[string]map[string]string) error {
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode bridge config: %w", err)
	}
	tmp := s.configPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("failed to write bridge config: %w", err)
	}
	if err := os.Rename(tmp, s.configPath()); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to save bridge config: %w", err)
	}
	return nil
}

func startBridgeProcess(launch *domain.BridgeLaunch) error {
	// #nosec G204 - the executable is an absolute path set by staff, args are passed without a shell
	cmd := exec.Command(launch.Executable, launch.Args...)
	// Many imaging programs locate their own resources relative to the working directory.
	cmd.Dir = filepath.Dir(launch.Executable)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// removeStaleBridgeExports deletes exported document copies old enough that the program they
// were sent to has had ample time to read them.
func removeStaleBridgeExports(maxAge time.Duration) {
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), bridgeExportPrefix+"*"))
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, dir := range matches {
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(dir)
		}
	}
}
