package demo

import (
	"fmt"
	"sort"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

func seedFeeSchedules(g *ServiceGraph, token string, now time.Time, summary *SeedSummary) error {
	feeSchedules := []*domain.FeeSchedule{
		{
			ID:          "fee_sched_1",
			CountryCode: domain.CountryUS,
			Code:        "D2392",
			ProviderID:  "prov_101",
			CustomFee:   29500,
			UpdatedAt:   now,
		},
		{
			ID:          "fee_sched_2",
			CountryCode: domain.CountryUS,
			Code:        "D2740",
			ProviderID:  "", // Practice-wide default
			CustomFee:   130000,
			UpdatedAt:   now,
		},
	}

	for _, fs := range feeSchedules {
		if _, err := g.Billing.SaveFeeSchedule(token, fs); err != nil {
			return fmt.Errorf("failed to seed fee schedule %s: %w", fs.ID, err)
		}
		summary.FeeSchedulesCount++
	}
	return nil
}

// demoPlans maps insurance plan names to their generated IDs.
type demoPlans map[string]string

func seedInsurancePlans(g *ServiceGraph, token string, summary *SeedSummary) (demoPlans, error) {
	plans := []struct {
		name     string
		archived bool
		fees     map[string]int64
	}{
		{"Delta Dental PPO", false, map[string]int64{"D0120": 5200, "D1110": 9800, "D2392": 24000}},
		{"MetLife Dental", false, map[string]int64{"D3330": 88000, "D2740": 102000}},
		{"Cigna Dental Health", false, map[string]int64{"D0120": 5500, "D1110": 9500}},
		{"Legacy Dental Network", true, map[string]int64{"D0120": 4800}},
	}
	ids := demoPlans{}
	for _, plan := range plans {
		p, err := g.Insurance.SaveProvider(token, &domain.InsuranceProvider{Name: plan.name})
		if err != nil {
			return nil, fmt.Errorf("failed to seed insurance plan %s: %w", plan.name, err)
		}
		ids[plan.name] = p.ID
		summary.InsurancePlans++
		codes := make([]string, 0, len(plan.fees))
		for code := range plan.fees {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		for _, code := range codes {
			if err := g.Insurance.SaveFee(token, &domain.InsuranceFee{ProviderID: p.ID, CountryCode: domain.CountryUS, Code: code, Fee: plan.fees[code]}); err != nil {
				return nil, fmt.Errorf("failed to seed %s fee for %s: %w", plan.name, code, err)
			}
			summary.InsuranceFees++
		}
		if plan.archived {
			p.Archived = true
			if _, err := g.Insurance.SaveProvider(token, p); err != nil {
				return nil, fmt.Errorf("failed to archive insurance plan %s: %w", plan.name, err)
			}
		}
	}
	return ids, nil
}

// demoPrice selects a pricing source the way the pricing review does in the app.
// plan is an insurance plan name for kind "insurance"; amount is used for kind "manual".
type demoPrice struct {
	kind      string
	clinician string
	plan      string
	amount    int64
}

func (p demoPrice) preview(g *ServiceGraph, token string, plans demoPlans, patient, code string) (*domain.PricingSource, error) {
	src, err := g.Insurance.Preview(token, &domain.PricingSource{
		Kind:        p.kind,
		PatientID:   patient,
		CountryCode: domain.CountryUS,
		Code:        code,
		ClinicianID: p.clinician,
		ProviderID:  plans[p.plan],
		Amount:      p.amount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to price %s for %s: %w", code, patient, err)
	}
	return src, nil
}

func seedBundles(g *ServiceGraph, token string, now time.Time, summary *SeedSummary) error {
	bundles := []*domain.TreatmentBundle{
		{
			ID:          "bundle_1",
			Shortname:   "exam-clean",
			Name:        "New Patient Exam & Hygiene Prophylaxis",
			Description: "Comprehensive oral evaluation, bitewing x-rays, and dental cleaning",
			Items: []domain.BundleItemTemplate{
				{ADACode: "D0120", Description: "Periodic Oral Evaluation", DefaultFee: 11000},
				{ADACode: "D1110", Description: "Prophylaxis - Adult", DefaultFee: 14000},
				{ADACode: "D0274", Description: "Bitewings - Four Radiographic Images", DefaultFee: 7500},
			},
			TotalFee:  32500,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "bundle_2",
			Shortname:   "rct-crown",
			Name:        "Endodontic Root Canal & Crown Therapy",
			Description: "Molar root canal treatment followed by porcelain/ceramic crown restoration",
			Items: []domain.BundleItemTemplate{
				{ADACode: "D3330", Description: "Molar Endodontic Therapy (Root Canal)", DefaultFee: 120000},
				{ADACode: "D2740", Description: "Porcelain/Ceramic Crown", DefaultFee: 135000},
			},
			TotalFee:  255000,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, b := range bundles {
		if _, err := g.Billing.CreateBundle(token, b); err != nil {
			return fmt.Errorf("failed to seed treatment bundle %s: %w", b.ID, err)
		}
		summary.BundlesCount++
	}
	return nil
}

func seedClaims(g *ServiceGraph, token string, now time.Time, plans demoPlans, summary *SeedSummary) error {
	claims := []*domain.Claim{
		{
			ID:               "claim_401",
			PatientID:        "pat_101",
			ProviderID:       "prov_101",
			AppointmentID:    "appt_201",
			InsuranceCarrier: "Delta Dental PPO",
			PolicyNumber:     "DEL-9948201",
			GroupNumber:      "GRP-10023",
			DateOfService:    "2026-06-13",
			Status:           domain.ClaimStatusPaid,
			Notes:            "Restorative composite filling claim processed and paid.",
			LineItems: []domain.ClaimLineItem{
				{
					ID:               "li_1",
					ToothConditionID: "cond_301",
					ToothNumber:      3,
					Surfaces:         []domain.ToothSurface{domain.SurfaceMesial, domain.SurfaceOcclusal},
					ADACode:          "D2392",
					Description:      "2-Surface Posterior Composite Restoration",
					Fee:              29500,
					InsuranceAllowed: 23600,
					PatientPortion:   5900,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:               "claim_402",
			PatientID:        "pat_103",
			ProviderID:       "prov_102",
			AppointmentID:    "appt_203",
			InsuranceCarrier: "MetLife Dental",
			PolicyNumber:     "MET-8840192",
			GroupNumber:      "GRP-88102",
			DateOfService:    "2026-06-14",
			Status:           domain.ClaimStatusSubmitted,
			Notes:            "Endodontic therapy and crown restoration submitted to insurer.",
			LineItems: []domain.ClaimLineItem{
				{
					ID:               "li_2",
					ToothConditionID: "cond_305",
					ToothNumber:      19,
					Surfaces:         []domain.ToothSurface{domain.SurfaceOcclusal},
					ADACode:          "D3330",
					Description:      "Molar Endodontic Therapy (Root Canal)",
					Fee:              88000,
					InsuranceAllowed: 70400,
					PatientPortion:   17600,
				},
				{
					ID:               "li_3",
					ToothConditionID: "cond_306",
					ToothNumber:      19,
					Surfaces:         []domain.ToothSurface{},
					ADACode:          "D2740",
					Description:      "Porcelain/Ceramic Crown",
					Fee:              130000,
					InsuranceAllowed: 104000,
					PatientPortion:   26000,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:               "claim_403",
			PatientID:        "pat_102",
			ProviderID:       "prov_101",
			AppointmentID:    "",
			InsuranceCarrier: "Cigna Dental Health",
			PolicyNumber:     "CGN-3310492",
			GroupNumber:      "GRP-44012",
			DateOfService:    "2026-06-15",
			Status:           domain.ClaimStatusDraft,
			Notes:            "Routine exam and cleaning ready for submission.",
			LineItems: []domain.ClaimLineItem{
				{
					ID:               "li_4",
					ToothConditionID: "",
					ToothNumber:      0,
					Surfaces:         []domain.ToothSurface{},
					ADACode:          "D0120",
					Description:      "Periodic Oral Evaluation",
					Fee:              5500,
					InsuranceAllowed: 5500,
					PatientPortion:   0,
				},
				{
					ID:               "li_5",
					ToothConditionID: "",
					ToothNumber:      0,
					Surfaces:         []domain.ToothSurface{},
					ADACode:          "D1110",
					Description:      "Prophylaxis - Adult",
					Fee:              9500,
					InsuranceAllowed: 9500,
					PatientPortion:   0,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	// Lines linked to a charted treatment copy its saved pricing; the rest need a review.
	direct := map[string]demoPrice{
		"li_4": {kind: "insurance", clinician: "prov_101", plan: "Cigna Dental Health"},
		"li_5": {kind: "insurance", clinician: "prov_101", plan: "Cigna Dental Health"},
	}
	for _, c := range claims {
		for i := range c.LineItems {
			li := &c.LineItems[i]
			price, ok := direct[li.ID]
			if !ok {
				continue
			}
			src, err := price.preview(g, token, plans, c.PatientID, li.ADACode)
			if err != nil {
				return err
			}
			li.Pricing = src
		}
		if _, err := g.Billing.CreateClaim(token, c); err != nil {
			return fmt.Errorf("failed to seed claim %s: %w", c.ID, err)
		}
		summary.ClaimsCount++
	}
	return nil
}

func seedPayments(g *ServiceGraph, token string, now time.Time, summary *SeedSummary) error {
	payments := []*domain.Payment{
		{
			ID:        "pay_501",
			PatientID: "pat_101",
			ClaimID:   "claim_401",
			Amount:    5900,
			Method:    domain.PaymentMethodCreditCard,
			Date:      "2026-06-14",
			Notes:     "Patient 20% co-payment for D2392",
			CreatedAt: now,
		},
		{
			ID:        "pay_502",
			PatientID: "pat_101",
			ClaimID:   "claim_401",
			Amount:    23600,
			Method:    domain.PaymentMethodInsurance,
			Date:      "2026-06-15",
			Notes:     "Delta Dental insurance claim payment EOB #99482",
			CreatedAt: now,
		},
	}

	for _, p := range payments {
		if _, err := g.Billing.RecordPayment(token, p); err != nil {
			return fmt.Errorf("failed to seed payment %s: %w", p.ID, err)
		}
		summary.PaymentsCount++
	}
	return nil
}
