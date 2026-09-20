package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestClaimRepository(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_billing_repo.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	// Seed patient first (due to foreign keys if any or integrity)
	patientRepo := sqlite.NewPatientRepository(db)
	ctx := context.Background()
	patient := &domain.Patient{
		ID:        "pat_claim_1",
		FirstName: "Jane",
		LastName:  "Doe",
	}
	if err := patientRepo.Create(ctx, patient); err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	configRepo := sqlite.NewPracticeConfigRepository(db)
	if err := configRepo.SaveProvider(ctx, &domain.Provider{ID: "prov_1", Name: "Dr Test", IsActive: true}); err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	repo := sqlite.NewClaimRepository(db)

	// 1. Test Input Validation for Create
	invalidClaim := &domain.Claim{ID: "", PatientID: "pat_claim_1", DateOfService: "2026-08-15"}
	if err := repo.Create(ctx, invalidClaim); err == nil {
		t.Errorf("Expected error when creating claim with empty ID")
	}

	// 2. Create Claim with line items
	claim := &domain.Claim{
		ID:               "clm_1001",
		PatientID:        "pat_claim_1",
		ProviderID:       "prov_1",
		InsuranceCarrier: "Delta Dental",
		PolicyNumber:     "POL-99",
		DateOfService:    "2026-08-15",
		Status:           domain.ClaimStatusDraft,
		Notes:            "Routine clean and exam",
		LineItems: []domain.ClaimLineItem{
			{
				ID:          "item_1",
				ADACode:     "D0120",
				Description: "Periodic Oral Evaluation",
				Fee:         5000,
			},
			{
				ID:          "item_2",
				ADACode:     "D1110",
				Description: "Prophylaxis - Adult",
				Fee:         12000,
			},
		},
	}

	if err := repo.Create(ctx, claim); err != nil {
		t.Fatalf("Failed to create valid claim: %v", err)
	}

	// 3. GetByID
	fetched, err := repo.GetByID(ctx, "clm_1001")
	if err != nil {
		t.Fatalf("Failed to get claim by ID: %v", err)
	}
	if fetched.PatientID != "pat_claim_1" || fetched.InsuranceCarrier != "Delta Dental" {
		t.Errorf("Unexpected claim data: %+v", fetched)
	}
	if len(fetched.LineItems) != 2 {
		t.Fatalf("Expected 2 line items, got %d", len(fetched.LineItems))
	}
	if fetched.LineItems[0].ADACode != "D0120" || fetched.LineItems[0].Fee != 5000 {
		t.Errorf("Unexpected first line item: %+v", fetched.LineItems[0])
	}

	// GetByID with invalid/nonexistent ID
	if _, err := repo.GetByID(ctx, ""); err == nil {
		t.Errorf("Expected error for empty ID in GetByID")
	}
	if _, err := repo.GetByID(ctx, "non_existent_clm"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound for non-existent claim, got: %v", err)
	}

	// 4. Update Claim
	fetched.Status = domain.ClaimStatusSubmitted
	fetched.Notes = "Submitted to Delta Dental"
	fetched.LineItems = append(fetched.LineItems, domain.ClaimLineItem{
		ID:          "item_3",
		ADACode:     "D0210",
		Description: "Intraoral - Comprehensive Series",
		Fee:         15000,
	})

	if err := repo.Update(ctx, fetched); err != nil {
		t.Fatalf("Failed to update claim: %v", err)
	}

	updated, err := repo.GetByID(ctx, "clm_1001")
	if err != nil {
		t.Fatalf("Failed to get updated claim: %v", err)
	}
	if updated.Status != domain.ClaimStatusSubmitted {
		t.Errorf("Expected status 'submitted', got '%s'", updated.Status)
	}
	if len(updated.LineItems) != 3 {
		t.Errorf("Expected 3 line items after update, got %d", len(updated.LineItems))
	}

	// Update non-existent claim
	nonExistentClaim := &domain.Claim{ID: "no_such_claim"}
	if err := repo.Update(ctx, nonExistentClaim); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound updating nonexistent claim, got: %v", err)
	}

	// 5. GetTotalBilled
	totalBilled, err := repo.GetTotalBilled(ctx, "pat_claim_1")
	if err != nil {
		t.Fatalf("Failed to get total billed: %v", err)
	}
	// 5000 + 12000 + 15000 = 32000
	if totalBilled != 32000 {
		t.Errorf("Expected total billed 32000, got %d", totalBilled)
	}

	// 5b. GetTotalBilledByPatient across multiple patients
	otherPatient := &domain.Patient{ID: "pat_claim_2", FirstName: "John", LastName: "Roe"}
	if err := patientRepo.Create(ctx, otherPatient); err != nil {
		t.Fatalf("Failed to create second patient: %v", err)
	}
	otherClaim := &domain.Claim{
		ID:            "clm_1002",
		PatientID:     "pat_claim_2",
		DateOfService: "2026-08-16",
		Status:        domain.ClaimStatusDraft,
		LineItems: []domain.ClaimLineItem{
			{ID: "item_4", ADACode: "D0120", Fee: 6000},
		},
	}
	if err := repo.Create(ctx, otherClaim); err != nil {
		t.Fatalf("Failed to create second patient's claim: %v", err)
	}

	billedByPatient, err := repo.GetTotalBilledByPatient(ctx)
	if err != nil {
		t.Fatalf("Failed to get total billed by patient: %v", err)
	}
	if billedByPatient["pat_claim_1"] != 32000 {
		t.Errorf("Expected pat_claim_1 total billed 32000, got %d", billedByPatient["pat_claim_1"])
	}
	if billedByPatient["pat_claim_2"] != 6000 {
		t.Errorf("Expected pat_claim_2 total billed 6000, got %d", billedByPatient["pat_claim_2"])
	}

	// 5c. A patient with a claim but no line items must still get a zero entry in the map.
	thirdPatient := &domain.Patient{ID: "pat_claim_3", FirstName: "Amy", LastName: "Nolan"}
	if err := patientRepo.Create(ctx, thirdPatient); err != nil {
		t.Fatalf("Failed to create third patient: %v", err)
	}
	if err := repo.Create(ctx, &domain.Claim{
		ID:            "clm_1003",
		PatientID:     "pat_claim_3",
		DateOfService: "2026-08-17",
		Status:        domain.ClaimStatusDraft,
		LineItems:     []domain.ClaimLineItem{},
	}); err != nil {
		t.Fatalf("Failed to create third patient's claim: %v", err)
	}
	billedByPatient2, err := repo.GetTotalBilledByPatient(ctx)
	if err != nil {
		t.Fatalf("Failed to get total billed by patient with empty line items present: %v", err)
	}
	if total, ok := billedByPatient2["pat_claim_3"]; !ok || total != 0 {
		t.Errorf("Expected pat_claim_3 to have a zero entry, got ok=%v total=%d", ok, total)
	}
	if total, err := repo.GetTotalBilled(ctx, "pat_claim_3"); err != nil || total != 0 {
		t.Errorf("Expected GetTotalBilled 0 for empty line items, got total=%d err=%v", total, err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM claims WHERE id = 'clm_1003'`); err != nil {
		t.Fatalf("Failed to clean up empty line_items claim: %v", err)
	}

	// 5d. The schema must reject NULL or malformed line_items rather than storing them.
	for name, lineItems := range map[string]any{"NULL": nil, "malformed JSON": "not-json"} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO claims (id, patient_id, date_of_service, status, line_items, created_at, updated_at)
			 VALUES ('clm_bad', 'pat_claim_3', '2026-08-19', 'draft', ?, ?, ?)`,
			lineItems, time.Now().UTC(), time.Now().UTC()); err == nil {
			t.Errorf("Expected schema to reject %s line_items", name)
		}
	}

	// 5e. A claim must not reference a nonexistent provider, and an invalid date is rejected.
	if err := repo.Create(ctx, &domain.Claim{ID: "clm_bad_prov", PatientID: "pat_claim_3", ProviderID: "no_such_provider", DateOfService: "2026-08-19"}); err == nil {
		t.Errorf("Expected foreign key error for nonexistent provider")
	}
	if err := repo.Create(ctx, &domain.Claim{ID: "clm_bad_date", PatientID: "pat_claim_3", DateOfService: "not-a-date"}); err == nil {
		t.Errorf("Expected error for invalid date_of_service")
	}

	// 6. List Claims
	claimsForPatient, err := repo.List(ctx, "pat_claim_1")
	if err != nil {
		t.Fatalf("Failed to list claims by patient: %v", err)
	}
	if len(claimsForPatient) != 1 {
		t.Errorf("Expected 1 claim for patient, got %d", len(claimsForPatient))
	}

	allClaims, err := repo.List(ctx, "")
	if err != nil {
		t.Fatalf("Failed to list all claims: %v", err)
	}
	if len(allClaims) != 2 {
		t.Errorf("Expected 2 claims in total list, got %d", len(allClaims))
	}

	// 7. Delete Claim
	if err := repo.Delete(ctx, "clm_1001"); err != nil {
		t.Fatalf("Failed to delete claim: %v", err)
	}

	if _, err := repo.GetByID(ctx, "clm_1001"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got: %v", err)
	}

	if err := repo.Delete(ctx, "clm_1001"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound deleting already deleted claim, got: %v", err)
	}
}
