package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/pkg/domain"
)

func TestProcedureRepository(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_procedure.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	repo := NewProcedureRepository(db)
	ctx := context.Background()

	// 1. Test listing seeded US codes
	codes, err := repo.List(ctx, domain.CountryUS)
	if err != nil {
		t.Fatalf("Failed to list US procedure codes: %v", err)
	}
	if len(codes) == 0 {
		t.Fatalf("Expected seeded US procedure codes, got 0")
	}

	// Find D2391
	var foundD2391 *domain.ProcedureCode
	for _, c := range codes {
		if c.Code == "D2391" {
			foundD2391 = c
			break
		}
	}
	if foundD2391 == nil {
		t.Fatalf("Expected D2391 in US codes")
	}
	if foundD2391.DefaultFee != 140.0 {
		t.Errorf("Expected fee 140.0, got %f", foundD2391.DefaultFee)
	}

	// 2. Test initial effective fee (should equal default fee)
	effFee, err := repo.GetEffectiveFee(ctx, domain.CountryUS, "D2391", "")
	if err != nil {
		t.Fatalf("Failed to get effective fee: %v", err)
	}
	if effFee != 140.0 {
		t.Errorf("Expected effective fee 140.0, got %f", effFee)
	}

	// 3. Save a practice-wide custom fee schedule override
	fs := &domain.FeeSchedule{
		ID:          "fee_test_1",
		CountryCode: domain.CountryUS,
		Code:        "D2391",
		ProviderID:  "",
		CustomFee:   175.50,
	}
	if err := repo.Save(ctx, fs); err != nil {
		t.Fatalf("Failed to save fee schedule: %v", err)
	}

	// Verify effective fee now returns custom fee 175.50
	effFee, err = repo.GetEffectiveFee(ctx, domain.CountryUS, "D2391", "")
	if err != nil {
		t.Fatalf("Failed to get updated effective fee: %v", err)
	}
	if effFee != 175.50 {
		t.Errorf("Expected effective fee 175.50, got %f", effFee)
	}

	// 4. Save a provider-specific custom fee override
	fsProv := &domain.FeeSchedule{
		ID:          "fee_test_2",
		CountryCode: domain.CountryUS,
		Code:        "D2391",
		ProviderID:  "prov_dr_smith",
		CustomFee:   200.00,
	}
	if err := repo.Save(ctx, fsProv); err != nil {
		t.Fatalf("Failed to save provider fee schedule: %v", err)
	}

	effFeeProv, err := repo.GetEffectiveFee(ctx, domain.CountryUS, "D2391", "prov_dr_smith")
	if err != nil {
		t.Fatalf("Failed to get provider effective fee: %v", err)
	}
	if effFeeProv != 200.00 {
		t.Errorf("Expected provider effective fee 200.00, got %f", effFeeProv)
	}
}
