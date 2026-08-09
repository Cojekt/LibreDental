package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage/sqlite"
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

	billingSvc := NewBillingService(claimRepo, paymentRepo, bundleRepo, procRepo, procRepo, chartRepo)

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
		Fee:         185.00,
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
	if claim.LineItems[0].ADACode != "D2392" || claim.LineItems[0].Fee != 185.00 {
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
