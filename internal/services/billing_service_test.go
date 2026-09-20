package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func init() {
	keyring.MockInit()
}

func TestBillingService_ProcedureCodesAndChartClaim(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_billing_service.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	patientRepo := sqlite.NewPatientRepository(db)
	chartRepo := sqlite.NewChartRepository(db)
	claimRepo := sqlite.NewClaimRepository(db)

	auditRepo := sqlite.NewAuditRepository(db)
	configRepo := sqlite.NewPracticeConfigRepository(db)
	err = configRepo.SaveProvider(ctx, &domain.Provider{ID: "prov_1", Name: "Test Prov", Pin: "1234", IsActive: true})
	if err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	auditSvc := NewAuditService(auditRepo, configRepo)
	token, err := auditSvc.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	paymentRepo := sqlite.NewPaymentRepository(db)
	bundleRepo := sqlite.NewBundleRepository(db)
	procRepo := sqlite.NewProcedureRepository(db)

	secretsSvc := NewSecretsService()
	billingSvc := NewBillingService(claimRepo, paymentRepo, bundleRepo, procRepo, procRepo, chartRepo, secretsSvc, auditSvc)

	// Create test patient
	patient := &domain.Patient{
		ID:          "pat_test_1",
		FirstName:   "Jane",
		LastName:    "Doe",
		DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Sex:         domain.SexFemale,
		Status:      domain.StatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := patientRepo.Create(ctx, patient); err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	// 1. Test ListProcedureCodes for US
	codes, err := billingSvc.ListProcedureCodes("US", "")
	if err != nil {
		t.Fatalf("Failed to list procedure codes: %v", err)
	}
	if len(codes) == 0 {
		t.Fatalf("Expected seeded procedure codes for US")
	}

	// 2. Add tooth condition to chart for patient
	cond := &domain.ToothCondition{
		ID:          "cond_test_1",
		PatientID:   "pat_test_1",
		ToothNumber: 14,
		Surfaces:    []domain.ToothSurface{domain.SurfaceMesial, domain.SurfaceOcclusal},
		ADACode:     "D2392",
		Description: "2-Surface Composite Resin",
		Status:      domain.ToothStatusTreatmentPlanned,
		Fee:         18500,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if _, err := chartRepo.SaveCondition(ctx, cond); err != nil {
		t.Fatalf("Failed to save tooth condition: %v", err)
	}

	// 3. Test CreateClaimFromChartConditions
	claim, err := billingSvc.CreateClaimFromChartConditions(token, "pat_test_1", "prov_1", []string{"cond_test_1"})
	if err != nil {
		t.Fatalf("Failed to create claim from chart condition: %v", err)
	}

	if claim == nil || len(claim.LineItems) != 1 {
		t.Fatalf("Expected 1 line item in generated claim")
	}
	if claim.LineItems[0].ADACode != "D2392" || claim.LineItems[0].Fee != 18500 {
		t.Errorf("Unexpected line item values: %+v", claim.LineItems[0])
	}

	// Verify chart condition status was updated to completed
	chart, err := chartRepo.GetChart(ctx, "pat_test_1")
	if err != nil {
		t.Fatalf("Failed to get updated chart: %v", err)
	}
	if len(chart.Conditions) == 0 || chart.Conditions[0].Status != domain.ToothStatusCompleted {
		t.Errorf("Expected condition status to be completed, got %s", chart.Conditions[0].Status)
	}
}

func TestBillingService_BundlesAndFeeSchedulesRequireAuth(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_billing_service_bundles.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	chartRepo := sqlite.NewChartRepository(db)
	claimRepo := sqlite.NewClaimRepository(db)
	paymentRepo := sqlite.NewPaymentRepository(db)
	bundleRepo := sqlite.NewBundleRepository(db)
	procRepo := sqlite.NewProcedureRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	configRepo := sqlite.NewPracticeConfigRepository(db)

	if err := configRepo.SaveProvider(ctx, &domain.Provider{ID: "prov_1", Name: "Test Prov", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	auditSvc := NewAuditService(auditRepo, configRepo)
	token, err := auditSvc.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	secretsSvc := NewSecretsService()
	billingSvc := NewBillingService(claimRepo, paymentRepo, bundleRepo, procRepo, procRepo, chartRepo, secretsSvc, auditSvc)

	bundle := &domain.TreatmentBundle{
		Shortname: "crwn",
		Name:      "Crown",
		Items: []domain.BundleItemTemplate{
			{ADACode: "D2740", Description: "Crown - porcelain", DefaultFee: 120000},
		},
	}

	// Unauthenticated calls must be rejected, not silently allowed.
	if _, err := billingSvc.CreateBundle("bogus-token", bundle); err != ErrUnauthorized {
		t.Fatalf("Expected ErrUnauthorized for CreateBundle without a session, got %v", err)
	}
	if _, err := billingSvc.UpdateBundle("bogus-token", bundle); err != ErrUnauthorized {
		t.Fatalf("Expected ErrUnauthorized for UpdateBundle without a session, got %v", err)
	}
	if err := billingSvc.DeleteBundle("bogus-token", "bundle_1"); err != ErrUnauthorized {
		t.Fatalf("Expected ErrUnauthorized for DeleteBundle without a session, got %v", err)
	}
	fee := &domain.FeeSchedule{Code: "D2740", CustomFee: 130000}
	if _, err := billingSvc.SaveFeeSchedule("bogus-token", fee); err != ErrUnauthorized {
		t.Fatalf("Expected ErrUnauthorized for SaveFeeSchedule without a session, got %v", err)
	}
	if err := billingSvc.DeleteFeeSchedule("bogus-token", "fee_1"); err != ErrUnauthorized {
		t.Fatalf("Expected ErrUnauthorized for DeleteFeeSchedule without a session, got %v", err)
	}

	// Authenticated calls should succeed and be reflected in storage.
	created, err := billingSvc.CreateBundle(token, bundle)
	if err != nil {
		t.Fatalf("Failed to create bundle with valid session: %v", err)
	}
	if created.ID == "" {
		t.Errorf("Expected generated bundle ID")
	}

	created.Name = "Crown (Updated)"
	if _, err := billingSvc.UpdateBundle(token, created); err != nil {
		t.Fatalf("Failed to update bundle with valid session: %v", err)
	}

	if err := billingSvc.DeleteBundle(token, created.ID); err != nil {
		t.Fatalf("Failed to delete bundle with valid session: %v", err)
	}

	savedFee, err := billingSvc.SaveFeeSchedule(token, fee)
	if err != nil {
		t.Fatalf("Failed to save fee schedule with valid session: %v", err)
	}
	if savedFee.ID == "" {
		t.Errorf("Expected generated fee schedule ID")
	}

	if err := billingSvc.DeleteFeeSchedule(token, savedFee.ID); err != nil {
		t.Fatalf("Failed to delete fee schedule with valid session: %v", err)
	}
}

func TestBillingService_SubmitClaimToProvider(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_billing_service_submit.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	claimRepo := sqlite.NewClaimRepository(db)

	auditRepo := sqlite.NewAuditRepository(db)
	configRepo := sqlite.NewPracticeConfigRepository(db)
	err = configRepo.SaveProvider(ctx, &domain.Provider{ID: "prov_1", Name: "Test Prov", Pin: "1234", IsActive: true})
	if err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	auditSvc := NewAuditService(auditRepo, configRepo)
	token, err := auditSvc.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	secretsSvc := NewSecretsService()
	billingSvc := NewBillingService(
		claimRepo,
		sqlite.NewPaymentRepository(db),
		sqlite.NewBundleRepository(db),
		sqlite.NewProcedureRepository(db),
		sqlite.NewProcedureRepository(db),
		sqlite.NewChartRepository(db),
		secretsSvc,
		auditSvc,
	)

	// Register the test provider
	testProv := &dummyTestProvider{}
	billingSvc.registerProvider(testProv)

	patientRepo := sqlite.NewPatientRepository(db)
	patient := &domain.Patient{
		ID:          "pat_test_1",
		FirstName:   "Jane",
		LastName:    "Doe",
		DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Sex:         domain.SexFemale,
		Status:      domain.StatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := patientRepo.Create(ctx, patient); err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	// Create a claim
	claim := &domain.Claim{
		ID:            "claim_test_1",
		PatientID:     "pat_test_1",
		Status:        domain.ClaimStatusDraft,
		DateOfService: "2026-08-23",
	}
	if err := claimRepo.Create(ctx, claim); err != nil {
		t.Fatalf("Failed to create claim: %v", err)
	}

	// Test 1: Successful submission
	result, err := billingSvc.SubmitClaimToProvider(token, "claim_test_1", "test_mock")
	if err != nil {
		t.Fatalf("SubmitClaimToProvider failed: %v", err)
	}
	if result.Status != domain.ClaimStatusSubmitted {
		t.Errorf("Expected status %v, got %v", domain.ClaimStatusSubmitted, result.Status)
	}

	// Verify the claim status in the database was updated
	updatedClaim, err := claimRepo.GetByID(ctx, "claim_test_1")
	if err != nil {
		t.Fatalf("Failed to get updated claim: %v", err)
	}
	if updatedClaim.Status != domain.ClaimStatusSubmitted {
		t.Errorf("Expected claim status in DB to be submitted, got %v", updatedClaim.Status)
	}

	// Test 2: Idempotency check - should fail because it's already submitted
	_, err = billingSvc.SubmitClaimToProvider(token, "claim_test_1", "test_mock")
	if err == nil {
		t.Fatalf("Expected error when submitting an already submitted claim")
	}

	// Test 3: Nil check handling
	claim2 := &domain.Claim{
		ID:            "claim_test_2",
		PatientID:     "pat_test_1",
		Status:        domain.ClaimStatusDraft,
		DateOfService: "2026-08-23",
	}
	if err := claimRepo.Create(ctx, claim2); err != nil {
		t.Fatalf("Failed to create claim2: %v", err)
	}
	testProv.submitFunc = func() (*domain.ClaimSubmissionResult, error) {
		return nil, nil // Return nil intentionally
	}
	_, err = billingSvc.SubmitClaimToProvider(token, "claim_test_2", "test_mock")
	if err == nil || err.Error() != `provider "test_mock" returned nil result` {
		t.Fatalf("Expected nil result error, got %v", err)
	}
}

func TestBillingService_GetAllPatientBalances(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_billing_service_balances.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	claimRepo := sqlite.NewClaimRepository(db)
	paymentRepo := sqlite.NewPaymentRepository(db)
	chartRepo := sqlite.NewChartRepository(db)
	bundleRepo := sqlite.NewBundleRepository(db)
	procRepo := sqlite.NewProcedureRepository(db)
	patientRepo := sqlite.NewPatientRepository(db)

	auditRepo := sqlite.NewAuditRepository(db)
	configRepo := sqlite.NewPracticeConfigRepository(db)
	if err := configRepo.SaveProvider(ctx, &domain.Provider{ID: "prov_1", Name: "Test Prov", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	auditSvc := NewAuditService(auditRepo, configRepo)
	token, err := auditSvc.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	secretsSvc := NewSecretsService()
	billingSvc := NewBillingService(claimRepo, paymentRepo, bundleRepo, procRepo, procRepo, chartRepo, secretsSvc, auditSvc)

	if _, err := billingSvc.GetAllPatientBalances("bogus-token"); err != ErrUnauthorized {
		t.Fatalf("Expected ErrUnauthorized without a session, got %v", err)
	}

	patientA := &domain.Patient{ID: "pat_bal_a", FirstName: "Alice", LastName: "A"}
	patientB := &domain.Patient{ID: "pat_bal_b", FirstName: "Bob", LastName: "B"}
	if err := patientRepo.Create(ctx, patientA); err != nil {
		t.Fatalf("Failed to create patient A: %v", err)
	}
	if err := patientRepo.Create(ctx, patientB); err != nil {
		t.Fatalf("Failed to create patient B: %v", err)
	}

	// Patient A: billed 10000, paid 4000 -> outstanding 6000
	if err := claimRepo.Create(ctx, &domain.Claim{
		ID:            "claim_bal_a",
		PatientID:     "pat_bal_a",
		DateOfService: "2026-08-15",
		Status:        domain.ClaimStatusDraft,
		LineItems:     []domain.ClaimLineItem{{ID: "li_a", ADACode: "D0120", Fee: 10000}},
	}); err != nil {
		t.Fatalf("Failed to create claim for patient A: %v", err)
	}
	if _, err := billingSvc.RecordPayment(token, &domain.Payment{
		ID: "pay_bal_a", PatientID: "pat_bal_a", Amount: 4000, Method: domain.PaymentMethodCash, Date: "2026-08-16",
	}); err != nil {
		t.Fatalf("Failed to record payment for patient A: %v", err)
	}

	// Patient B: billed 2000, paid 2000 -> outstanding 0
	if err := claimRepo.Create(ctx, &domain.Claim{
		ID:            "claim_bal_b",
		PatientID:     "pat_bal_b",
		DateOfService: "2026-08-15",
		Status:        domain.ClaimStatusDraft,
		LineItems:     []domain.ClaimLineItem{{ID: "li_b", ADACode: "D0120", Fee: 2000}},
	}); err != nil {
		t.Fatalf("Failed to create claim for patient B: %v", err)
	}
	if _, err := billingSvc.RecordPayment(token, &domain.Payment{
		ID: "pay_bal_b", PatientID: "pat_bal_b", Amount: 2000, Method: domain.PaymentMethodCash, Date: "2026-08-16",
	}); err != nil {
		t.Fatalf("Failed to record payment for patient B: %v", err)
	}

	// Patient C: a payment recorded with no linked claim (e.g. a prepayment or credit),
	// so billed=0, paid=3000 -> outstanding must clamp to 0, not go negative.
	patientC := &domain.Patient{ID: "pat_bal_c", FirstName: "Cass", LastName: "C"}
	if err := patientRepo.Create(ctx, patientC); err != nil {
		t.Fatalf("Failed to create patient C: %v", err)
	}
	if _, err := billingSvc.RecordPayment(token, &domain.Payment{
		ID: "pay_bal_c", PatientID: "pat_bal_c", Amount: 3000, Method: domain.PaymentMethodCash, Date: "2026-08-16",
	}); err != nil {
		t.Fatalf("Failed to record payment for patient C: %v", err)
	}

	balances, err := billingSvc.GetAllPatientBalances(token)
	if err != nil {
		t.Fatalf("Failed to get all patient balances: %v", err)
	}
	if len(balances) != 3 {
		t.Fatalf("Expected 3 patient balances, got %d", len(balances))
	}
	// Sorted by outstanding descending, so patient A (6000) comes first.
	if balances[0].PatientID != "pat_bal_a" || balances[0].Outstanding != 6000 {
		t.Errorf("Expected patient A first with outstanding 6000, got %+v", balances[0])
	}

	var patientCBalance *domain.PatientBalance
	var outstandingSum int64
	for _, b := range balances {
		outstandingSum += b.Outstanding
		if b.PatientID == "pat_bal_c" {
			patientCBalance = b
		}
	}
	if patientCBalance == nil {
		t.Fatalf("Expected patient C to be present in balances")
	}
	if patientCBalance.Outstanding != 0 {
		t.Errorf("Expected patient C's overpayment to clamp outstanding to 0, got %d", patientCBalance.Outstanding)
	}
	if patientCBalance.TotalPaid != 3000 {
		t.Errorf("Expected patient C total paid 3000, got %d", patientCBalance.TotalPaid)
	}
	// The aggregate total must equal the sum of the (clamped) per-patient rows, so the
	// header and the "outstanding by patient" list the frontend derives from it agree.
	if outstandingSum != 6000 {
		t.Errorf("Expected sum of outstanding balances to be 6000, got %d", outstandingSum)
	}
}

type dummyTestProvider struct {
	submitFunc func() (*domain.ClaimSubmissionResult, error)
}

func (p *dummyTestProvider) Name() string                             { return "test_mock" }
func (p *dummyTestProvider) SupportedCountries() []domain.CountryCode { return nil }
func (p *dummyTestProvider) SubmitClaim(ctx context.Context, claim *domain.Claim, config map[string]string) (*domain.ClaimSubmissionResult, error) {
	if p.submitFunc != nil {
		return p.submitFunc()
	}
	return &domain.ClaimSubmissionResult{Status: domain.ClaimStatusSubmitted}, nil
}
