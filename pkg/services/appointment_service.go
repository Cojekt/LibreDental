package services

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage"
)

type AppointmentService struct {
	repo storage.AppointmentRepository
}

func NewAppointmentService(repo storage.AppointmentRepository) *AppointmentService {
	return &AppointmentService{repo: repo}
}

func (s *AppointmentService) ListAppointments(filter domain.AppointmentFilter) ([]*domain.Appointment, error) {
	appts, err := s.repo.List(context.Background(), filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list appointments: %w", err)
	}
	if appts == nil {
		appts = []*domain.Appointment{}
	}
	return appts, nil
}

func (s *AppointmentService) GetAppointment(id string) (*domain.Appointment, error) {
	appt, err := s.repo.GetByID(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointment: %w", err)
	}
	return appt, nil
}

func (s *AppointmentService) CreateAppointment(a *domain.Appointment) (*domain.Appointment, error) {
	if a == nil {
		return nil, fmt.Errorf("%w: appointment cannot be nil", storage.ErrInvalidInput)
	}
	if a.ID == "" {
		a.ID = fmt.Sprintf("appt_%d", time.Now().UnixNano())
	}
	err := s.repo.Create(context.Background(), a)
	if err != nil {
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}
	return a, nil
}

func (s *AppointmentService) UpdateAppointment(a *domain.Appointment) (*domain.Appointment, error) {
	if a == nil {
		return nil, fmt.Errorf("%w: appointment cannot be nil", storage.ErrInvalidInput)
	}
	err := s.repo.Update(context.Background(), a)
	if err != nil {
		return nil, fmt.Errorf("failed to update appointment: %w", err)
	}
	return a, nil
}

func (s *AppointmentService) DeleteAppointment(id string) error {
	err := s.repo.Delete(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to delete appointment: %w", err)
	}
	return nil
}

func (s *AppointmentService) UpdateAppointmentStatus(id string, status string) (*domain.Appointment, error) {
	appt, err := s.GetAppointment(id)
	if err != nil {
		return nil, err
	}
	appt.Status = domain.AppointmentStatus(status)
	return s.UpdateAppointment(appt)
}
