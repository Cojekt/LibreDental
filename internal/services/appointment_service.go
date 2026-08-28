package services

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type AppointmentService struct {
	repo         storage.AppointmentRepository
	auditService *AuditService
}

func NewAppointmentService(repo storage.AppointmentRepository, auditService *AuditService) *AppointmentService {
	return &AppointmentService{repo: repo, auditService: auditService}
}

func (s *AppointmentService) ListAppointments(token string, filter domain.AppointmentFilter) ([]*domain.Appointment, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	appts, err := s.repo.List(context.Background(), filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list appointments: %w", err)
	}
	if appts == nil {
		appts = []*domain.Appointment{}
	}
	_ = s.auditService.LogAction(token, domain.AuditActionRead, "appointment", "Listed appointments")
	return appts, nil
}

func (s *AppointmentService) GetAppointment(token string, id string) (*domain.Appointment, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	appt, err := s.repo.GetByID(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointment: %w", err)
	}
	_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, appt.PatientID, "appointment", "Viewed appointment")
	return appt, nil
}

func (s *AppointmentService) CreateAppointment(token string, a *domain.Appointment) (*domain.Appointment, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
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
	if err := s.auditService.LogPatientAction(token, domain.AuditActionCreate, a.PatientID, "appointment", "Created appointment"); err != nil {
		return nil, fmt.Errorf("failed to log audit action: %w", err)
	}
	return a, nil
}

func (s *AppointmentService) UpdateAppointment(token string, a *domain.Appointment) (*domain.Appointment, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	if a == nil {
		return nil, fmt.Errorf("%w: appointment cannot be nil", storage.ErrInvalidInput)
	}
	err := s.repo.Update(context.Background(), a)
	if err != nil {
		return nil, fmt.Errorf("failed to update appointment: %w", err)
	}
	if err := s.auditService.LogPatientAction(token, domain.AuditActionUpdate, a.PatientID, "appointment", "Updated appointment"); err != nil {
		return nil, fmt.Errorf("failed to log audit action: %w", err)
	}
	return a, nil
}

func (s *AppointmentService) DeleteAppointment(token string, id string) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}
	appt, err := s.repo.GetByID(context.Background(), id)
	if err != nil {
		return err
	}
	err = s.repo.Delete(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to delete appointment: %w", err)
	}
	if err := s.auditService.LogPatientAction(token, domain.AuditActionDelete, appt.PatientID, "appointment", "Deleted appointment"); err != nil {
		return fmt.Errorf("failed to log audit action: %w", err)
	}
	return nil
}

func (s *AppointmentService) UpdateAppointmentStatus(token string, id string, status string) (*domain.Appointment, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	appt, err := s.repo.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	appt.Status = domain.AppointmentStatus(status)
	return s.UpdateAppointment(token, appt)
}
