package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

type TimecardService struct {
	timecardRepo       *sqlite.TimecardRepository
	practiceConfigRepo *sqlite.PracticeConfigRepository
}

func NewTimecardService(timecardRepo *sqlite.TimecardRepository, practiceConfigRepo *sqlite.PracticeConfigRepository) *TimecardService {
	return &TimecardService{
		timecardRepo:       timecardRepo,
		practiceConfigRepo: practiceConfigRepo,
	}
}

// ClockIn starts a new timecard for the given provider.
func (s *TimecardService) ClockIn(providerID string) (*domain.Timecard, error) {
	ctx := context.Background()

	// Check if already clocked in
	active, err := s.timecardRepo.GetActiveTimecard(ctx, providerID)
	if err == nil && active != nil {
		return nil, fmt.Errorf("provider is already clocked in (Timecard ID: %s)", active.ID)
	}

	// Fetch provider to get the current hourly rate
	providers, err := s.practiceConfigRepo.ListProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch providers: %w", err)
	}

	var provider *domain.Provider
	for _, p := range providers {
		if p.ID == providerID {
			provider = p
			break
		}
	}

	if provider == nil {
		return nil, fmt.Errorf("provider not found")
	}

	// Create new timecard
	t := &domain.Timecard{
		ID:         fmt.Sprintf("tc_%d", time.Now().UnixNano()),
		ProviderID: providerID,
		ClockIn:    time.Now(),
		HourlyRate: provider.HourlyRate,
	}

	if err := s.timecardRepo.SaveTimecard(ctx, t); err != nil {
		return nil, fmt.Errorf("failed to save timecard: %w", err)
	}

	return t, nil
}

// ClockOut ends the active timecard for the given provider.
func (s *TimecardService) ClockOut(providerID string) (*domain.Timecard, error) {
	ctx := context.Background()

	active, err := s.timecardRepo.GetActiveTimecard(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("could not find active timecard to clock out: %w", err)
	}

	now := time.Now()
	active.ClockOut = &now

	// Calculate total minutes using 1-minute precision rounding
	duration := active.ClockOut.Sub(active.ClockIn)
	minutes := int64(math.Round(duration.Minutes()))
	active.TotalMinutes = minutes
	active.TotalPay = (active.TotalMinutes*active.HourlyRate + 30) / 60

	if err := s.timecardRepo.SaveTimecard(ctx, active); err != nil {
		return nil, fmt.Errorf("failed to save timecard on clock out: %w", err)
	}

	return active, nil
}

// GetActiveTimecard returns the active timecard if the provider is clocked in.
func (s *TimecardService) GetActiveTimecard(providerID string) (*domain.Timecard, error) {
	ctx := context.Background()
	tc, err := s.timecardRepo.GetActiveTimecard(ctx, providerID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, nil // Not an error to not be clocked in
		}
		return nil, fmt.Errorf("failed to fetch active timecard: %w", err)
	}
	return tc, nil
}

// ListTimecards returns timecards for a specific provider or all providers if "all" is passed.
func (s *TimecardService) ListTimecards(providerID string, startDateStr string, endDateStr string) ([]*domain.Timecard, error) {
	ctx := context.Background()

	var startDate *time.Time
	if startDateStr != "" {
		if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = &t
		}
	}

	var endDate *time.Time
	if endDateStr != "" {
		if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = &t
		}
	}

	return s.timecardRepo.ListTimecards(ctx, providerID, startDate, endDate)
}

// EditTimecardHours allows manual overriding of a timecard's recorded minutes.
func (s *TimecardService) EditTimecardHours(timecardID string, providerID string, newMinutes int64) error {
	ctx := context.Background()
	timecards, err := s.timecardRepo.ListTimecards(ctx, providerID, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch timecards: %w", err)
	}

	for _, t := range timecards {
		if t.ID == timecardID {
			t.TotalMinutes = newMinutes
			t.TotalPay = (newMinutes*t.HourlyRate + 30) / 60
			t.IsManual = true
			if err := s.timecardRepo.SaveTimecard(ctx, t); err != nil {
				return fmt.Errorf("failed to save edited timecard: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("timecard %s not found for provider %s", timecardID, providerID)
}

// CreateManualTimecard allows creating retroactive time entries.
func (s *TimecardService) CreateManualTimecard(providerID string, minutes int64, date string) error {
	ctx := context.Background()

	providers, err := s.practiceConfigRepo.ListProviders(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch providers: %w", err)
	}

	var provider *domain.Provider
	for _, p := range providers {
		if p.ID == providerID {
			provider = p
			break
		}
	}

	if provider == nil {
		return fmt.Errorf("provider not found")
	}

	parsedDate, err := time.Parse(time.RFC3339, date)
	if err != nil {
		parsedDate = time.Now()
	}

	t := &domain.Timecard{
		ID:         fmt.Sprintf("tc_%d", time.Now().UnixNano()),
		ProviderID: providerID,
		ClockIn:    parsedDate,
		ClockOut:   &parsedDate,
		HourlyRate:   provider.HourlyRate,
		TotalMinutes: minutes,
		TotalPay:     (minutes*provider.HourlyRate + 30) / 60,
		IsManual:     true,
	}

	if err := s.timecardRepo.SaveTimecard(ctx, t); err != nil {
		return fmt.Errorf("failed to save manual timecard: %w", err)
	}
	return nil
}

// GetTotalOwed returns the total unpaid amount owed to a provider.
func (s *TimecardService) GetTotalOwed(providerID string) (int64, error) {
	ctx := context.Background()
	return s.timecardRepo.GetTotalOwed(ctx, providerID)
}

// DeleteTimecard removes a specific timecard record.
func (s *TimecardService) DeleteTimecard(id string) error {
	ctx := context.Background()
	return s.timecardRepo.DeleteTimecard(ctx, id)
}

// PaySalary marks all unpaid timecards for a provider as paid.
func (s *TimecardService) PaySalary(providerID string) error {
	ctx := context.Background()
	return s.timecardRepo.MarkTimecardsPaid(ctx, providerID)
}
