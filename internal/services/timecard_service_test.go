package services_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestTimecardService_ValidationAndErrors(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_timecard_svc.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	configRepo := sqlite.NewPracticeConfigRepository(db)
	timecardRepo := sqlite.NewTimecardRepository(db)
	service := services.NewTimecardService(timecardRepo, configRepo)
	ctx := context.Background()

	prov := &domain.Provider{
		ID:         "prov_svc_1",
		Name:       "Dr. Service",
		Role:       domain.RoleDentist,
		HourlyRate: 75,
		IsActive:   true,
	}
	if err := configRepo.SaveProvider(ctx, prov); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}

	t.Run("ClockIn duplicate active check", func(t *testing.T) {
		tc1, err := service.ClockIn("prov_svc_1")
		if err != nil {
			t.Fatalf("ClockIn failed: %v", err)
		}
		if tc1 == nil {
			t.Fatalf("Expected timecard on ClockIn, got nil")
		}

		// Second clock in should fail
		_, err = service.ClockIn("prov_svc_1")
		if err == nil {
			t.Fatalf("Expected error when clocking in twice, got nil")
		}

		// Clock out
		_, err = service.ClockOut("prov_svc_1")
		if err != nil {
			t.Fatalf("ClockOut failed: %v", err)
		}
	})

	t.Run("ListTimecards invalid date parsing", func(t *testing.T) {
		_, err := service.ListTimecards("prov_svc_1", "invalid-date", "")
		if err == nil {
			t.Errorf("Expected error for invalid start date, got nil")
		}

		_, err = service.ListTimecards("prov_svc_1", "", "invalid-date")
		if err == nil {
			t.Errorf("Expected error for invalid end date, got nil")
		}

		validDate := time.Now().Format(time.RFC3339)
		list, err := service.ListTimecards("prov_svc_1", validDate, validDate)
		if err != nil {
			t.Errorf("Expected success for valid dates, got %v", err)
		}
		if list == nil {
			t.Errorf("Expected non-nil slice from ListTimecards")
		}
	})

	t.Run("CreateManualTimecard validation", func(t *testing.T) {
		validDate := time.Now().Format(time.RFC3339)

		// Non-positive minutes
		err := service.CreateManualTimecard("prov_svc_1", 0, validDate)
		if err == nil {
			t.Errorf("Expected error for 0 minutes in CreateManualTimecard, got nil")
		}

		err = service.CreateManualTimecard("prov_svc_1", -30, validDate)
		if err == nil {
			t.Errorf("Expected error for negative minutes in CreateManualTimecard, got nil")
		}

		// Invalid date format
		err = service.CreateManualTimecard("prov_svc_1", 60, "not-a-date")
		if err == nil {
			t.Errorf("Expected error for invalid date format in CreateManualTimecard, got nil")
		}

		// Valid manual timecard
		err = service.CreateManualTimecard("prov_svc_1", 60, validDate)
		if err != nil {
			t.Errorf("Expected success for valid manual timecard, got %v", err)
		}
	})
}
