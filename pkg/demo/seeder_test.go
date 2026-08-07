package demo

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage/sqlite"
)

func TestSeedDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_demo.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open test sqlite db: %v", err)
	}
	defer db.Close()

	summary, err := SeedDatabase(db)
	if err != nil {
		t.Fatalf("SeedDatabase failed: %v", err)
	}

	if !summary.PracticeConfigured {
		t.Errorf("Expected PracticeConfigured to be true")
	}
	if summary.ProvidersCount != 3 {
		t.Errorf("Expected 3 providers, got %d", summary.ProvidersCount)
	}
	if summary.OperatoriesCount != 3 {
		t.Errorf("Expected 3 operatories, got %d", summary.OperatoriesCount)
	}
	if summary.PatientsCount != 6 {
		t.Errorf("Expected 6 patients, got %d", summary.PatientsCount)
	}
	if summary.AppointmentsCount != 10 {
		t.Errorf("Expected 10 appointments, got %d", summary.AppointmentsCount)
	}
	if summary.ConditionsCount != 12 {
		t.Errorf("Expected 12 conditions, got %d", summary.ConditionsCount)
	}

	ctx := context.Background()

	// Verify practice config
	configRepo := sqlite.NewPracticeConfigRepository(db)
	cfg, err := configRepo.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to query practice config: %v", err)
	}
	if cfg.ClinicName != "Apex Dental Studio" {
		t.Errorf("Expected ClinicName 'Apex Dental Studio', got '%s'", cfg.ClinicName)
	}

	// Verify patients
	patientRepo := sqlite.NewPatientRepository(db)
	patients, total, err := patientRepo.List(ctx, domain.PatientFilter{})
	if err != nil {
		t.Fatalf("Failed to list patients: %v", err)
	}
	if total != 6 || len(patients) != 6 {
		t.Errorf("Expected 6 patients in list, got total %d, count %d", total, len(patients))
	}

	// Verify appointments
	apptRepo := sqlite.NewAppointmentRepository(db)
	appts, err := apptRepo.List(ctx, domain.AppointmentFilter{})
	if err != nil {
		t.Fatalf("Failed to list appointments: %v", err)
	}
	if len(appts) != 10 {
		t.Errorf("Expected 10 appointments in list, got %d", len(appts))
	}

	uniqueDays := make(map[string]bool)
	afterHoursCount := 0

	for _, a := range appts {
		dayStr := a.StartTime.Format("2006-01-02")
		uniqueDays[dayStr] = true

		startHour := a.StartTime.Hour()
		endHour := a.EndTime.Hour()
		endMin := a.EndTime.Minute()

		// Normal business hours: 08:00 - 17:00
		isOutsideHours := startHour < 8 || startHour >= 17 || (endHour > 17 || (endHour == 17 && endMin > 0))
		if isOutsideHours {
			afterHoursCount++
		}
	}

	if len(uniqueDays) <= 1 {
		t.Errorf("Expected appointments to be spread across multiple days, got %d unique day(s)", len(uniqueDays))
	}

	if afterHoursCount != 1 {
		t.Errorf("Expected exactly 1 appointment outside normal business hours, got %d", afterHoursCount)
	}

	// Verify dental chart
	chartRepo := sqlite.NewChartRepository(db)
	chart, err := chartRepo.GetChart(ctx, "pat_101")
	if err != nil {
		t.Fatalf("Failed to get chart for pat_101: %v", err)
	}
	if len(chart.Conditions) != 2 {
		t.Errorf("Expected 2 conditions for pat_101, got %d", len(chart.Conditions))
	}
}
