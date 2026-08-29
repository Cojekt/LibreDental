package services

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type ChartService struct {
	chartRepo    storage.ChartRepository
	auditService *AuditService
}

func NewChartService(chartRepo storage.ChartRepository, auditService *AuditService) *ChartService {
	return &ChartService{chartRepo: chartRepo, auditService: auditService}
}

func (s *ChartService) GetPatientChart(token string, patientID string) (*domain.DentalChart, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	if patientID == "" {
		return nil, fmt.Errorf("%w: patient ID is required", storage.ErrInvalidInput)
	}

	chart, err := s.chartRepo.GetChart(context.Background(), patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient dental chart: %w", err)
	}

	if chart.Conditions == nil {
		chart.Conditions = []domain.ToothCondition{}
	}

	_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, patientID, "dental_chart", "Viewed dental chart")
	return chart, nil
}

func (s *ChartService) SaveToothCondition(token string, c *domain.ToothCondition) (*domain.ToothCondition, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	if c == nil {
		return nil, fmt.Errorf("%w: tooth condition cannot be nil", storage.ErrInvalidInput)
	}

	action := domain.AuditActionUpdate
	if c.ID == "" {
		c.ID = fmt.Sprintf("cond_%d", time.Now().UnixNano())
		action = domain.AuditActionCreate
	} else {
		chart, err := s.chartRepo.GetChart(context.Background(), c.PatientID)
		exists := false
		if err == nil && chart != nil {
			for _, existing := range chart.Conditions {
				if existing.ID == c.ID {
					exists = true
					break
				}
			}
		}
		if !exists {
			action = domain.AuditActionCreate
		}
	}

	err := s.chartRepo.SaveCondition(context.Background(), c)
	if err != nil {
		return nil, fmt.Errorf("failed to save tooth condition: %w", err)
	}

	if err := s.auditService.LogPatientAction(token, action, c.PatientID, "dental_chart", "Saved tooth condition"); err != nil {
		fmt.Printf("Warning: failed to log audit action: %v\n", err)
	}
	return c, nil
}

func (s *ChartService) DeleteToothCondition(token string, id string, patientID string) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}
	if id == "" {
		return fmt.Errorf("%w: condition ID is required", storage.ErrInvalidInput)
	}

	err := s.chartRepo.DeleteCondition(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to delete tooth condition: %w", err)
	}

	if err := s.auditService.LogPatientAction(token, domain.AuditActionDelete, patientID, "dental_chart", "Deleted tooth condition"); err != nil {
		fmt.Printf("Warning: failed to log audit action: %v\n", err)
	}
	return nil
}
