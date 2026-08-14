package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func seedChartConditions(ctx context.Context, chartRepo *sqlite.ChartRepository, now time.Time, summary *SeedSummary) error {
	conditions := []*domain.ToothCondition{
		{
			ID:          "cond_301",
			PatientID:   "pat_101",
			ToothNumber: 3,
			Surfaces:    []domain.ToothSurface{domain.SurfaceMesial, domain.SurfaceOcclusal},
			ADACode:     "D2392",
			Description: "2-Surface Posterior Composite Restoration",
			Status:      domain.ToothStatusCompleted,
			Fee:         28000,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_302",
			PatientID:   "pat_101",
			ToothNumber: 14,
			Surfaces:    []domain.ToothSurface{domain.SurfaceOcclusal},
			ADACode:     "D2391",
			Description: "1-Surface Posterior Composite",
			Status:      domain.ToothStatusExisting,
			Fee:         19000,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_303",
			PatientID:   "pat_102",
			ToothNumber: 30,
			Surfaces:    []domain.ToothSurface{domain.SurfaceMesial, domain.SurfaceOcclusal, domain.SurfaceDistal},
			ADACode:     "D2393",
			Description: "3-Surface Posterior Composite",
			Status:      domain.ToothStatusTreatmentPlanned,
			Fee:         35000,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_304",
			PatientID:   "pat_102",
			ToothNumber: 1,
			Surfaces:    []domain.ToothSurface{},
			ADACode:     "D7140",
			Description: "Impacted Wisdom Tooth Extraction",
			Status:      domain.ToothStatusCompleted,
			Fee:         45000,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_305",
			PatientID:   "pat_103",
			ToothNumber: 19,
			Surfaces:    []domain.ToothSurface{domain.SurfaceOcclusal},
			ADACode:     "D3330",
			Description: "Molar Endodontic Therapy (Root Canal)",
			Status:      domain.ToothStatusTreatmentPlanned,
			Fee:         120000,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_306",
			PatientID:   "pat_103",
			ToothNumber: 19,
			Surfaces:    []domain.ToothSurface{},
			ADACode:     "D2740",
			Description: "Porcelain/Ceramic Crown",
			Status:      domain.ToothStatusTreatmentPlanned,
			Fee:         135000,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_307",
			PatientID:   "pat_103",
			ToothNumber: 32,
			Surfaces:    []domain.ToothSurface{},
			ADACode:     "D7140",
			Description: "Missing Tooth (Extracted)",
			Status:      domain.ToothStatusMissing,
			Fee:         0,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_308",
			PatientID:   "pat_105",
			ToothNumber: 14,
			Surfaces:    []domain.ToothSurface{},
			ADACode:     "D2750",
			Description: "Porcelain Fused to High Noble Metal Crown",
			Status:      domain.ToothStatusTreatmentPlanned,
			Fee:         140000,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_309",
			PatientID:   "pat_105",
			ToothNumber: 18,
			Surfaces:    []domain.ToothSurface{domain.SurfaceMesial, domain.SurfaceOcclusal},
			ADACode:     "D2150",
			Description: "Amalgam Restoration 2 Surfaces",
			Status:      domain.ToothStatusExisting,
			Fee:         22000,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_310",
			PatientID:   "pat_106",
			ToothNumber: 8,
			Surfaces:    []domain.ToothSurface{domain.SurfaceIncisal, domain.SurfaceFacial},
			ADACode:     "D2331",
			Description: "Anterior Composite 2 Surfaces",
			Status:      domain.ToothStatusExisting,
			Fee:         26000,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_311",
			PatientID:   "pat_106",
			ToothNumber: 9,
			Surfaces:    []domain.ToothSurface{domain.SurfaceIncisal, domain.SurfaceFacial},
			ADACode:     "D2331",
			Description: "Anterior Composite 2 Surfaces",
			Status:      domain.ToothStatusExisting,
			Fee:         26000,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_312",
			PatientID:   "pat_104",
			ToothNumber: 30,
			Surfaces:    []domain.ToothSurface{domain.SurfaceOcclusal},
			ADACode:     "D1351",
			Description: "Dental Sealant - Per Tooth",
			Status:      domain.ToothStatusCompleted,
			Fee:         6500,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, cond := range conditions {
		if err := chartRepo.SaveCondition(ctx, cond); err != nil {
			return fmt.Errorf("failed to seed tooth condition %s: %w", cond.ID, err)
		}
		summary.ConditionsCount++
	}
	return nil
}
