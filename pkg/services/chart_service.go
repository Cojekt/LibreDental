package services

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage"
)

type ChartService struct {
	chartRepo storage.ChartRepository
}

func NewChartService(chartRepo storage.ChartRepository) *ChartService {
	return &ChartService{chartRepo: chartRepo}
}

func (s *ChartService) GetPatientChart(patientID string) (*domain.DentalChart, error) {
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

	return chart, nil
}

func (s *ChartService) SaveToothCondition(c *domain.ToothCondition) (*domain.ToothCondition, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: tooth condition cannot be nil", storage.ErrInvalidInput)
	}

	if c.ID == "" {
		c.ID = fmt.Sprintf("cond_%d", time.Now().UnixNano())
	}

	err := s.chartRepo.SaveCondition(context.Background(), c)
	if err != nil {
		return nil, fmt.Errorf("failed to save tooth condition: %w", err)
	}

	return c, nil
}

func (s *ChartService) DeleteToothCondition(id string) error {
	if id == "" {
		return fmt.Errorf("%w: condition ID is required", storage.ErrInvalidInput)
	}

	err := s.chartRepo.DeleteCondition(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to delete tooth condition: %w", err)
	}

	return nil
}
