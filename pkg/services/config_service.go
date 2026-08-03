package services

import (
	"context"
	"errors"
	"fmt"

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
