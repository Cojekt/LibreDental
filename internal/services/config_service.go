package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type PracticeConfigService struct {
	repo storage.PracticeConfigRepository
}

func NewPracticeConfigService(repo storage.PracticeConfigRepository) *PracticeConfigService {
	return &PracticeConfigService{repo: repo}
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
func (s *PracticeConfigService) SetConfig(countryCode string) (*domain.PracticeConfig, error) {
	meta, err := s.GetCountryConfig(countryCode)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch country config for %s: %w", countryCode, err)
	}

	cfg := domain.NewPracticeConfig(*meta)

	if err := s.repo.Save(context.Background(), cfg); err != nil {
		return nil, fmt.Errorf("failed to save practice config: %w", err)
	}
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
func (s *PracticeConfigService) UpdatePracticeConfig(cfg domain.PracticeConfig) (*domain.PracticeConfig, error) {
	err := s.repo.Save(context.Background(), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to update practice config: %w", err)
	}
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
			if p.Pin != pin {
				return nil, errors.New("incorrect pin")
			}
			p.Pin = "****"
			return p, nil
		}
	}
	return nil, errors.New("provider not found")
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

// SaveProvider creates or updates a clinic provider/staff member.
func (s *PracticeConfigService) SaveProvider(p domain.Provider) (*domain.Provider, error) {
	if p.ID == "" {
		p.ID = fmt.Sprintf("prov_%d", time.Now().UnixNano())
	} else {
		if p.Pin == "****" {
			existingProviders, err := s.repo.ListProviders(context.Background())
			if err == nil {
				for _, ex := range existingProviders {
					if ex.ID == p.ID {
						p.Pin = ex.Pin
						break
					}
				}
			}
		}
	}

	err := s.repo.SaveProvider(context.Background(), &p)
	if err != nil {
		return nil, fmt.Errorf("failed to save provider: %w", err)
	}

	p.Pin = "****"
	return &p, nil
}

// DeleteProvider removes a provider record by ID.
func (s *PracticeConfigService) DeleteProvider(id string) error {
	return s.repo.DeleteProvider(context.Background(), id)
}

// ListOperatories fetches all configured operatories/treatment rooms.
func (s *PracticeConfigService) ListOperatories() ([]*domain.Operatory, error) {
	return s.repo.ListOperatories(context.Background())
}

// SaveOperatory creates or updates a clinic operatory/room.
func (s *PracticeConfigService) SaveOperatory(op domain.Operatory) (*domain.Operatory, error) {
	if op.ID == "" {
		op.ID = fmt.Sprintf("op_%d", time.Now().UnixNano())
	}
	err := s.repo.SaveOperatory(context.Background(), &op)
	if err != nil {
		return nil, fmt.Errorf("failed to save operatory: %w", err)
	}
	return &op, nil
}

// DeleteOperatory removes an operatory record by ID.
func (s *PracticeConfigService) DeleteOperatory(id string) error {
	return s.repo.DeleteOperatory(context.Background(), id)
}
