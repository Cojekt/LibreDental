package services

import (
	"context"
	"fmt"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type PatientService struct {
	repo         storage.PatientRepository
	auditService *AuditService
}

func NewPatientService(repo storage.PatientRepository, auditService *AuditService) *PatientService {
	return &PatientService{repo: repo, auditService: auditService}
}

func (s *PatientService) ListPatients(token string, query string, status string) ([]*domain.Patient, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	patients, _, err := s.repo.List(context.Background(), domain.PatientFilter{Query: query, Status: status})
	if err != nil {
		return nil, fmt.Errorf("failed to list patients: %w", err)
	}
	if patients == nil {
		patients = []*domain.Patient{}
	}
	_ = s.auditService.LogAction(token, domain.AuditActionRead, "patient_demographics", "Listed patients")
	return patients, nil
}

func (s *PatientService) GetPatient(token string, id string) (*domain.Patient, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	p, err := s.repo.GetByID(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient: %w", err)
	}
	_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, p.ID, "patient_demographics", "Viewed patient record")
	return p, nil
}

func (s *PatientService) CreatePatient(token string, p *domain.Patient) (*domain.Patient, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	err := s.repo.Create(context.Background(), p)
	if err != nil {
		return nil, fmt.Errorf("failed to create patient: %w", err)
	}
	_ = s.auditService.LogPatientAction(token, domain.AuditActionCreate, p.ID, "patient_demographics", "Created new patient record")
	return p, nil
}

func (s *PatientService) UpdatePatient(token string, p *domain.Patient) (*domain.Patient, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	err := s.repo.Update(context.Background(), p)
	if err != nil {
		return nil, fmt.Errorf("failed to update patient: %w", err)
	}
	_ = s.auditService.LogPatientAction(token, domain.AuditActionUpdate, p.ID, "patient_demographics", "Updated patient record")
	return p, nil
}

func (s *PatientService) ArchivePatient(token string, id string) error {
	p, err := s.GetPatient(token, id)
	if err != nil {
		return err
	}
	p.Status = domain.StatusArchived
	_, err = s.UpdatePatient(token, p)
	if err == nil {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionUpdate, p.ID, "patient_demographics", "Archived patient record")
	}
	return err
}
