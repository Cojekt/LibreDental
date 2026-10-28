package services

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestInsuranceServiceEndToEnd(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auditDB, err := sqlite.OpenAudit(filepath.Join(dir, "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer auditDB.Close()
	config := sqlite.NewPracticeConfigRepository(db)
	if err := config.SaveProvider(ctx, &domain.Provider{ID: "dentist", Name: "Dentist", Pin: "1234", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	audit := NewAuditService(sqlite.NewAuditRepository(auditDB), config)
	token, err := audit.CreateSession("dentist", "1234")
	if err != nil {
		t.Fatal(err)
	}
	repo := sqlite.NewInsuranceRepository(db)
	svc := NewInsuranceService(repo, audit)
	for _, invalidToken := range []string{"", "invalid"} {
		if _, err := svc.ListProviders(invalidToken); !errors.Is(err, ErrUnauthorized) {
			t.Fatal(err)
		}
		if _, err := svc.ListFees(invalidToken, "", "US"); !errors.Is(err, ErrUnauthorized) {
			t.Fatal(err)
		}
		if _, err := svc.Preview(invalidToken, nil); !errors.Is(err, ErrUnauthorized) {
			t.Fatal(err)
		}
		if _, err := svc.SaveProvider(invalidToken, &domain.InsuranceProvider{Name: "Blocked"}); !errors.Is(err, ErrUnauthorized) {
			t.Fatal(err)
		}
		if err := svc.SaveFee(invalidToken, nil); !errors.Is(err, ErrUnauthorized) {
			t.Fatal(err)
		}
		if err := svc.DeleteFee(invalidToken, "", "US", ""); !errors.Is(err, ErrUnauthorized) {
			t.Fatal(err)
		}
	}
	insurer, err := svc.SaveProvider(token, &domain.InsuranceProvider{Name: "User Entered Company"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SaveFee(token, &domain.InsuranceFee{ProviderID: insurer.ID, CountryCode: domain.CountryUS, Code: "D2391", Fee: 10000}); err != nil {
		t.Fatal(err)
	}
	patientRepo := sqlite.NewPatientRepository(db)
	for _, id := range []string{"a", "b"} {
		if err := patientRepo.Create(ctx, &domain.Patient{ID: id, FirstName: id, LastName: "Test", InsuranceCarrier: "Original patient text"}); err != nil {
			t.Fatal(err)
		}
	}
	chartRepo := sqlite.NewChartRepository(db)
	chart := NewChartService(chartRepo, audit)
	proc := sqlite.NewProcedureRepository(db)
	claims := sqlite.NewClaimRepository(db)
	billing := NewBillingService(claims, sqlite.NewPaymentRepository(db), sqlite.NewBundleRepository(db), proc, proc, chartRepo, patientRepo, config, NewSecretsService(), audit)
	AttachInsurancePricing(chart, billing, repo)
	preview, err := svc.Preview(token, &domain.PricingSource{Kind: "insurance", PatientID: "a", CountryCode: domain.CountryUS, Code: "D2391", ProviderID: insurer.ID})
	if err != nil {
		t.Fatal(err)
	}
	payload := &domain.SaveToothConditionPayload{ID: "condition-a", PatientID: "a", ToothNumber: 3, ADACode: "D2391", Description: "Filling", Status: domain.ToothStatusTreatmentPlanned, Fee: preview.Amount, Pricing: preview}
	if _, err := chart.SaveToothCondition("invalid", payload); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
	saved, err := chart.SaveToothCondition(token, payload)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := billing.CreateClaimFromChartConditions(token, "a", "dentist", []string{saved.ID})
	if err != nil {
		t.Fatal(err)
	}
	if claim.LineItems[0].Fee != 10000 || claim.LineItems[0].Pricing.ProviderID != insurer.ID {
		t.Fatal("snapshot not copied")
	}
	if _, err := billing.CreateClaimFromChartConditions(token, "a", "dentist", []string{saved.ID}); !errors.Is(err, storage.ErrInvalidInput) {
		t.Fatal("duplicate chart charge", err)
	}
	if err := svc.SaveFee(token, &domain.InsuranceFee{ProviderID: insurer.ID, CountryCode: domain.CountryUS, Code: "D2391", Fee: 14000}); err != nil {
		t.Fatal(err)
	}
	stale := *payload
	stale.ID = "stale"
	if _, err := chart.SaveToothCondition(token, &stale); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale accepted", err)
	}
	current, err := svc.Preview(token, &domain.PricingSource{Kind: "insurance", PatientID: "b", CountryCode: domain.CountryUS, Code: "D2391", ProviderID: insurer.ID, ClinicianID: "dentist"})
	if err != nil || current.Amount != 14000 {
		t.Fatal(current, err)
	}
	direct := &domain.Claim{ID: "direct-b", PatientID: "b", ProviderID: "dentist", DateOfService: time.Now().Format("2006-01-02"), LineItems: []domain.ClaimLineItem{{ID: "line-b", ADACode: "D2391", Description: "Filling", Fee: current.Amount, Pricing: current}}}
	if _, err := billing.CreateClaim("", direct); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("unauthenticated claim accepted", err)
	}
	if _, err := billing.CreateClaim(token, direct); err != nil {
		t.Fatal(err)
	}
	old, err := billing.GetClaim(token, claim.ID)
	if err != nil || old.LineItems[0].Fee != 10000 {
		t.Fatal("historical claim changed", err)
	}
	if old.LineItems[0].Pricing.ProviderName != "User Entered Company" {
		t.Fatal("historical name lost")
	}
	patient, err := patientRepo.GetByID(ctx, "a")
	if err != nil || patient.InsuranceCarrier != "Original patient text" {
		t.Fatal("patient insurance text changed")
	}
	// Existing audited claim edits preserve untouched source metadata.
	direct.Notes = "Audited correction note"
	if _, err := billing.UpdateClaim(token, direct); err != nil {
		t.Fatal(err)
	}
	updated, err := billing.GetClaim(token, direct.ID)
	if err != nil || updated.LineItems[0].Fee != 14000 || updated.LineItems[0].Pricing == nil || updated.LineItems[0].Pricing.ProviderName != "User Entered Company" {
		t.Fatal("notes edit changed direct pricing snapshot", updated, err)
	}
	if _, err := svc.SaveProvider(token, &domain.InsuranceProvider{Name: "User Entered Company"}); err == nil {
		t.Fatal("duplicate accepted")
	}
	if err := svc.SaveFee(token, &domain.InsuranceFee{ProviderID: insurer.ID, CountryCode: domain.CountryUS, Code: "D2391", Fee: -1}); err == nil {
		t.Fatal("invalid fee accepted")
	}
	if err := svc.DeleteFee(token, "", "US", "D2391"); err == nil {
		t.Fatal("invalid removal accepted")
	}
	if _, err := chart.SaveToothCondition(token, &domain.SaveToothConditionPayload{ID: "nonbillable-review", PatientID: "a", ToothNumber: 1, Status: domain.ToothStatusExisting}); err != nil {
		t.Fatal(err)
	}
	logs, err := audit.GetAuditLogs(token, "", 1000, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range []string{"insurance_provider", "insurance_fee"} {
		found := false
		for _, l := range logs {
			if l.Resource == resource && strings.Contains(l.Details, "failed:") {
				found = true
			}
		}
		if !found {
			t.Errorf("missing failure audit for %s", resource)
		}
	}
	for _, l := range logs {
		if l.Resource == "pricing" && strings.Contains(l.Details, "nonbillable-review") {
			t.Fatal("nonbillable finding logged as pricing request")
		}
	}
	for _, resource := range []string{"insurance_provider", "insurance_fee", "pricing", "claim", "dental_chart"} {
		found := false
		for _, l := range logs {
			if l.Resource == resource {
				found = true
			}
		}
		if !found {
			t.Errorf("missing audit resource %s", resource)
		}
	}
}

func TestChartClaimProviderFollowsPricingClinician(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auditDB, err := sqlite.OpenAudit(filepath.Join(dir, "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer auditDB.Close()
	config := sqlite.NewPracticeConfigRepository(db)
	for _, id := range []string{"dentist-a", "dentist-b"} {
		if err := config.SaveProvider(ctx, &domain.Provider{ID: id, Name: id, Pin: "1234", IsActive: true}); err != nil {
			t.Fatal(err)
		}
	}
	audit := NewAuditService(sqlite.NewAuditRepository(auditDB), config)
	token, err := audit.CreateSession("dentist-a", "1234")
	if err != nil {
		t.Fatal(err)
	}
	patientRepo := sqlite.NewPatientRepository(db)
	if err := patientRepo.Create(ctx, &domain.Patient{ID: "p", FirstName: "P", LastName: "Test"}); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.NewInsuranceRepository(db)
	svc := NewInsuranceService(repo, audit)
	chartRepo := sqlite.NewChartRepository(db)
	chart := NewChartService(chartRepo, audit)
	proc := sqlite.NewProcedureRepository(db)
	billing := NewBillingService(sqlite.NewClaimRepository(db), sqlite.NewPaymentRepository(db), sqlite.NewBundleRepository(db), proc, proc, chartRepo, patientRepo, config, NewSecretsService(), audit)
	AttachInsurancePricing(chart, billing, repo)
	for i, clinician := range []string{"dentist-a", "", "dentist-b"} {
		price, err := svc.Preview(token, &domain.PricingSource{Kind: "standard", PatientID: "p", CountryCode: domain.CountryUS, Code: "D2391", ClinicianID: clinician})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := chart.SaveToothCondition(token, &domain.SaveToothConditionPayload{ID: fmt.Sprintf("cond-%d", i), PatientID: "p", ToothNumber: 3 + i, ADACode: "D2391", Description: "Filling", Status: domain.ToothStatusTreatmentPlanned, Fee: price.Amount, Pricing: price}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := billing.CreateClaimFromChartConditions(token, "p", "", []string{"cond-0", "cond-2"}); !errors.Is(err, storage.ErrPricingClinician) {
		t.Fatal("treatments priced for different clinicians combined on one claim", err)
	}
	claim, err := billing.CreateClaimFromChartConditions(token, "p", "", []string{"cond-0", "cond-1"})
	if err != nil {
		t.Fatal(err)
	}
	if claim.ProviderID != "dentist-a" || len(claim.LineItems) != 2 {
		t.Fatal("claim provider not taken from treatment pricing", claim.ProviderID, len(claim.LineItems))
	}
}
