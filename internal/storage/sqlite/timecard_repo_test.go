package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestTimecardRepository_EmptyListReturnsNonNil(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_timecard_empty.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewTimecardRepository(db)
	ctx := context.Background()

	timecards, err := repo.ListTimecards(ctx, "prov_101", nil, nil)
	if err != nil {
		t.Fatalf("ListTimecards failed: %v", err)
	}
	if timecards == nil {
		t.Fatalf("Expected non-nil empty slice for ListTimecards, got nil")
	}
	if len(timecards) != 0 {
		t.Errorf("Expected 0 timecards, got %d", len(timecards))
	}
}

func TestTimecardRepository_DeleteMissingReturnsErrNotFound(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_timecard_delete.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewTimecardRepository(db)
	ctx := context.Background()

	err = repo.DeleteTimecard(ctx, "non_existent_tc")
	if err != storage.ErrNotFound {
		t.Fatalf("Expected storage.ErrNotFound when deleting missing timecard, got: %v", err)
	}
}

func TestTimecardRepository_ActiveTimecardUniqueness(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_timecard_unique.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	configRepo := sqlite.NewPracticeConfigRepository(db)
	repo := sqlite.NewTimecardRepository(db)
	ctx := context.Background()

	// Create provider first due to FK constraint
	prov := &domain.Provider{
		ID:       "prov_unique_1",
		Name:     "Dr. Unique",
		Role:     domain.RoleDentist,
		IsActive: true,
	}
	if err := configRepo.SaveProvider(ctx, prov); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}

	// Create first active timecard
	tc1 := &domain.Timecard{
		ID:         "tc_active_1",
		ProviderID: "prov_unique_1",
		ClockIn:    time.Now(),
		HourlyRate: 50,
	}
	if err := repo.SaveTimecard(ctx, tc1); err != nil {
		t.Fatalf("Failed to save first active timecard: %v", err)
	}

	// Attempting to create a second active timecard for the same provider should fail due to partial unique index
	tc2 := &domain.Timecard{
		ID:         "tc_active_2",
		ProviderID: "prov_unique_1",
		ClockIn:    time.Now(),
		HourlyRate: 50,
	}
	err = repo.SaveTimecard(ctx, tc2)
	if err == nil {
		t.Fatalf("Expected unique constraint error when inserting second active timecard for provider, but got no error")
	}

	// Clocking out first timecard allows another active timecard
	now := time.Now()
	tc1.ClockOut = &now
	tc1.TotalMinutes = 60
	tc1.TotalPay = 50
	if err := repo.SaveTimecard(ctx, tc1); err != nil {
		t.Fatalf("Failed to clock out first timecard: %v", err)
	}

	// Now inserting tc2 should succeed since tc1 is closed
	if err := repo.SaveTimecard(ctx, tc2); err != nil {
		t.Fatalf("Failed to save new active timecard after clocking out previous: %v", err)
	}
}
