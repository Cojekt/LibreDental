package services

import (
	"context"
	"fmt"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage"
)

type PatientService struct {
	repo storage.PatientRepository
}

func NewPatientService(repo storage.PatientRepository) *PatientService {
	return &PatientService{repo: repo}
}

func (s *PatientService) ListPatients(query string, status string) ([]*domain.Patient, error) {
	patients, _, err := s.repo.List(context.Background(), domain.PatientFilter{Query: query, Status: status})
	if err != nil {
		return nil, fmt.Errorf("failed to list patients: %w", err)
	}
	if patients == nil {
		patients = []*domain.Patient{}
	}
	return patients, nil
}

func (s *PatientService) GetPatient(id string) (*domain.Patient, error) {
	p, err := s.repo.GetByID(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient: %w", err)
	}
	return p, nil
}

func (s *PatientService) CreatePatient(p *domain.Patient) (*domain.Patient, error) {
	err := s.repo.Create(context.Background(), p)
	if err != nil {
		return nil, fmt.Errorf("failed to create patient: %w", err)
	}
	return p, nil
}

func (s *PatientService) UpdatePatient(p *domain.Patient) (*domain.Patient, error) {
	err := s.repo.Update(context.Background(), p)
	if err != nil {
		return nil, fmt.Errorf("failed to update patient: %w", err)
	}
	return p, nil
}

func (s *PatientService) ArchivePatient(id string) error {
	p, err := s.GetPatient(id)
	if err != nil {
		return err
	}
	p.Status = domain.StatusArchived
	_, err = s.UpdatePatient(p)
	return err
}
