package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type insuranceFixture struct {
	db       *DB
	repo     *InsuranceRepository
	path     string
	provider *domain.InsuranceProvider
}

func setupInsurance(t *testing.T) *insuranceFixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "insurance.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	f := &insuranceFixture{db: db, repo: NewInsuranceRepository(db), path: path, provider: &domain.InsuranceProvider{Name: " Example Insurance "}}
	ctx := context.Background()
	if err := f.repo.SaveProvider(ctx, f.provider); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"patient-a", "patient-b"} {
		if err := NewPatientRepository(db).Create(ctx, &domain.Patient{ID: id, FirstName: id, LastName: "Test"}); err != nil {
			t.Fatal(err)
		}
	}
	return f
}
func (f *insuranceFixture) fee(t *testing.T, code string, fee int64) {
	t.Helper()
	if err := f.repo.SaveFee(context.Background(), &domain.InsuranceFee{ProviderID: f.provider.ID, CountryCode: domain.CountryUS, Code: code, Fee: fee}); err != nil {
		t.Fatal(err)
	}
}
func (f *insuranceFixture) preview(t *testing.T, patient, code, kind string) *domain.PricingSource {
	t.Helper()
	s, err := f.repo.Preview(context.Background(), &domain.PricingSource{Kind: kind, PatientID: patient, CountryCode: domain.CountryUS, Code: code, ProviderID: f.provider.ID})
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func treatment(id string, s *domain.PricingSource) *domain.ToothCondition {
	return &domain.ToothCondition{ID: id, PatientID: s.PatientID, ToothNumber: 3, ADACode: s.Code, Description: "Test procedure", Status: domain.ToothStatusTreatmentPlanned, Fee: s.Amount, Pricing: s}
}

func TestInsuranceStaleConfirmationAndAtomicity(t *testing.T) {
	f := setupInsurance(t)
	ctx := context.Background()
	f.fee(t, "D2391", 0)
	review := f.preview(t, "patient-a", "D2391", "insurance")
	f.fee(t, "D2391", 100)
	if _, err := f.repo.ConfirmCondition(ctx, treatment("stale", review)); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale accepted", err)
	}
	if _, err := NewChartRepository(f.db).GetConditionByID(ctx, "stale"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("partial charge saved", err)
	}
	review = f.preview(t, "patient-a", "D2391", "insurance")
	wrong := treatment("wrong-patient", review)
	wrong.PatientID = "patient-b"
	if _, err := f.repo.ConfirmCondition(ctx, wrong); err == nil {
		t.Fatal("patient identity mismatch accepted")
	}
	f.fee(t, "D2391", 0)
	c := treatment("zero", f.preview(t, "patient-a", "D2391", "insurance"))
	if _, err := f.repo.ConfirmCondition(ctx, c); err != nil {
		t.Fatal(err)
	}
	stored, err := NewChartRepository(f.db).GetConditionByID(ctx, "zero")
	if err != nil || stored.Fee != 0 || stored.Pricing == nil {
		t.Fatal("zero not persisted", err)
	}
	direct := &domain.Claim{ID: "direct", PatientID: "patient-a", DateOfService: "2026-10-22", LineItems: []domain.ClaimLineItem{{ID: "direct-line", ADACode: "D2391", Fee: 0, Pricing: f.preview(t, "patient-a", "D2391", "insurance")}}}
	if err := f.repo.ConfirmClaim(ctx, direct, false); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.ConfirmClaim(ctx, direct, false); err != nil {
		t.Fatal("direct retry", err)
	}
	direct.LineItems[0].Fee = 1
	if err := f.repo.ConfirmClaim(ctx, direct, false); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("different retry accepted", err)
	}
}
func TestInsuranceStandardManualAndConcurrentRetry(t *testing.T) {
	f := setupInsurance(t)
	ctx := context.Background()
	config := NewPracticeConfigRepository(f.db)
	if err := config.SaveProvider(ctx, &domain.Provider{ID: "clinician", Name: "Test", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	proc := NewProcedureRepository(f.db)
	for _, s := range []*domain.FeeSchedule{{CountryCode: domain.CountryUS, Code: "D2391", CustomFee: 100}, {CountryCode: domain.CountryUS, Code: "D2391", ProviderID: "clinician", CustomFee: 0}} {
		if err := proc.Save(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	selection := &domain.PricingSource{Kind: "standard", PatientID: "patient-a", Code: "D2391", CountryCode: domain.CountryUS, ClinicianID: "clinician"}
	s, err := f.repo.Preview(ctx, selection)
	if err != nil || s.Amount != 0 || s.BaseFee != 14000 {
		t.Fatal(s, err)
	}
	f.fee(t, "D2391", 555)
	selection.Kind = "insurance"
	selection.ProviderID = f.provider.ID
	s, err = f.repo.Preview(ctx, selection)
	if err != nil || s.Amount != 555 {
		t.Fatal("combined prices", s, err)
	}
	selection.Kind = "manual"
	selection.Amount = 123
	s, err = f.repo.Preview(ctx, selection)
	if err != nil || s.Amount != 123 {
		t.Fatal(s, err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 4)
	raw, _ := json.Marshal(s)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var cp domain.PricingSource
			_ = json.Unmarshal(raw, &cp)
			_, err := f.repo.ConfirmCondition(ctx, treatment("same-id", &cp))
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM dental_conditions WHERE id='same-id'`).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate", count, err)
	}
}

func TestInsuranceNewClaimRowsRequireConfirmation(t *testing.T) {
	f := setupInsurance(t)
	ctx := context.Background()
	claim := &domain.Claim{ID: "legacy", PatientID: "patient-a", DateOfService: "2026-10-22", LineItems: []domain.ClaimLineItem{{ID: "old", ADACode: "D2391", Fee: 123}}}
	if err := NewClaimRepository(f.db).Create(ctx, claim); err != nil {
		t.Fatal(err)
	}
	claim.LineItems = append(claim.LineItems, domain.ClaimLineItem{ID: "new", ADACode: "D2391", Fee: 999})
	if err := f.repo.ConfirmClaim(ctx, claim, true); !errors.Is(err, storage.ErrInvalidInput) {
		t.Fatal("new row bypassed confirmation", err)
	}
	saved, err := NewClaimRepository(f.db).GetByID(ctx, claim.ID)
	if err != nil || len(saved.LineItems) != 1 || saved.LineItems[0].Fee != 123 {
		t.Fatal("failed confirmation partially saved claim", saved, err)
	}
}

func TestConcurrentChartImportsBillConditionOnce(t *testing.T) {
	f := setupInsurance(t)
	ctx := context.Background()
	f.fee(t, "D2391", 100)
	c := treatment("chart-condition", f.preview(t, "patient-a", "D2391", "insurance"))
	if _, err := f.repo.ConfirmCondition(ctx, c); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 4)
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			claim := &domain.Claim{ID: fmt.Sprintf("claim-%d", i), PatientID: "patient-a", DateOfService: "2026-10-22", LineItems: []domain.ClaimLineItem{{ID: fmt.Sprintf("line-%d", i), ADACode: "D2391", ToothConditionID: c.ID, Fee: 100}}}
			results <- f.repo.ConfirmClaim(ctx, claim, false)
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if !errors.Is(err, storage.ErrConflict) {
			t.Fatal(err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("chart condition billed %d times", succeeded)
	}
}

func TestInsuranceCatalog(t *testing.T) {
	f := setupInsurance(t)
	ctx := context.Background()
	for _, name := range []string{"example insurance", " EXAMPLE   Insurance "} {
		if err := f.repo.SaveProvider(ctx, &domain.InsuranceProvider{Name: name}); !errors.Is(err, storage.ErrInsuranceDuplicate) {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"", " ", string(make([]byte, 161))} {
		if err := f.repo.SaveProvider(ctx, &domain.InsuranceProvider{Name: name}); !errors.Is(err, storage.ErrInvalidInput) {
			t.Fatal(err)
		}
	}
	other := &domain.InsuranceProvider{Name: "Other insurer"}
	if err := f.repo.SaveProvider(ctx, other); err != nil {
		t.Fatal(err)
	}
	f.fee(t, "D2391", 0)
	if err := f.repo.SaveFee(ctx, &domain.InsuranceFee{ProviderID: other.ID, CountryCode: domain.CountryUS, Code: "D2391", Fee: 111}); err != nil {
		t.Fatal(err)
	}
	if got := f.preview(t, "patient-a", "D2391", "insurance"); got.Amount != 0 {
		t.Fatal(got)
	}
	for _, fee := range []int64{-1, 9007199254740992} {
		if err := f.repo.SaveFee(ctx, &domain.InsuranceFee{ProviderID: f.provider.ID, CountryCode: domain.CountryUS, Code: "D2391", Fee: fee}); err == nil {
			t.Fatal("invalid fee accepted")
		}
	}
	if err := f.repo.SaveFee(ctx, &domain.InsuranceFee{ProviderID: "missing", CountryCode: domain.CountryUS, Code: "D2391", Fee: 1}); err == nil {
		t.Fatal("missing provider accepted")
	}
	if err := f.repo.SaveFee(ctx, &domain.InsuranceFee{ProviderID: f.provider.ID, CountryCode: domain.CountryUS, Code: "missing", Fee: 1}); err == nil {
		t.Fatal("missing procedure accepted")
	}
	if err := f.repo.DeleteFee(ctx, f.provider.ID, domain.CountryUS, "D2391"); err != nil {
		t.Fatal(err)
	}
	fees, err := f.repo.ListFees(ctx, other.ID, domain.CountryUS)
	if err != nil || len(fees) != 1 || fees[0].Fee != 111 {
		t.Fatal(fees, err)
	}
	if _, err := f.repo.Preview(ctx, &domain.PricingSource{Kind: "insurance", PatientID: "patient-a", CountryCode: domain.CountryUS, Code: "D2391", ProviderID: f.provider.ID}); !errors.Is(err, storage.ErrInsuranceFeeMissing) {
		t.Fatal(err)
	}
	f.fee(t, "D2391", 100)
	f.provider.Archived = true
	if err := f.repo.SaveProvider(ctx, f.provider); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Preview(ctx, &domain.PricingSource{Kind: "insurance", PatientID: "patient-a", CountryCode: domain.CountryUS, Code: "D2391", ProviderID: f.provider.ID}); !errors.Is(err, storage.ErrInsuranceUnavailable) {
		t.Fatal(err)
	}
	f.provider.Archived = false
	f.provider.Name = "Renamed"
	if err := f.repo.SaveProvider(ctx, f.provider); err != nil {
		t.Fatal(err)
	}
	if got := f.preview(t, "patient-a", "D2391", "insurance"); got.ProviderName != "Renamed" {
		t.Fatal(got)
	}
}
func TestInsuranceSnapshotsAndRestart(t *testing.T) {
	f := setupInsurance(t)
	ctx := context.Background()
	f.fee(t, "D2391", 10000)
	a := treatment("a", f.preview(t, "patient-a", "D2391", "insurance"))
	if _, err := f.repo.ConfirmCondition(ctx, a); err != nil {
		t.Fatal(err)
	}
	claim := &domain.Claim{ID: "old-claim", PatientID: a.PatientID, DateOfService: "2026-10-23", LineItems: []domain.ClaimLineItem{{ID: "a", ToothConditionID: a.ID, ADACode: a.ADACode, Fee: a.Fee}}}
	if err := f.repo.ConfirmClaim(ctx, claim, false); err != nil {
		t.Fatal(err)
	}
	f.fee(t, "D2391", 14000)
	b := treatment("b", f.preview(t, "patient-b", "D2391", "insurance"))
	if _, err := f.repo.ConfirmCondition(ctx, b); err != nil {
		t.Fatal(err)
	}
	f.provider.Name = "New name"
	f.provider.Archived = true
	if err := f.repo.SaveProvider(ctx, f.provider); err != nil {
		t.Fatal(err)
	}
	if err := f.db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := Open(f.path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	saved, err := NewChartRepository(db).GetConditionByID(ctx, "a")
	if err != nil || saved.Fee != 10000 || saved.Pricing.ProviderName != "Example Insurance" {
		t.Fatal(saved, err)
	}
	savedB, err := NewChartRepository(db).GetConditionByID(ctx, "b")
	if err != nil || savedB.Fee != 14000 {
		t.Fatal(savedB, err)
	}
	old, err := NewClaimRepository(db).GetByID(ctx, "old-claim")
	if err != nil || old.LineItems[0].Fee != 10000 || old.LineItems[0].Pricing.ProviderName != "Example Insurance" {
		t.Fatal(old, err)
	}
	old.Notes = "Unrelated edit"
	if err := NewInsuranceRepository(db).ConfirmClaim(ctx, old, true); err != nil {
		t.Fatal(err)
	}
	if old.LineItems[0].Fee != 10000 {
		t.Fatal("historical claim repriced")
	}
}

func TestInsuranceClaimChangesCannotBypassConfirmation(t *testing.T) {
	for _, insured := range []bool{false, true} {
		t.Run(fmt.Sprintf("insured=%t", insured), func(t *testing.T) {
			f := setupInsurance(t)
			ctx := context.Background()
			f.fee(t, "D2391", 100)
			claim := &domain.Claim{ID: "correction", PatientID: "patient-a", DateOfService: "2026-10-23", LineItems: []domain.ClaimLineItem{{ID: "line", ADACode: "D2391", Fee: 100}}}
			if insured {
				claim.LineItems[0].Pricing = f.preview(t, "patient-a", "D2391", "insurance")
				if err := f.repo.ConfirmClaim(ctx, claim, false); err != nil {
					t.Fatal(err)
				}
			} else if err := NewClaimRepository(f.db).Create(ctx, claim); err != nil {
				t.Fatal(err)
			}
			for _, codeChanged := range []bool{false, true} {
				changed, err := NewClaimRepository(f.db).GetByID(ctx, claim.ID)
				if err != nil {
					t.Fatal(err)
				}
				changed.LineItems[0].Pricing = nil
				if codeChanged {
					changed.LineItems[0].ADACode = "D0120"
				} else {
					changed.LineItems[0].Fee = 200
				}
				if err := f.repo.ConfirmClaim(ctx, changed, true); !errors.Is(err, storage.ErrInvalidInput) {
					t.Fatal("unconfirmed correction accepted", err)
				}
				saved, err := NewClaimRepository(f.db).GetByID(ctx, claim.ID)
				if err != nil || saved.LineItems[0].Fee != 100 || saved.LineItems[0].ADACode != "D2391" {
					t.Fatal("failed correction changed stored claim", saved, err)
				}
			}
			claim.Notes = "Notes only"
			claim.LineItems[0].Pricing = nil
			if err := f.repo.ConfirmClaim(ctx, claim, true); err != nil {
				t.Fatal("unchanged line requires confirmation", err)
			}
			if insured && claim.LineItems[0].Pricing == nil {
				t.Fatal("unchanged source lost")
			}
			manual, err := f.repo.Preview(ctx, &domain.PricingSource{Kind: "manual", PatientID: "patient-a", CountryCode: domain.CountryUS, Code: "D2391", Amount: 200})
			if err != nil {
				t.Fatal(err)
			}
			claim.LineItems[0].Fee = 200
			claim.LineItems[0].Pricing = manual
			if err := f.repo.ConfirmClaim(ctx, claim, true); err != nil {
				t.Fatal("confirmed correction rejected", err)
			}
		})
	}
}

func TestInsuranceConditionResponseOmitsRequestToken(t *testing.T) {
	b, err := json.Marshal(&domain.ToothCondition{})
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(b, &response); err != nil {
		t.Fatal(err)
	}
	if _, ok := response["expected_updated_at"]; ok {
		t.Fatal("request-only token leaked into response")
	}
}
