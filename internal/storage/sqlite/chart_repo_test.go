package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestChartRepository_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_chart.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	patientRepo := sqlite.NewPatientRepository(db)
	chartRepo := sqlite.NewChartRepository(db)
	ctx := context.Background()

	// 1. Create Patient first (for FK integrity)
	pat := &domain.Patient{
		ID:        "pat_chart_1",
		FirstName: "Jane",
		LastName:  "Doe",
	}
	if err := patientRepo.Create(ctx, pat); err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	// 2. Save ToothCondition 1 (Caries on tooth #3 MO surfaces)
	cond1 := &domain.ToothCondition{
		ID:          "cond_1",
		PatientID:   "pat_chart_1",
		ToothNumber: 3,
		Surfaces:    []domain.ToothSurface{domain.SurfaceMesial, domain.SurfaceOcclusal},
		ADACode:     "D2392",
		Description: "Posterior 2-Surface Composite Resin",
		Status:      domain.ToothStatusTreatmentPlanned,
		Fee:         175.0,
	}

	if err := chartRepo.SaveCondition(ctx, cond1); err != nil {
		t.Fatalf("Failed to save tooth condition 1: %v", err)
	}

	// 3. Save ToothCondition 2 (Missing tooth #1)
	cond2 := &domain.ToothCondition{
		ID:          "cond_2",
		PatientID:   "pat_chart_1",
		ToothNumber: 1,
		Surfaces:    []domain.ToothSurface{},
		ADACode:     "D7140",
		Description: "Extracted / Missing Tooth",
		Status:      domain.ToothStatusMissing,
		Fee:         0.0,
	}

	if err := chartRepo.SaveCondition(ctx, cond2); err != nil {
		t.Fatalf("Failed to save tooth condition 2: %v", err)
	}

	// 4. Fetch Chart
	chart, err := chartRepo.GetChart(ctx, "pat_chart_1")
	if err != nil {
		t.Fatalf("Failed to get chart: %v", err)
	}

	if chart.PatientID != "pat_chart_1" {
		t.Errorf("Expected patient_id 'pat_chart_1', got '%s'", chart.PatientID)
	}

	if len(chart.Conditions) != 2 {
		t.Fatalf("Expected 2 conditions, got %d", len(chart.Conditions))
	}

	// Condition order by tooth_number ASC: tooth 1 should come before tooth 3
	if chart.Conditions[0].ToothNumber != 1 || chart.Conditions[0].Status != domain.ToothStatusMissing {
		t.Errorf("Expected first condition tooth #1 missing, got tooth #%d status %s",
			chart.Conditions[0].ToothNumber, chart.Conditions[0].Status)
	}

	if chart.Conditions[1].ToothNumber != 3 || len(chart.Conditions[1].Surfaces) != 2 {
		t.Errorf("Expected second condition tooth #3 with 2 surfaces, got tooth #%d surfaces %v",
			chart.Conditions[1].ToothNumber, chart.Conditions[1].Surfaces)
	}

	// 5. Update Condition 1 (Change status to completed)
	cond1.Status = domain.ToothStatusCompleted
	if err := chartRepo.SaveCondition(ctx, cond1); err != nil {
		t.Fatalf("Failed to update tooth condition: %v", err)
	}

	updatedChart, err := chartRepo.GetChart(ctx, "pat_chart_1")
	if err != nil {
		t.Fatalf("Failed to get updated chart: %v", err)
	}
	if updatedChart.Conditions[1].Status != domain.ToothStatusCompleted {
		t.Errorf("Expected status completed, got %s", updatedChart.Conditions[1].Status)
	}

	// 6. Delete Condition 2
	if err := chartRepo.DeleteCondition(ctx, "cond_2"); err != nil {
		t.Fatalf("Failed to delete condition 2: %v", err)
	}

	finalChart, err := chartRepo.GetChart(ctx, "pat_chart_1")
	if err != nil {
		t.Fatalf("Failed to get final chart: %v", err)
	}
	if len(finalChart.Conditions) != 1 {
		t.Errorf("Expected 1 condition after deletion, got %d", len(finalChart.Conditions))
	}

	// 7. Non-existent delete returns ErrNotFound
	if err := chartRepo.DeleteCondition(ctx, "non_existent"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound on non-existent delete, got: %v", err)
	}
}
