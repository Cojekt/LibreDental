package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// BillingService exposes billing operations to the Wails frontend.
type BillingService struct {
	claimRepo    storage.ClaimRepository
	payRepo      storage.PaymentRepository
	bundleRepo   storage.TreatmentBundleRepository
	procRepo     storage.ProcedureCodeRepository
	feeRepo      storage.FeeScheduleRepository
	chartRepo    storage.ChartRepository
	secrets      *SecretsService
	auditService *AuditService
	providers    map[string]domain.ClaimProvider
}

func NewBillingService(
	claimRepo storage.ClaimRepository,
	payRepo storage.PaymentRepository,
	bundleRepo storage.TreatmentBundleRepository,
	procRepo storage.ProcedureCodeRepository,
	feeRepo storage.FeeScheduleRepository,
	chartRepo storage.ChartRepository,
	secrets *SecretsService,
	auditService *AuditService,
) *BillingService {
	return &BillingService{
		claimRepo:    claimRepo,
		payRepo:      payRepo,
		bundleRepo:   bundleRepo,
		procRepo:     procRepo,
		feeRepo:      feeRepo,
		chartRepo:    chartRepo,
		secrets:      secrets,
		auditService: auditService,
		providers:    make(map[string]domain.ClaimProvider),
	}
}

// ─── Provider Registry ───────────────────────────────────────────────────────

// RegisterClaimProvider registers a new claim provider for use.
// Exposed as a function rather than a method so Wails does not bind it.
func RegisterClaimProvider(s *BillingService, p domain.ClaimProvider) {
	s.registerProvider(p)
}

// registerProvider registers a new claim provider for use.
func (s *BillingService) registerProvider(p domain.ClaimProvider) {
	if p != nil {
		s.providers[p.Name()] = p
	}
}

// ListProviders returns a list of registered provider names.
func (s *BillingService) ListProviders() []string {
	var names []string
	for name := range s.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetProviderConfig retrieves configuration for a specific provider.
func (s *BillingService) GetProviderConfig(providerName string) (map[string]string, error) {
	if providerName == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	return s.secrets.GetProviderConfig(providerName)
}

// SetProviderConfig saves configuration for a specific provider.
func (s *BillingService) SetProviderConfig(providerName string, config map[string]string) error {
	if providerName == "" {
		return fmt.Errorf("provider name is required")
	}
	return s.secrets.SetProviderConfig(providerName, config)
}

// ─── Claims ──────────────────────────────────────────────────────────────────

// CreateClaim creates a new insurance claim.
func (s *BillingService) CreateClaim(token string, c *domain.Claim) (*domain.Claim, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

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
	if err := s.auditService.LogPatientAction(token, domain.AuditActionCreate, c.PatientID, "claim", "Created claim"); err != nil {
		return nil, fmt.Errorf("claim created but failed to log audit: %w", err)
	}
	return c, nil
}

// GetClaim retrieves a claim by ID.
func (s *BillingService) GetClaim(token string, id string) (*domain.Claim, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	if id == "" {
		return nil, fmt.Errorf("%w: claim ID is required", storage.ErrInvalidInput)
	}
	c, err := s.claimRepo.GetByID(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to get claim: %w", err)
	}
	_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, c.PatientID, "claim", "Viewed claim")
	return c, nil
}

// ListClaims returns all claims, optionally filtered by patient ID.
func (s *BillingService) ListClaims(token string, patientID string) ([]*domain.Claim, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	claims, err := s.claimRepo.List(context.Background(), patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list claims: %w", err)
	}
	if patientID != "" {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, patientID, "claim", "Listed claims")
	} else {
		_ = s.auditService.LogAction(token, domain.AuditActionRead, "claim", "Listed all claims")
	}
	return claims, nil
}

// UpdateClaim updates an existing claim.
func (s *BillingService) UpdateClaim(token string, c *domain.Claim) (*domain.Claim, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	if c == nil || c.ID == "" {
		return nil, fmt.Errorf("%w: claim and ID are required", storage.ErrInvalidInput)
	}
	// Ensure all line items have IDs
	for i := range c.LineItems {
		if c.LineItems[i].ID == "" {
			c.LineItems[i].ID = fmt.Sprintf("li_%d_%d", time.Now().UnixNano(), i)
		}
	}
	existing, err := s.claimRepo.GetByID(context.Background(), c.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing claim: %w", err)
	}
	if err := s.claimRepo.Update(context.Background(), c); err != nil {
		return nil, fmt.Errorf("failed to update claim: %w", err)
	}
	_ = s.auditService.LogPatientAction(token, domain.AuditActionUpdate, c.PatientID, "claim", "Updated claim")
	if existing.PatientID != c.PatientID {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionUpdate, existing.PatientID, "claim", "Reassigned claim to another patient")
	}
	return c, nil
}

// DeleteClaim removes a claim by ID.
func (s *BillingService) DeleteClaim(token string, id string) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}

	if id == "" {
		return fmt.Errorf("%w: claim ID is required", storage.ErrInvalidInput)
	}
	c, err := s.claimRepo.GetByID(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to fetch claim: %w", err)
	}
	err = s.claimRepo.Delete(context.Background(), id)
	if err == nil {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionDelete, c.PatientID, "claim", "Deleted claim")
	}
	return err
}

// SubmitClaimToProvider sends a claim to the specified external provider.
func (s *BillingService) SubmitClaimToProvider(token string, claimID string, providerName string) (*domain.ClaimSubmissionResult, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	if claimID == "" {
		return nil, fmt.Errorf("%w: claim ID is required", storage.ErrInvalidInput)
	}
	if providerName == "" {
		return nil, fmt.Errorf("%w: provider name is required", storage.ErrInvalidInput)
	}

	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", providerName)
	}

	claim, err := s.GetClaim(token, claimID)
	if err != nil {
		return nil, fmt.Errorf("failed to get claim for submission: %w", err)
	}

	if claim.Status != domain.ClaimStatusDraft && claim.Status != domain.ClaimStatusRejected {
		return nil, fmt.Errorf("claim cannot be submitted in status: %s", claim.Status)
	}

	config, err := s.secrets.getRawProviderConfig(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve config for provider %q: %w", providerName, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := provider.SubmitClaim(ctx, claim, config)
	if err != nil {
		return nil, fmt.Errorf("provider %q failed to submit claim: %w", providerName, err)
	}

	if result == nil {
		return nil, fmt.Errorf("provider %q returned nil result", providerName)
	}

	// Update claim status based on result
	if result.Status != "" {
		latestClaim, err := s.GetClaim(token, claimID)
		if err != nil {
			return result, fmt.Errorf("claim submitted but failed to fetch latest claim for status update: %w", err)
		}
		latestClaim.Status = result.Status
		if _, err := s.UpdateClaim(token, latestClaim); err != nil {
			// Log this error in a real app, since the submission succeeded but local update failed
			return result, fmt.Errorf("claim submitted but failed to update local status: %w", err)
		}
	}

	_ = s.auditService.LogPatientAction(token, domain.AuditActionExport, claim.PatientID, "claim", "Submitted claim to provider")
	return result, nil
}

// ─── Payments ────────────────────────────────────────────────────────────────

// RecordPayment records a new payment from a patient or insurer.
func (s *BillingService) RecordPayment(token string, p *domain.Payment) (*domain.Payment, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	if p == nil {
		return nil, fmt.Errorf("%w: payment cannot be nil", storage.ErrInvalidInput)
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("pay_%d", time.Now().UnixNano())
	}
	if err := s.payRepo.Create(context.Background(), p); err != nil {
		return nil, fmt.Errorf("failed to record payment: %w", err)
	}
	_ = s.auditService.LogPatientAction(token, domain.AuditActionCreate, p.PatientID, "payment", "Recorded payment")
	return p, nil
}

// ListPayments returns all payments for a patient.
func (s *BillingService) ListPayments(token string, patientID string) ([]*domain.Payment, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	payments, err := s.payRepo.List(context.Background(), patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}
	if patientID != "" {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, patientID, "payment", "Listed payments")
	} else {
		_ = s.auditService.LogAction(token, domain.AuditActionRead, "payment", "Listed all payments")
	}
	return payments, nil
}

// DeletePayment removes a payment record by ID.
func (s *BillingService) DeletePayment(token string, id string) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}

	if id == "" {
		return fmt.Errorf("%w: payment ID is required", storage.ErrInvalidInput)
	}
	p, err := s.payRepo.GetByID(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to fetch payment: %w", err)
	}
	err = s.payRepo.Delete(context.Background(), id)
	if err == nil {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionDelete, p.PatientID, "payment", "Deleted payment")
	}
	return err
}

// GetPatientBalance computes a patient's outstanding balance:
// total_billed (sum of all claim line item fees) minus total_paid (sum of all payments).
func (s *BillingService) GetPatientBalance(token string, patientID string) (*domain.PatientBalance, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

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

	_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, patientID, "patient_balance", "Viewed patient balance")
	return &domain.PatientBalance{
		PatientID:   patientID,
		TotalBilled: billed,
		TotalPaid:   paid,
		Outstanding: billed - paid,
	}, nil
}

// GetRevenueStats returns payments for a specific date range, for analytics.
func (s *BillingService) GetRevenueStats(token string, startDate, endDate string) ([]*domain.Payment, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	if startDate == "" || endDate == "" {
		return nil, fmt.Errorf("start and end date are required for revenue stats")
	}
	stats, err := s.payRepo.ListByDateRange(context.Background(), startDate, endDate)
	if err == nil {
		_ = s.auditService.LogAction(token, domain.AuditActionRead, "revenue_stats", "Viewed revenue stats")
	}
	return stats, err
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
	var total int64
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

	var total int64
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

// ─── Procedure Codes & Fee Schedules ─────────────────────────────────────────

// ListProcedureCodes returns all active procedure codes for a given country,
// with EffectiveFee populated based on provider or practice custom fee overrides.
func (s *BillingService) ListProcedureCodes(countryCode string, providerID string) ([]*domain.ProcedureCode, error) {
	if countryCode == "" {
		countryCode = "US"
	}
	ctx := context.Background()
	cc := domain.CountryCode(countryCode)

	codes, err := s.procRepo.List(ctx, cc)
	if err != nil {
		return nil, fmt.Errorf("failed to list procedure codes: %w", err)
	}

	for _, pc := range codes {
		effFee, err := s.feeRepo.GetEffectiveFee(ctx, cc, pc.Code, providerID)
		if err == nil && effFee > 0 {
			pc.EffectiveFee = effFee
		} else {
			pc.EffectiveFee = pc.DefaultFee
		}
	}

	return codes, nil
}

// SaveFeeSchedule saves a custom fee override for a procedure code.
func (s *BillingService) SaveFeeSchedule(fee *domain.FeeSchedule) (*domain.FeeSchedule, error) {
	if fee == nil || fee.Code == "" {
		return nil, fmt.Errorf("%w: fee schedule and code are required", storage.ErrInvalidInput)
	}
	if fee.CountryCode == "" {
		fee.CountryCode = domain.CountryUS
	}
	if err := s.feeRepo.Save(context.Background(), fee); err != nil {
		return nil, fmt.Errorf("failed to save fee schedule: %w", err)
	}
	return fee, nil
}

// ListFeeSchedules returns all custom fee schedule overrides for a country and optional provider.
func (s *BillingService) ListFeeSchedules(countryCode string, providerID string) ([]*domain.FeeSchedule, error) {
	if countryCode == "" {
		countryCode = "US"
	}
	return s.feeRepo.ListFeeSchedules(context.Background(), domain.CountryCode(countryCode), providerID)
}

// DeleteFeeSchedule removes a custom fee schedule override.
func (s *BillingService) DeleteFeeSchedule(id string) error {
	if id == "" {
		return fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}
	return s.feeRepo.Delete(context.Background(), id)
}

// ─── Dental Charting Integration ─────────────────────────────────────────────

// CreateClaimFromChartConditions creates an insurance claim directly from charted tooth conditions.
func (s *BillingService) CreateClaimFromChartConditions(token string, patientID string, providerID string, conditionIDs []string) (*domain.Claim, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	if patientID == "" {
		return nil, fmt.Errorf("%w: patient ID is required", storage.ErrInvalidInput)
	}
	if len(conditionIDs) == 0 {
		return nil, fmt.Errorf("%w: at least one condition ID is required", storage.ErrInvalidInput)
	}

	ctx := context.Background()
	chart, err := s.chartRepo.GetChart(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch patient chart: %w", err)
	}

	condMap := make(map[string]domain.ToothCondition)
	for _, c := range chart.Conditions {
		condMap[c.ID] = c
	}

	nowStr := time.Now().Format("2006-01-02")
	claim := &domain.Claim{
		ID:            fmt.Sprintf("claim_%d", time.Now().UnixNano()),
		PatientID:     patientID,
		ProviderID:    providerID,
		DateOfService: nowStr,
		Status:        domain.ClaimStatusDraft,
		Notes:         "Generated from dental chart conditions",
		LineItems:     []domain.ClaimLineItem{},
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	for i, condID := range conditionIDs {
		cond, exists := condMap[condID]
		if !exists {
			continue
		}

		adaCode := cond.ADACode
		if adaCode == "" {
			adaCode = "PROC"
		}

		lineItem := domain.ClaimLineItem{
			ID:               fmt.Sprintf("li_%d_%d", time.Now().UnixNano(), i),
			ToothConditionID: cond.ID,
			ToothNumber:      cond.ToothNumber,
			Surfaces:         cond.Surfaces,
			ADACode:          adaCode,
			Description:      cond.Description,
			Fee:              cond.Fee,
		}
		claim.LineItems = append(claim.LineItems, lineItem)

		// Mark tooth condition status as completed if it was treatment planned
		if cond.Status == domain.ToothStatusTreatmentPlanned {
			cond.Status = domain.ToothStatusCompleted
			_ = s.chartRepo.SaveCondition(ctx, &cond)
		}
	}

	if len(claim.LineItems) == 0 {
		return nil, fmt.Errorf("%w: no matching conditions found to create claim", storage.ErrInvalidInput)
	}

	if err := s.claimRepo.Create(ctx, claim); err != nil {
		return nil, fmt.Errorf("failed to create claim from chart: %w", err)
	}
	_ = s.auditService.LogPatientAction(token, domain.AuditActionCreate, claim.PatientID, "claim", "Created claim from chart")
	return claim, nil
}
