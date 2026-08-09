package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func seedFeeSchedules(ctx context.Context, procedureRepo *sqlite.ProcedureRepository, now time.Time, summary *SeedSummary) error {
	feeSchedules := []*domain.FeeSchedule{
		{
			ID:          "fee_sched_1",
			CountryCode: domain.CountryUS,
			Code:        "D2392",
			ProviderID:  "prov_101",
			CustomFee:   295.00,
			UpdatedAt:   now,
		},
		{
			ID:          "fee_sched_2",
			CountryCode: domain.CountryUS,
			Code:        "D2740",
			ProviderID:  "", // Practice-wide default
			CustomFee:   1300.00,
			UpdatedAt:   now,
		},
	}

	for _, fs := range feeSchedules {
		if err := procedureRepo.Save(ctx, fs); err != nil {
			return fmt.Errorf("failed to seed fee schedule %s: %w", fs.ID, err)
		}
		summary.FeeSchedulesCount++
	}
	return nil
}

func seedBundles(ctx context.Context, bundleRepo *sqlite.BundleRepository, now time.Time, summary *SeedSummary) error {
	bundles := []*domain.TreatmentBundle{
		{
			ID:          "bundle_1",
			Shortname:   "exam-clean",
			Name:        "New Patient Exam & Hygiene Prophylaxis",
			Description: "Comprehensive oral evaluation, bitewing x-rays, and dental cleaning",
			Items: []domain.BundleItemTemplate{
				{ADACode: "D0120", Description: "Periodic Oral Evaluation", DefaultFee: 110.00},
				{ADACode: "D1110", Description: "Prophylaxis - Adult", DefaultFee: 140.00},
				{ADACode: "D0274", Description: "Bitewings - Four Radiographic Images", DefaultFee: 75.00},
			},
			TotalFee:  325.00,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "bundle_2",
			Shortname:   "rct-crown",
			Name:        "Endodontic Root Canal & Crown Therapy",
			Description: "Molar root canal treatment followed by porcelain/ceramic crown restoration",
			Items: []domain.BundleItemTemplate{
				{ADACode: "D3330", Description: "Molar Endodontic Therapy (Root Canal)", DefaultFee: 1200.00},
				{ADACode: "D2740", Description: "Porcelain/Ceramic Crown", DefaultFee: 1350.00},
			},
			TotalFee:  2550.00,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, b := range bundles {
		if err := bundleRepo.Create(ctx, b); err != nil {
			return fmt.Errorf("failed to seed treatment bundle %s: %w", b.ID, err)
		}
		summary.BundlesCount++
	}
	return nil
}

func seedClaims(ctx context.Context, claimRepo *sqlite.ClaimRepository, now time.Time, summary *SeedSummary) error {
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
					Fee:              280.00,
					InsuranceAllowed: 224.00,
					PatientPortion:   56.00,
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
					Fee:              1200.00,
					InsuranceAllowed: 960.00,
					PatientPortion:   240.00,
				},
				{
					ID:               "li_3",
					ToothConditionID: "cond_306",
					ToothNumber:      19,
					Surfaces:         []domain.ToothSurface{},
					ADACode:          "D2740",
					Description:      "Porcelain/Ceramic Crown",
					Fee:              1350.00,
					InsuranceAllowed: 1080.00,
					PatientPortion:   270.00,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, c := range claims {
		if err := claimRepo.Create(ctx, c); err != nil {
			return fmt.Errorf("failed to seed claim %s: %w", c.ID, err)
		}
		summary.ClaimsCount++
	}
	return nil
}

func seedPayments(ctx context.Context, paymentRepo *sqlite.PaymentRepository, now time.Time, summary *SeedSummary) error {
	payments := []*domain.Payment{
		{
			ID:        "pay_501",
			PatientID: "pat_101",
			ClaimID:   "claim_401",
			Amount:    56.00,
			Method:    domain.PaymentMethodCreditCard,
			Date:      "2026-06-14",
			Notes:     "Patient 20% co-payment for D2392",
			CreatedAt: now,
		},
		{
			ID:        "pay_502",
			PatientID: "pat_101",
			ClaimID:   "claim_401",
			Amount:    224.00,
			Method:    domain.PaymentMethodInsurance,
			Date:      "2026-06-15",
			Notes:     "Delta Dental insurance claim payment EOB #99482",
			CreatedAt: now,
		},
	}

	for _, p := range payments {
		if err := paymentRepo.Create(ctx, p); err != nil {
			return fmt.Errorf("failed to seed payment %s: %w", p.ID, err)
		}
		summary.PaymentsCount++
	}
	return nil
}
