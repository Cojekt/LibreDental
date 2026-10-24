package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type InsuranceService struct {
	repo  storage.InsuranceRepository
	audit *AuditService
}

func NewInsuranceService(repo storage.InsuranceRepository, audit *AuditService) *InsuranceService {
	return &InsuranceService{repo: repo, audit: audit}
}
func AttachInsurancePricing(chart *ChartService, billing *BillingService, repo storage.InsuranceRepository) {
	chart.pricing = repo
	billing.pricing = repo
}
func (s *InsuranceService) authorize(token string) error {
	if s.audit.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}
	return nil
}
func (s *InsuranceService) mutation(token, resource, details string) error {
	if err := s.authorize(token); err != nil {
		return err
	}
	return s.audit.LogAction(token, domain.AuditActionUpdate, resource, details)
}
func (s *InsuranceService) ListProviders(token string) ([]*domain.InsuranceProvider, error) {
	if err := s.authorize(token); err != nil {
		return nil, err
	}
	return s.repo.ListProviders(context.Background())
}
func insuranceDetails(value any) string { b, _ := json.Marshal(value); return string(b) }
func (s *InsuranceService) SaveProvider(token string, p *domain.InsuranceProvider) (*domain.InsuranceProvider, error) {
	if err := s.mutation(token, "insurance_provider", "Requested insurance provider change: "+insuranceDetails(p)); err != nil {
		return nil, err
	}
	action := domain.AuditActionUpdate
	if p != nil && p.ID == "" {
		action = domain.AuditActionCreate
	}
	if err := s.repo.SaveProvider(context.Background(), p); err != nil {
		return nil, errors.Join(err, s.audit.LogAction(token, action, "insurance_provider", "Insurance provider change failed: "+err.Error()))
	}
	if err := s.audit.LogAction(token, action, "insurance_provider", "Saved insurance provider: "+insuranceDetails(p)); err != nil {
		return nil, err
	}
	return p, nil
}
func (s *InsuranceService) ListFees(token, plan, cc string) ([]*domain.InsuranceFee, error) {
	if err := s.authorize(token); err != nil {
		return nil, err
	}
	return s.repo.ListFees(context.Background(), plan, domain.CountryCode(cc))
}
func (s *InsuranceService) SaveFee(token string, f *domain.InsuranceFee) error {
	if err := s.mutation(token, "insurance_fee", "Requested plan procedure fee change: "+insuranceDetails(f)); err != nil {
		return err
	}
	if err := s.repo.SaveFee(context.Background(), f); err != nil {
		return errors.Join(err, s.audit.LogAction(token, domain.AuditActionUpdate, "insurance_fee", "Insurance fee change failed: "+err.Error()))
	}
	return s.audit.LogAction(token, domain.AuditActionUpdate, "insurance_fee", "Saved plan procedure fee: "+insuranceDetails(f))
}
func (s *InsuranceService) DeleteFee(token, plan, cc, code string) error {
	if err := s.authorize(token); err != nil {
		return err
	}
	if err := s.audit.LogAction(token, domain.AuditActionDelete, "insurance_fee", "Requested plan procedure fee removal: "+plan+" / "+cc+" / "+code); err != nil {
		return err
	}
	if err := s.repo.DeleteFee(context.Background(), plan, domain.CountryCode(cc), code); err != nil {
		return errors.Join(err, s.audit.LogAction(token, domain.AuditActionDelete, "insurance_fee", "Insurance fee removal failed: "+err.Error()))
	}
	return s.audit.LogAction(token, domain.AuditActionDelete, "insurance_fee", "Removed plan procedure fee: "+plan+" / "+cc+" / "+code)
}
func (s *InsuranceService) Preview(token string, p *domain.PricingSource) (*domain.PricingSource, error) {
	if err := s.authorize(token); err != nil {
		return nil, err
	}
	return s.repo.Preview(context.Background(), p)
}
