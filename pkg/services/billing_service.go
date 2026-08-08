package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage"
)

// BillingService exposes billing operations to the Wails frontend.
type BillingService struct {
	claimRepo  storage.ClaimRepository
	payRepo    storage.PaymentRepository
	bundleRepo storage.TreatmentBundleRepository
}

func NewBillingService(
	claimRepo storage.ClaimRepository,
	payRepo storage.PaymentRepository,
	bundleRepo storage.TreatmentBundleRepository,
) *BillingService {
	return &BillingService{
		claimRepo:  claimRepo,
		payRepo:    payRepo,
		bundleRepo: bundleRepo,
	}
}

// ─── Claims ──────────────────────────────────────────────────────────────────

// CreateClaim creates a new insurance claim.
func (s *BillingService) CreateClaim(c *domain.Claim) (*domain.Claim, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: claim cannot be nil", storage.ErrInvalidInput)
	}
	if c.ID == "" {
		c.ID = fmt.Sprintf("claim_%d", time.Now().UnixNano())
	}
	// Assign IDs to any line items that are missing them
	for i := range c.LineItems {
		if c.LineItems[i].ID == "" {
			c.LineItems[i].ID = fmt.Sprintf("li_%d_%d", time.Now().UnixNano(), i)
		}
	}
	if err := s.claimRepo.Create(context.Background(), c); err != nil {
		return nil, fmt.Errorf("failed to create claim: %w", err)
	}
	return c, nil
}

// GetClaim retrieves a claim by ID.
func (s *BillingService) GetClaim(id string) (*domain.Claim, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: claim ID is required", storage.ErrInvalidInput)
	}
	c, err := s.claimRepo.GetByID(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to get claim: %w", err)
	}
	return c, nil
}

// ListClaims returns all claims, optionally filtered by patient ID.
func (s *BillingService) ListClaims(patientID string) ([]*domain.Claim, error) {
	claims, err := s.claimRepo.List(context.Background(), patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list claims: %w", err)
	}
	return claims, nil
}

// UpdateClaim updates an existing claim.
func (s *BillingService) UpdateClaim(c *domain.Claim) (*domain.Claim, error) {
	if c == nil || c.ID == "" {
		return nil, fmt.Errorf("%w: claim and ID are required", storage.ErrInvalidInput)
	}
	// Ensure all line items have IDs
	for i := range c.LineItems {
		if c.LineItems[i].ID == "" {
			c.LineItems[i].ID = fmt.Sprintf("li_%d_%d", time.Now().UnixNano(), i)
		}
	}
	if err := s.claimRepo.Update(context.Background(), c); err != nil {
		return nil, fmt.Errorf("failed to update claim: %w", err)
	}
	return c, nil
}

// DeleteClaim removes a claim by ID.
func (s *BillingService) DeleteClaim(id string) error {
	if id == "" {
		return fmt.Errorf("%w: claim ID is required", storage.ErrInvalidInput)
	}
	return s.claimRepo.Delete(context.Background(), id)
}

// ─── Payments ────────────────────────────────────────────────────────────────

// RecordPayment records a new payment from a patient or insurer.
func (s *BillingService) RecordPayment(p *domain.Payment) (*domain.Payment, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: payment cannot be nil", storage.ErrInvalidInput)
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("pay_%d", time.Now().UnixNano())
	}
	if err := s.payRepo.Create(context.Background(), p); err != nil {
		return nil, fmt.Errorf("failed to record payment: %w", err)
	}
	return p, nil
}

// ListPayments returns all payments for a patient.
func (s *BillingService) ListPayments(patientID string) ([]*domain.Payment, error) {
	payments, err := s.payRepo.List(context.Background(), patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}
	return payments, nil
}

// DeletePayment removes a payment record by ID.
func (s *BillingService) DeletePayment(id string) error {
	if id == "" {
		return fmt.Errorf("%w: payment ID is required", storage.ErrInvalidInput)
	}
	return s.payRepo.Delete(context.Background(), id)
}

// GetPatientBalance computes a patient's outstanding balance:
// total_billed (sum of all claim line item fees) minus total_paid (sum of all payments).
func (s *BillingService) GetPatientBalance(patientID string) (*domain.PatientBalance, error) {
	if patientID == "" {
		return nil, fmt.Errorf("%w: patient ID is required", storage.ErrInvalidInput)
	}

	billed, err := s.claimRepo.GetTotalBilled(context.Background(), patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to compute total billed: %w", err)
	}

	paid, err := s.payRepo.GetTotalPaid(context.Background(), patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to compute total paid: %w", err)
	}

	return &domain.PatientBalance{
		PatientID:   patientID,
		TotalBilled: billed,
		TotalPaid:   paid,
		Outstanding: billed - paid,
	}, nil
}

// ─── Treatment Bundles ────────────────────────────────────────────────────────

// CreateBundle creates a new clinic-wide treatment bundle template.
func (s *BillingService) CreateBundle(b *domain.TreatmentBundle) (*domain.TreatmentBundle, error) {
	if b == nil {
		return nil, fmt.Errorf("%w: bundle cannot be nil", storage.ErrInvalidInput)
	}
	if b.ID == "" {
		b.ID = fmt.Sprintf("bundle_%d", time.Now().UnixNano())
	}
	b.Shortname = strings.ToLower(strings.TrimSpace(b.Shortname))
	if b.Shortname == "" {
		return nil, fmt.Errorf("%w: bundle shortname is required", storage.ErrInvalidInput)
	}

	// Recompute total_fee from items
	var total float64
	for _, item := range b.Items {
		total += item.DefaultFee
	}
	b.TotalFee = total

	if err := s.bundleRepo.Create(context.Background(), b); err != nil {
		return nil, fmt.Errorf("failed to create bundle: %w", err)
	}
	return b, nil
}

// GetBundle retrieves a bundle by ID.
func (s *BillingService) GetBundle(id string) (*domain.TreatmentBundle, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: bundle ID is required", storage.ErrInvalidInput)
	}
	return s.bundleRepo.GetByID(context.Background(), id)
}

// GetBundleByShortname retrieves a bundle by its shortname — used for fast data entry lookup.
func (s *BillingService) GetBundleByShortname(shortname string) (*domain.TreatmentBundle, error) {
	if shortname == "" {
		return nil, fmt.Errorf("%w: shortname is required", storage.ErrInvalidInput)
	}
	return s.bundleRepo.GetByShortname(context.Background(), strings.ToLower(strings.TrimSpace(shortname)))
}

// ListBundles returns all bundle templates.
func (s *BillingService) ListBundles() ([]*domain.TreatmentBundle, error) {
	bundles, err := s.bundleRepo.List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list bundles: %w", err)
	}
	return bundles, nil
}

// UpdateBundle updates an existing bundle template.
func (s *BillingService) UpdateBundle(b *domain.TreatmentBundle) (*domain.TreatmentBundle, error) {
	if b == nil || b.ID == "" {
		return nil, fmt.Errorf("%w: bundle and ID are required", storage.ErrInvalidInput)
	}
	b.Shortname = strings.ToLower(strings.TrimSpace(b.Shortname))

	var total float64
	for _, item := range b.Items {
		total += item.DefaultFee
	}
	b.TotalFee = total

	if err := s.bundleRepo.Update(context.Background(), b); err != nil {
		return nil, fmt.Errorf("failed to update bundle: %w", err)
	}
	return b, nil
}

// DeleteBundle removes a bundle template by ID.
func (s *BillingService) DeleteBundle(id string) error {
	if id == "" {
		return fmt.Errorf("%w: bundle ID is required", storage.ErrInvalidInput)
	}
	return s.bundleRepo.Delete(context.Background(), id)
}
