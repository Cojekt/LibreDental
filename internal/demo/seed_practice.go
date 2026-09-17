package demo

import (
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

// bootstrapFirstProvider creates the very first provider through PracticeConfigService
// without a session token (the same way a fresh install's onboarding would, since no
// session can exist before any provider does), then opens a session as that provider.
// The returned token is used to authenticate every subsequent seed step.
func bootstrapFirstProvider(g *ServiceGraph, now time.Time, summary *SeedSummary) (string, error) {
	first := domain.Provider{
		ID:            "prov_101",
		Name:          "Dr. Sarah Jenkins",
		Role:          domain.RoleDentist,
		Specialty:     "General Dentistry & Restorative",
		LicenseNumber: "DEN-98212",
		Email:         "s.jenkins@apexdentalstudio.com",
		Phone:         "555-0101",
		Color:         "#3b82f6",
		Pin:           "1111",
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if _, err := g.Practice.SaveProvider("", first); err != nil {
		return "", fmt.Errorf("failed to bootstrap first provider: %w", err)
	}
	summary.ProvidersCount++

	token, err := g.Audit.CreateSession(first.ID, first.Pin)
	if err != nil {
		return "", fmt.Errorf("failed to create bootstrap session: %w", err)
	}
	return token, nil
}

func seedPracticeConfig(g *ServiceGraph, now time.Time, summary *SeedSummary) error {
	cfg, err := g.Practice.SetConfig("", string(domain.CountryUS))
	if err != nil {
		return fmt.Errorf("failed to initialize practice config: %w", err)
	}

	cfg.ClinicName = "Apex Dental Studio"
	cfg.Tagline = "Modern Dental Care & Implant Center"
	cfg.TaxID = "94-1234567"
	cfg.LicenseNumber = "DEN-CA-884920"
	cfg.Phone = "(555) 234-5678"
	cfg.Email = "info@apexdentalstudio.com"
	cfg.Website = "https://apexdentalstudio.example.com"
	cfg.AddressLine1 = "101 Dental Plaza, Suite 200"
	cfg.City = "San Francisco"
	cfg.StateProvince = "CA"
	cfg.PostalCode = "94105"
	cfg.DateFormat = "MM/DD/YYYY"
	cfg.BusinessHours = domain.DefaultBusinessHours()
	cfg.CreatedAt = now
	cfg.UpdatedAt = now

	if _, err := g.Practice.UpdatePracticeConfig("", *cfg); err != nil {
		return fmt.Errorf("failed to seed practice config: %w", err)
	}
	summary.PracticeConfigured = true
	return nil
}

func seedProviders(g *ServiceGraph, token string, now time.Time, summary *SeedSummary) error {
	providers := []domain.Provider{
		{
			ID:            "prov_102",
			Name:          "Dr. Marcus Vance",
			Role:          domain.RoleDentist,
			Specialty:     "Endodontics & Surgery",
			LicenseNumber: "DEN-77419",
			Email:         "m.vance@apexdentalstudio.com",
			Phone:         "555-0102",
			Color:         "#10b981",
			Pin:           "2222",
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "prov_103",
			Name:          "Elena Rostova",
			Role:          domain.RoleHygienist,
			Specialty:     "Preventive Dental Hygiene",
			LicenseNumber: "HYG-33104",
			Email:         "e.rostova@apexdentalstudio.com",
			Phone:         "555-0103",
			Color:         "#f59e0b",
			Pin:           "3333",
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	for _, p := range providers {
		if _, err := g.Practice.SaveProvider(token, p); err != nil {
			return fmt.Errorf("failed to seed provider %s: %w", p.Name, err)
		}
		summary.ProvidersCount++
	}
	return nil
}

func seedOperatories(g *ServiceGraph, token string, now time.Time, summary *SeedSummary) error {
	operatories := []domain.Operatory{
		{
			ID:        "op_1",
			Name:      "Operatory 1",
			RoomCode:  "Room 101",
			Type:      domain.OperatoryTypeGeneral,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "op_2",
			Name:      "Hygiene Suite",
			RoomCode:  "Room 102",
			Type:      domain.OperatoryTypeHygiene,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "op_3",
			Name:      "Surgical Suite",
			RoomCode:  "Room 103",
			Type:      domain.OperatoryTypeSurgery,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, op := range operatories {
		if _, err := g.Practice.SaveOperatory(token, op); err != nil {
			return fmt.Errorf("failed to seed operatory %s: %w", op.Name, err)
		}
		summary.OperatoriesCount++
	}
	return nil
}
