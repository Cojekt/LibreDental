package services_test

import (
	"context"
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
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	auditDbPath := filepath.Join(tempDir, "test_audit_service.db")
	auditDb, err := sqlite.OpenAudit(auditDbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite audit db: %v", err)
	}
	defer auditDb.Close()

	auditRepo := sqlite.NewAuditRepository(auditDb)
	configRepo := sqlite.NewPracticeConfigRepository(db)
	configRepo.SaveProvider(context.Background(), &domain.Provider{ID: "test_user", Name: "Test User", Pin: "1234", IsActive: true})
	auditService := services.NewAuditService(auditRepo, configRepo)
	token, _ := auditService.CreateSession("test_user", "1234")

	patientRepo := sqlite.NewPatientRepository(db)
	patientService := services.NewPatientService(patientRepo, auditService)

	chartRepo := sqlite.NewChartRepository(db)
	chartService := services.NewChartService(chartRepo, auditService)

	pat, err := patientService.CreatePatient(token, &domain.Patient{
		ID:        "pat_svc_1",
		FirstName: "Bob",
		LastName:  "Builder",
	})
	if err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	cond, err := chartService.SaveToothCondition(token, &domain.ToothCondition{
		PatientID:   pat.ID,
		ToothNumber: 14,
		Surfaces:    []domain.ToothSurface{domain.SurfaceOcclusal},
		ADACode:     "D2391",
		Description: "1-Surface Posterior Composite",
		Status:      domain.ToothStatusTreatmentPlanned,
		Fee:         12000,
	})
	if err != nil {
		t.Fatalf("Failed to save condition: %v", err)
	}
	if cond.ID == "" {
		t.Errorf("Expected auto-generated ID for tooth condition")
	}

	chart, err := chartService.GetPatientChart(token, pat.ID)
	if err != nil {
		t.Fatalf("Failed to get chart: %v", err)
	}
	if len(chart.Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(chart.Conditions))
	}

	err = chartService.DeleteToothCondition(token, cond.ID, pat.ID)
	if err != nil {
		t.Fatalf("Failed to delete condition: %v", err)
	}

	emptyChart, err := chartService.GetPatientChart(token, pat.ID)
	if err != nil {
		t.Fatalf("Failed to get empty chart: %v", err)
	}
	if len(emptyChart.Conditions) != 0 {
		t.Errorf("Expected 0 conditions after deletion, got %d", len(emptyChart.Conditions))
	}
}
