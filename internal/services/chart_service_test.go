package services_test

import (
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestChartService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_chart_service.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	patientRepo := sqlite.NewPatientRepository(db)
	patientService := services.NewPatientService(patientRepo)

	chartRepo := sqlite.NewChartRepository(db)
	chartService := services.NewChartService(chartRepo)

	pat, err := patientService.CreatePatient(&domain.Patient{
		ID:        "pat_svc_1",
		FirstName: "Bob",
		LastName:  "Builder",
	})
	if err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	cond, err := chartService.SaveToothCondition(&domain.ToothCondition{
		PatientID:   pat.ID,
		ToothNumber: 14,
		Surfaces:    []domain.ToothSurface{domain.SurfaceOcclusal},
		ADACode:     "D2391",
		Description: "1-Surface Posterior Composite",
		Status:      domain.ToothStatusTreatmentPlanned,
		Fee:         120.0,
	})
	if err != nil {
		t.Fatalf("Failed to save condition: %v", err)
	}
	if cond.ID == "" {
		t.Errorf("Expected auto-generated ID for tooth condition")
	}

	chart, err := chartService.GetPatientChart(pat.ID)
	if err != nil {
		t.Fatalf("Failed to get chart: %v", err)
	}
	if len(chart.Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(chart.Conditions))
	}

	err = chartService.DeleteToothCondition(cond.ID)
	if err != nil {
		t.Fatalf("Failed to delete condition: %v", err)
	}

	emptyChart, err := chartService.GetPatientChart(pat.ID)
	if err != nil {
		t.Fatalf("Failed to get empty chart: %v", err)
	}
	if len(emptyChart.Conditions) != 0 {
		t.Errorf("Expected 0 conditions after deletion, got %d", len(emptyChart.Conditions))
	}
}
