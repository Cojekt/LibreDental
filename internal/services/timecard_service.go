package services

import (
	"context"
	"errors"
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
	auditService       *AuditService
}

func NewTimecardService(timecardRepo *sqlite.TimecardRepository, practiceConfigRepo *sqlite.PracticeConfigRepository, auditService *AuditService) *TimecardService {
	return &TimecardService{
		timecardRepo:       timecardRepo,
		practiceConfigRepo: practiceConfigRepo,
		auditService:       auditService,
	}
}

// logAction records an audit entry when a session is available. Clocking in/out and
// payroll aren't gated behind staff login in this app, so this is best-effort
// attribution, not an access check.
func (s *TimecardService) logAction(token string, action domain.AuditAction, resource string, details string) {
	if s.auditService == nil {
		return
	}
	if err := s.auditService.LogAction(token, action, resource, details); err != nil && !errors.Is(err, ErrUnauthorized) {
		fmt.Printf("Warning: failed to log audit action: %v\n", err)
	}
}

// ClockIn starts a new timecard for the given provider.
func (s *TimecardService) ClockIn(token string, providerID string) (*domain.Timecard, error) {
	ctx := context.Background()

	// Check if already clocked in
	active, err := s.timecardRepo.GetActiveTimecard(ctx, providerID)
	if err != nil && err != storage.ErrNotFound {
		return nil, fmt.Errorf("failed to check active timecard: %w", err)
	}
	if active != nil {
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
	if !provider.IsActive {
		return nil, fmt.Errorf("provider is inactive")
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

	s.logAction(token, domain.AuditActionCreate, "timecard", fmt.Sprintf("Clocked in provider %s", providerID))
	return t, nil
}

// ClockOut ends the active timecard for the given provider.
func (s *TimecardService) ClockOut(token string, providerID string) (*domain.Timecard, error) {
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

	s.logAction(token, domain.AuditActionUpdate, "timecard", fmt.Sprintf("Clocked out provider %s", providerID))
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
		t, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid start date: %w", err)
		}
		startDate = &t
	}

	var endDate *time.Time
	if endDateStr != "" {
		t, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid end date: %w", err)
		}
		endDate = &t
	}

	return s.timecardRepo.ListTimecards(ctx, providerID, startDate, endDate)
}

// EditTimecardHours allows manual overriding of a timecard's recorded minutes.
func (s *TimecardService) EditTimecardHours(token string, timecardID string, providerID string, newMinutes int64) error {
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
			s.logAction(token, domain.AuditActionUpdate, "timecard", fmt.Sprintf("Edited timecard %s hours for provider %s", timecardID, providerID))
			return nil
		}
	}
	return fmt.Errorf("timecard %s not found for provider %s", timecardID, providerID)
}

// CreateManualTimecard allows creating retroactive time entries.
func (s *TimecardService) CreateManualTimecard(token string, providerID string, minutes int64, date string) error {
	ctx := context.Background()

	if minutes <= 0 {
		return fmt.Errorf("minutes must be greater than zero")
	}

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
		return fmt.Errorf("invalid date: %w", err)
	}

	t := &domain.Timecard{
		ID:           fmt.Sprintf("tc_%d", time.Now().UnixNano()),
		ProviderID:   providerID,
		ClockIn:      parsedDate,
		ClockOut:     &parsedDate,
		HourlyRate:   provider.HourlyRate,
		TotalMinutes: minutes,
		TotalPay:     (minutes*provider.HourlyRate + 30) / 60,
		IsManual:     true,
	}

	if err := s.timecardRepo.SaveTimecard(ctx, t); err != nil {
		return fmt.Errorf("failed to save manual timecard: %w", err)
	}
	s.logAction(token, domain.AuditActionCreate, "timecard", fmt.Sprintf("Created manual timecard for provider %s", providerID))
	return nil
}

// GetTotalOwed returns the total unpaid amount owed to a provider.
func (s *TimecardService) GetTotalOwed(providerID string) (int64, error) {
	ctx := context.Background()
	return s.timecardRepo.GetTotalOwed(ctx, providerID)
}

// DeleteTimecard removes a specific timecard record.
func (s *TimecardService) DeleteTimecard(token string, id string) error {
	ctx := context.Background()
	if err := s.timecardRepo.DeleteTimecard(ctx, id); err != nil {
		return err
	}
	s.logAction(token, domain.AuditActionDelete, "timecard", fmt.Sprintf("Deleted timecard %s", id))
	return nil
}

// PaySalary marks all unpaid timecards for a provider as paid.
func (s *TimecardService) PaySalary(token string, providerID string) error {
	ctx := context.Background()
	if err := s.timecardRepo.MarkTimecardsPaid(ctx, providerID); err != nil {
		return err
	}
	s.logAction(token, domain.AuditActionUpdate, "timecard", fmt.Sprintf("Paid salary for provider %s", providerID))
	return nil
}
