package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage"
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
	code := domain.CountryCode(countryCode)
	cfg := domain.NewPracticeConfig(code)

	err := s.repo.Save(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to save practice config: %w", err)
	}
	return cfg, nil
}

// GetSupportedCountries returns the list of all supported country configurations.
func (s *PracticeConfigService) GetSupportedCountries() ([]domain.CountryConfig, error) {
	return domain.GetSupportedCountries(), nil
}

// GetCountryConfig returns country metadata for a specific country code.
func (s *PracticeConfigService) GetCountryConfig(countryCode string) (*domain.CountryConfig, error) {
	cfg, ok := domain.GetCountryConfig(domain.CountryCode(countryCode))
	if !ok {
		return nil, fmt.Errorf("unsupported country code: %s", countryCode)
	}
	return &cfg, nil
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
func (s *PracticeConfigService) ListProviders() ([]*domain.Provider, error) {
	return s.repo.ListProviders(context.Background())
}

// SaveProvider creates or updates a clinic provider/staff member.
func (s *PracticeConfigService) SaveProvider(p domain.Provider) (*domain.Provider, error) {
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
