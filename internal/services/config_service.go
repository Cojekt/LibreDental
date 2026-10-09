package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type PracticeConfigService struct {
	repo         storage.PracticeConfigRepository
	auditService *AuditService
}

var pinPattern = regexp.MustCompile(`^[0-9]{4}$`)

// NewPracticeConfigService constructs the service. auditService may be nil (e.g. the
// internal instance AuditService itself uses for PIN verification during login, before
// any session exists) — mutations then skip the session check and aren't audit-logged.
func NewPracticeConfigService(repo storage.PracticeConfigRepository, auditService *AuditService) *PracticeConfigService {
	return &PracticeConfigService{repo: repo, auditService: auditService}
}

func (s *PracticeConfigService) requireSession(token string) error {
	if s.auditService != nil && s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}
	return nil
}

func (s *PracticeConfigService) logAction(token string, action domain.AuditAction, resource string, details string) {
	if s.auditService == nil {
		return
	}
	// SetConfig's first-run call has no session to attribute, so ErrUnauthorized is expected there.
	if err := s.auditService.LogAction(token, action, resource, details); err != nil && !errors.Is(err, ErrUnauthorized) {
		fmt.Printf("Warning: failed to log audit action: %v\n", err)
	}
}

// GetConfig fetches the current practice configuration, or returns nil if unconfigured.
func (s *PracticeConfigService) GetConfig() (*domain.PracticeConfig, error) {
	cfg, err := s.repo.Get(context.Background())
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch practice config: %w", err)
	}
	return cfg, nil
}

// SetConfig initializes or updates the practice country and derives all regional defaults.
// It runs as step one of first-run onboarding, before any provider exists, so a session is
// only required once the clinic has an active provider who could have logged in.
func (s *PracticeConfigService) SetConfig(token string, countryCode string) (*domain.PracticeConfig, error) {
	save := s.repo.Save
	if s.requireSession(token) != nil {
		// The repo checks for an active provider in the same statement as the write.
		save = s.repo.SaveInitialConfig
	}

	meta, err := s.GetCountryConfig(countryCode)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch country config for %s: %w", countryCode, err)
	}

	cfg := domain.NewPracticeConfig(*meta)

	_, existErr := s.repo.Get(context.Background())
	if existErr != nil && !errors.Is(existErr, storage.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing practice config: %w", existErr)
	}
	action := domain.AuditActionUpdate
	if errors.Is(existErr, storage.ErrNotFound) {
		action = domain.AuditActionCreate
	}

	if err := save(context.Background(), cfg); err != nil {
		if errors.Is(err, storage.ErrAlreadyInitialized) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("failed to save practice config: %w", err)
	}
	s.logAction(token, action, "practice_config", "Set practice config during onboarding")
	return cfg, nil
}

// GetSupportedCountries returns the list of all supported country configurations from the database.
func (s *PracticeConfigService) GetSupportedCountries() ([]domain.CountryConfig, error) {
	return s.repo.ListCountryConfigs(context.Background())
}

// GetCountryConfig returns country metadata for a specific country code from the database, or default fallback.
func (s *PracticeConfigService) GetCountryConfig(countryCode string) (*domain.CountryConfig, error) {
	ctx := context.Background()
	if countryCode != "" {
		cfg, err := s.repo.GetCountryConfig(ctx, countryCode)
		if err == nil {
			return cfg, nil
		}
	}
	// Fallback to default country config stored in database
	defCfg, err := s.repo.GetDefaultCountryConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch default country config from DB: %w", err)
	}
	return defCfg, nil
}

// UpdatePracticeConfig updates practice details and regional configuration.
func (s *PracticeConfigService) UpdatePracticeConfig(token string, cfg domain.PracticeConfig) (*domain.PracticeConfig, error) {
	if err := s.requireSession(token); err != nil {
		return nil, err
	}
	cfg.NPI = strings.TrimSpace(cfg.NPI)
	if cfg.NPI != "" && !validNPI(cfg.NPI) {
		return nil, fmt.Errorf("%w: practice NPI is not a valid 10-digit NPI", storage.ErrInvalidInput)
	}
	cfg.Timezone = strings.TrimSpace(cfg.Timezone)
	if err := validatePracticeTimezone(cfg.Timezone); err != nil {
		return nil, err
	}
	err := s.repo.Save(context.Background(), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to update practice config: %w", err)
	}
	s.logAction(token, domain.AuditActionUpdate, "practice_config", "Updated practice config")
	return &cfg, nil
}

// ListProviders fetches all configured clinic providers and staff.

// VerifyProviderPin checks a provider's pin and returns the redacted provider.
func (s *PracticeConfigService) VerifyProviderPin(id string, pin string) (*domain.Provider, error) {
	providers, err := s.repo.ListProviders(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	for _, p := range providers {
		if p.ID == id {
			if !p.IsActive {
				return nil, errors.New("provider is inactive")
			}
			if p.Pin == "" || pin == "" {
				return nil, errors.New("pin not set or empty")
			}
			if p.Pin != pin {
				return nil, errors.New("incorrect pin")
			}
			p.Pin = "****"
			return p, nil
		}
	}
	return nil, errors.New("provider not found")
}

// GetProviderPin returns a provider's PIN so logged-in staff can look up a forgotten
// one. PINs are attribution for the audit log rather than a security boundary (see
// domain.Provider.Pin), but every reveal is still audit-logged.
func (s *PracticeConfigService) GetProviderPin(token string, id string) (string, error) {
	if err := s.requireSession(token); err != nil {
		return "", err
	}
	providers, err := s.repo.ListProviders(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to list providers: %w", err)
	}
	for _, p := range providers {
		if p.ID == id {
			s.logAction(token, domain.AuditActionRead, "provider", fmt.Sprintf("Revealed PIN for provider %s", id))
			return p.Pin, nil
		}
	}
	return "", storage.ErrNotFound
}

func (s *PracticeConfigService) ListProviders() ([]*domain.Provider, error) {
	providers, err := s.repo.ListProviders(context.Background())
	if err != nil {
		return nil, err
	}
	for _, p := range providers {
		p.Pin = "****"
	}
	return providers, nil
}

// NeedsInitialProvider reports whether first-run onboarding still has to create the
// clinic's first provider.
func (s *PracticeConfigService) NeedsInitialProvider() (bool, error) {
	hasProvider, err := s.repo.HasActiveProvider(context.Background())
	if err != nil {
		return false, err
	}
	return !hasProvider, nil
}

// CreateInitialProvider is the only way to create a provider without a session: it
// succeeds only while the clinic has no active provider, then logs in as the new
// provider and returns the session token so the creation is attributed in the audit log.
func (s *PracticeConfigService) CreateInitialProvider(p domain.Provider) (string, error) {
	if s.auditService == nil {
		return "", errors.New("audit service not configured")
	}
	if p.Name == "" {
		return "", fmt.Errorf("%w: provider name is required", storage.ErrInvalidInput)
	}
	if !pinPattern.MatchString(p.Pin) {
		return "", fmt.Errorf("%w: pin must be exactly 4 digits", storage.ErrInvalidInput)
	}
	if err := rejectReservedProviderID(p.ID); err != nil {
		return "", err
	}
	if p.Role == "" {
		p.Role = domain.RoleDentist
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("prov_%d", time.Now().UnixNano())
	}
	if err := normalizeProviderIdentifiers(&p); err != nil {
		return "", err
	}

	if err := s.repo.CreateInitialProvider(context.Background(), &p); err != nil {
		return "", err
	}

	token, err := s.auditService.CreateSession(p.ID, p.Pin)
	if err != nil {
		return "", fmt.Errorf("failed to open session for initial provider: %w", err)
	}
	s.logAction(token, domain.AuditActionCreate, "provider", fmt.Sprintf("Created initial provider %s during onboarding", p.ID))
	return token, nil
}

// SaveProvider creates or updates a clinic provider/staff member.
func (s *PracticeConfigService) SaveProvider(token string, p domain.Provider) (*domain.Provider, error) {
	if err := s.requireSession(token); err != nil {
		return nil, err
	}
	if err := rejectReservedProviderID(p.ID); err != nil {
		return nil, err
	}
	isNew := p.ID == ""
	if p.ID == "" {
		p.ID = fmt.Sprintf("prov_%d", time.Now().UnixNano())
	} else {
		if p.Pin == "****" {
			existingProviders, err := s.repo.ListProviders(context.Background())
			if err != nil {
				return nil, fmt.Errorf("failed to list providers for pin preservation: %w", err)
			}

			found := false
			for _, ex := range existingProviders {
				if ex.ID == p.ID {
					p.Pin = ex.Pin
					found = true
					break
				}
			}

			if !found {
				return nil, fmt.Errorf("provider %q not found for pin preservation", p.ID)
			}
		}
	}

	if err := normalizeProviderIdentifiers(&p); err != nil {
		return nil, err
	}
	err := s.repo.SaveProvider(context.Background(), &p)
	if err != nil {
		return nil, fmt.Errorf("failed to save provider: %w", err)
	}

	action := domain.AuditActionUpdate
	if isNew {
		action = domain.AuditActionCreate
	}
	s.logAction(token, action, "provider", fmt.Sprintf("Saved provider %s", p.ID))

	p.Pin = "****"
	return &p, nil
}

// validatePracticeTimezone accepts an empty timezone (not set yet) or an IANA name. "Local" is
// rejected: it would mean the timezone of whichever machine runs the backend, which in LAN
// mode isn't necessarily where appointments are entered.
func validatePracticeTimezone(name string) error {
	if name == "" {
		return nil
	}
	if name == "Local" {
		return fmt.Errorf("%w: choose a named timezone, not the computer's local one", storage.ErrInvalidInput)
	}
	if _, err := time.LoadLocation(name); err != nil {
		return fmt.Errorf("%w: %q is not a known timezone", storage.ErrInvalidInput, name)
	}
	return nil
}

// rejectReservedProviderID stops a staff account from taking a system actor ID, which would
// make its actions look automatic in the audit trail. Case and surrounding spaces are ignored
// so look-alike IDs are refused too.
func rejectReservedProviderID(id string) error {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(id)), domain.SystemActorPrefix) {
		return fmt.Errorf("%w: provider ID %q is reserved", storage.ErrInvalidInput, id)
	}
	return nil
}

// normalizeProviderIdentifiers trims the claim identifiers on a provider and rejects malformed
// ones at entry, rather than letting them surface later as a failed claim submission.
func normalizeProviderIdentifiers(p *domain.Provider) error {
	p.NPI = strings.TrimSpace(p.NPI)
	p.TaxonomyCode = strings.ToUpper(strings.TrimSpace(p.TaxonomyCode))
	if p.NPI != "" && !validNPI(p.NPI) {
		return fmt.Errorf("%w: provider NPI is not a valid 10-digit NPI", storage.ErrInvalidInput)
	}
	if p.TaxonomyCode != "" && !taxonomyPattern.MatchString(p.TaxonomyCode) {
		return fmt.Errorf("%w: provider taxonomy code must be 10 characters ending in X", storage.ErrInvalidInput)
	}
	return nil
}

// DeleteProvider deactivates a provider record by ID. At least one provider must
// remain active at all times, so deactivating the last active provider is forbidden.
// The check-then-deactivate logic lives in the repository as a single atomic
// operation, so concurrent deletes of different providers can't race past it.
func (s *PracticeConfigService) DeleteProvider(token string, id string) error {
	if err := s.requireSession(token); err != nil {
		return err
	}
	if err := s.repo.DeleteProvider(context.Background(), id); err != nil {
		return err
	}
	s.logAction(token, domain.AuditActionDelete, "provider", fmt.Sprintf("Deleted provider %s", id))
	return nil
}

// ListOperatories fetches all configured operatories/treatment rooms.
func (s *PracticeConfigService) ListOperatories() ([]*domain.Operatory, error) {
	return s.repo.ListOperatories(context.Background())
}

// SaveOperatory creates or updates a clinic operatory/room.
func (s *PracticeConfigService) SaveOperatory(token string, op domain.Operatory) (*domain.Operatory, error) {
	if err := s.requireSession(token); err != nil {
		return nil, err
	}
	isNew := op.ID == ""
	if op.ID == "" {
		op.ID = fmt.Sprintf("op_%d", time.Now().UnixNano())
	}
	err := s.repo.SaveOperatory(context.Background(), &op)
	if err != nil {
		return nil, fmt.Errorf("failed to save operatory: %w", err)
	}
	action := domain.AuditActionUpdate
	if isNew {
		action = domain.AuditActionCreate
	}
	s.logAction(token, action, "operatory", fmt.Sprintf("Saved operatory %s", op.ID))
	return &op, nil
}

// DeleteOperatory removes an operatory record by ID.
func (s *PracticeConfigService) DeleteOperatory(token string, id string) error {
	if err := s.requireSession(token); err != nil {
		return err
	}
	if err := s.repo.DeleteOperatory(context.Background(), id); err != nil {
		return err
	}
	s.logAction(token, domain.AuditActionDelete, "operatory", fmt.Sprintf("Deleted operatory %s", id))
	return nil
}
