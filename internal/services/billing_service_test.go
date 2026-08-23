package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

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
	paymentRepo := sqlite.NewPaymentRepository(db)
	bundleRepo := sqlite.NewBundleRepository(db)
	procRepo := sqlite.NewProcedureRepository(db)

	secretsSvc := NewSecretsService()
	billingSvc := NewBillingService(claimRepo, paymentRepo, bundleRepo, procRepo, procRepo, chartRepo, secretsSvc)

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
	if err := chartRepo.SaveCondition(ctx, cond); err != nil {
		t.Fatalf("Failed to save tooth condition: %v", err)
	}

	// 3. Test CreateClaimFromChartConditions
	claim, err := billingSvc.CreateClaimFromChartConditions("pat_test_1", "prov_1", []string{"cond_test_1"})
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

	secretsSvc := NewSecretsService()
	billingSvc := NewBillingService(
		claimRepo,
		sqlite.NewPaymentRepository(db),
		sqlite.NewBundleRepository(db),
		sqlite.NewProcedureRepository(db),
		sqlite.NewProcedureRepository(db),
		sqlite.NewChartRepository(db),
		secretsSvc,
	)

	// Register the test provider
	testProv := &dummyTestProvider{}
	billingSvc.RegisterProvider(testProv)

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
	result, err := billingSvc.SubmitClaimToProvider("claim_test_1", "test_mock")
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
	_, err = billingSvc.SubmitClaimToProvider("claim_test_1", "test_mock")
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
	_, err = billingSvc.SubmitClaimToProvider("claim_test_2", "test_mock")
	if err == nil || err.Error() != `provider "test_mock" returned nil result` {
		t.Fatalf("Expected nil result error, got %v", err)
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
