package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func seedPracticeConfig(ctx context.Context, practiceConfigRepo *sqlite.PracticeConfigRepository, now time.Time, summary *SeedSummary) error {
	config := &domain.PracticeConfig{
		ID:            1,
		ClinicName:    "Apex Dental Studio",
		Tagline:       "Modern Dental Care & Implant Center",
		TaxID:         "94-1234567",
		LicenseNumber: "DEN-CA-884920",
		Phone:         "(555) 234-5678",
		Email:         "info@apexdentalstudio.com",
		Website:       "https://apexdentalstudio.example.com",
		AddressLine1:  "101 Dental Plaza, Suite 200",
		City:          "San Francisco",
		StateProvince: "CA",
		PostalCode:    "94105",
		CountryCode:   domain.CountryUS,
		Currency:      "USD",
		ToothSystem:   domain.ToothSystemUniversal,
		DateFormat:    "MM/DD/YYYY",
		BusinessHours: domain.DefaultBusinessHours(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := practiceConfigRepo.Save(ctx, config); err != nil {
		return fmt.Errorf("failed to seed practice config: %w", err)
	}
	summary.PracticeConfigured = true
	return nil
}

func seedProviders(ctx context.Context, practiceConfigRepo *sqlite.PracticeConfigRepository, now time.Time, summary *SeedSummary) error {
	providers := []*domain.Provider{
		{
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
		},
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
		if err := practiceConfigRepo.SaveProvider(ctx, p); err != nil {
			return fmt.Errorf("failed to seed provider %s: %w", p.Name, err)
		}
		summary.ProvidersCount++
	}
	return nil
}

func seedOperatories(ctx context.Context, practiceConfigRepo *sqlite.PracticeConfigRepository, now time.Time, summary *SeedSummary) error {
	operatories := []*domain.Operatory{
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
		if err := practiceConfigRepo.SaveOperatory(ctx, op); err != nil {
			return fmt.Errorf("failed to seed operatory %s: %w", op.Name, err)
		}
		summary.OperatoriesCount++
	}
	return nil
}
