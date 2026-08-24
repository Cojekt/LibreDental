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
					Fee:              28000,
					InsuranceAllowed: 22400,
					PatientPortion:   5600,
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
					Fee:              120000,
					InsuranceAllowed: 96000,
					PatientPortion:   24000,
				},
				{
					ID:               "li_3",
					ToothConditionID: "cond_306",
					ToothNumber:      19,
					Surfaces:         []domain.ToothSurface{},
					ADACode:          "D2740",
					Description:      "Porcelain/Ceramic Crown",
					Fee:              135000,
					InsuranceAllowed: 108000,
					PatientPortion:   27000,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:               "claim_403",
			PatientID:        "pat_101",
			ProviderID:       "prov_101",
			AppointmentID:    "appt_201",
			InsuranceCarrier: "Cigna Dental",
			PolicyNumber:     "CIG-1234567",
			GroupNumber:      "GRP-99999",
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
					Fee:              11000,
					InsuranceAllowed: 11000,
					PatientPortion:   0,
				},
				{
					ID:               "li_5",
					ToothConditionID: "",
					ToothNumber:      0,
					Surfaces:         []domain.ToothSurface{},
					ADACode:          "D1110",
					Description:      "Prophylaxis - Adult",
					Fee:              14000,
					InsuranceAllowed: 14000,
					PatientPortion:   0,
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
			Amount:    5600,
			Method:    domain.PaymentMethodCreditCard,
			Date:      "2026-06-14",
			Notes:     "Patient 20% co-payment for D2392",
			CreatedAt: now,
		},
		{
			ID:        "pay_502",
			PatientID: "pat_101",
			ClaimID:   "claim_401",
			Amount:    22400,
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
