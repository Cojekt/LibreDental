package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage"
)

type ConfigService struct {
	repo storage.ConfigRepository
}

func NewConfigService(repo storage.ConfigRepository) *ConfigService {
	return &ConfigService{repo: repo}
}

// GetConfig fetches the current practice configuration, or returns nil if unconfigured.
func (s *ConfigService) GetConfig() (*domain.PracticeConfig, error) {
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
func (s *ConfigService) SetConfig(countryCode string) (*domain.PracticeConfig, error) {
	code := domain.CountryCode(countryCode)
	cfg := domain.NewPracticeConfig(code)

	err := s.repo.Save(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to save practice config: %w", err)
	}
	return cfg, nil
}

// GetSupportedCountries returns the list of all supported country configurations.
func (s *ConfigService) GetSupportedCountries() ([]domain.CountryConfig, error) {
	return domain.GetSupportedCountries(), nil
}

// GetCountryConfig returns country metadata for a specific country code.
func (s *ConfigService) GetCountryConfig(countryCode string) (*domain.CountryConfig, error) {
	cfg, ok := domain.GetCountryConfig(domain.CountryCode(countryCode))
	if !ok {
		return nil, fmt.Errorf("unsupported country code: %s", countryCode)
	}
	return &cfg, nil
}

// UpdatePracticeConfig updates practice details and regional configuration.
func (s *ConfigService) UpdatePracticeConfig(cfg domain.PracticeConfig) (*domain.PracticeConfig, error) {
	err := s.repo.Save(context.Background(), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to update practice config: %w", err)
	}
	return &cfg, nil
}

// ListProviders fetches all configured clinic providers and staff.
func (s *ConfigService) ListProviders() ([]*domain.Provider, error) {
	return s.repo.ListProviders(context.Background())
}

// SaveProvider creates or updates a clinic provider/staff member.
func (s *ConfigService) SaveProvider(p domain.Provider) (*domain.Provider, error) {
	if p.ID == "" {
		p.ID = fmt.Sprintf("prov_%d", time.Now().UnixNano())
	}
	err := s.repo.SaveProvider(context.Background(), &p)
	if err != nil {
		return nil, fmt.Errorf("failed to save provider: %w", err)
	}
	return &p, nil
}

// DeleteProvider removes a provider record by ID.
func (s *ConfigService) DeleteProvider(id string) error {
	return s.repo.DeleteProvider(context.Background(), id)
}

// ListOperatories fetches all configured operatories/treatment rooms.
func (s *ConfigService) ListOperatories() ([]*domain.Operatory, error) {
	return s.repo.ListOperatories(context.Background())
}

// SaveOperatory creates or updates a clinic operatory/room.
func (s *ConfigService) SaveOperatory(op domain.Operatory) (*domain.Operatory, error) {
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
func (s *ConfigService) DeleteOperatory(id string) error {
	return s.repo.DeleteOperatory(context.Background(), id)
}
