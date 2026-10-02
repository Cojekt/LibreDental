package services

import (
	"context"
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
// registry pattern as the claim and notification integrations. Bridge config lives in the
// database rather than the keychain, since none of it is secret.
type BridgeService struct {
	repo         storage.ProgramBridgeRepository
	patientRepo  storage.PatientRepository
	documents    *DocumentService
	auditService *AuditService
	serverMode   bool
	start        func(*domain.BridgeLaunch) error

	mu      sync.RWMutex
	bridges map[string]domain.ProgramBridge
}

func NewBridgeService(repo storage.ProgramBridgeRepository, patientRepo storage.PatientRepository, documents *DocumentService, auditService *AuditService) *BridgeService {
	return &BridgeService{
		repo:         repo,
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

// ListBridges returns the registered bridges with their current config.
func (s *BridgeService) ListBridges(token string) ([]domain.BridgeInfo, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	if s.serverMode {
		return []domain.BridgeInfo{}, nil
	}

	stored, err := s.repo.List(context.Background())
	if err != nil {
		return nil, err
	}
	byName := make(map[string]domain.BridgeConfig, len(stored))
	for _, cfg := range stored {
		byName[cfg.Name] = *cfg
	}

	infos := make([]domain.BridgeInfo, 0)
	for _, b := range s.registered() {
		cfg, ok := byName[b.Name()]
		if !ok {
			cfg = defaultBridgeConfig(b)
		}
		infos = append(infos, domain.BridgeInfo{Capabilities: b.Capabilities(), Config: cfg})
	}
	return infos, nil
}

// SetBridgeConfig saves the config for the bridge named by config.Name. The change is audited
// before it is saved, so a config that can launch programs never exists without a record of
// who set it.
func (s *BridgeService) SetBridgeConfig(token string, config domain.BridgeConfig) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}
	if s.serverMode {
		return ErrBridgesUnavailable
	}
	if _, ok := s.bridge(config.Name); !ok {
		return fmt.Errorf("program bridge %q not registered", config.Name)
	}
	if config.Enabled {
		if err := validateBridgeExecutable(config.Path); err != nil {
			return err
		}
	}
	if err := validateBridgeArgs(config.Args); err != nil {
		return err
	}

	ctx := context.Background()
	if existing, err := s.repo.Get(ctx, config.Name); err == nil {
		config.CreatedAt = existing.CreatedAt
	} else if !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("failed to load program bridge config: %w", err)
	}
	detail := fmt.Sprintf("Updated configuration for program bridge %s (enabled: %t, path: %s, args: %s)",
		config.Name, config.Enabled, config.Path, config.Args)
	if err := s.auditService.LogAction(token, domain.AuditActionUpdate, "program_bridge_config", detail); err != nil {
		return fmt.Errorf("failed to log audit, bridge config not saved: %w", err)
	}
	if err := s.repo.Save(ctx, &config); err != nil {
		_ = s.auditService.LogAction(token, domain.AuditActionUpdate, "program_bridge_config",
			fmt.Sprintf("Failed to save configuration for program bridge %s: %v", config.Name, err))
		return err
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

	b, ok := s.bridge(name)
	if !ok {
		return fmt.Errorf("program bridge %q not registered", name)
	}
	ctx := context.Background()
	config, err := s.configFor(ctx, b)
	if err != nil {
		return err
	}
	if !config.Enabled {
		return fmt.Errorf("%w: %s is not enabled", ErrBridgeNotConfigured, name)
	}

	needed := domain.BridgeCapabilityPatient
	if len(documentIDs) > 0 {
		needed = domain.BridgeCapabilityDocuments
	}
	if !slices.Contains(b.Capabilities(), needed) {
		return fmt.Errorf("%w: program bridge %q does not support %s", storage.ErrInvalidInput, name, needed)
	}

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

func (s *BridgeService) bridge(name string) (domain.ProgramBridge, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.bridges[name]
	return b, ok
}

func (s *BridgeService) registered() []domain.ProgramBridge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bridges := make([]domain.ProgramBridge, 0, len(s.bridges))
	for _, b := range s.bridges {
		bridges = append(bridges, b)
	}
	sort.Slice(bridges, func(i, j int) bool { return bridges[i].Name() < bridges[j].Name() })
	return bridges
}

// configFor returns the stored config, or the bridge's defaults if it was never configured.
func (s *BridgeService) configFor(ctx context.Context, b domain.ProgramBridge) (domain.BridgeConfig, error) {
	cfg, err := s.repo.Get(ctx, b.Name())
	if errors.Is(err, storage.ErrNotFound) {
		return defaultBridgeConfig(b), nil
	}
	if err != nil {
		return domain.BridgeConfig{}, fmt.Errorf("failed to load program bridge config: %w", err)
	}
	return *cfg, nil
}

func defaultBridgeConfig(b domain.ProgramBridge) domain.BridgeConfig {
	cfg := b.DefaultConfig()
	cfg.Name = b.Name()
	return cfg
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
