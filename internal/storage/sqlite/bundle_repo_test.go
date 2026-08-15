package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestBundleRepository(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_bundle_repo.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewBundleRepository(db)
	ctx := context.Background()

	// 1. Validation check for invalid input on Create
	invalidBundle := &domain.TreatmentBundle{ID: "", Shortname: "crwn", Name: "Crown Bundle"}
	if err := repo.Create(ctx, invalidBundle); err == nil {
		t.Errorf("Expected error when creating bundle with empty ID")
	}

	// 2. Create valid treatment bundle
	bundle := &domain.TreatmentBundle{
		ID:          "bnd_101",
		Shortname:   "crwn",
		Name:        "Single Crown Bundle",
		Description: "Includes prep, temporary, and final crown placement",
		TotalFee:    120000,
		Items: []domain.BundleItemTemplate{
			{
				ADACode:     "D2740",
				Description: "Crown - Porcelain/Ceramic Substrate",
				DefaultFee:  120000,
			},
		},
	}

	if err := repo.Create(ctx, bundle); err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	// 3. GetByID
	fetchedByID, err := repo.GetByID(ctx, "bnd_101")
	if err != nil {
		t.Fatalf("Failed to get bundle by ID: %v", err)
	}
	if fetchedByID.Name != "Single Crown Bundle" || fetchedByID.Shortname != "crwn" {
		t.Errorf("Unexpected bundle data: %+v", fetchedByID)
	}
	if len(fetchedByID.Items) != 1 {
		t.Fatalf("Expected 1 item template, got %d", len(fetchedByID.Items))
	}
	if fetchedByID.Items[0].ADACode != "D2740" {
		t.Errorf("Expected item ADACode 'D2740', got '%s'", fetchedByID.Items[0].ADACode)
	}

	// GetByID with invalid / nonexistent ID
	if _, err := repo.GetByID(ctx, ""); err == nil {
		t.Errorf("Expected error for empty ID in GetByID")
	}
	if _, err := repo.GetByID(ctx, "non_existent_bnd"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound for nonexistent bundle, got: %v", err)
	}

	// 4. GetByShortname
	fetchedByShortname, err := repo.GetByShortname(ctx, "crwn")
	if err != nil {
		t.Fatalf("Failed to get bundle by shortname: %v", err)
	}
	if fetchedByShortname.ID != "bnd_101" {
		t.Errorf("Expected ID 'bnd_101', got '%s'", fetchedByShortname.ID)
	}

	if _, err := repo.GetByShortname(ctx, ""); err == nil {
		t.Errorf("Expected error for empty shortname in GetByShortname")
	}
	if _, err := repo.GetByShortname(ctx, "unknown_shortname"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound for unknown shortname, got: %v", err)
	}

	// 5. Update
	fetchedByID.Name = "Full Porcelain Crown Bundle"
	fetchedByID.TotalFee = 125000
	if err := repo.Update(ctx, fetchedByID); err != nil {
		t.Fatalf("Failed to update bundle: %v", err)
	}

	updated, err := repo.GetByID(ctx, "bnd_101")
	if err != nil {
		t.Fatalf("Failed to get updated bundle: %v", err)
	}
	if updated.Name != "Full Porcelain Crown Bundle" || updated.TotalFee != 125000 {
		t.Errorf("Unexpected updated bundle data: %+v", updated)
	}

	// Update nonexistent
	if err := repo.Update(ctx, &domain.TreatmentBundle{ID: "no_such_bnd"}); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound updating nonexistent bundle, got: %v", err)
	}

	// 6. List Bundles
	// Create a second bundle
	secondBundle := &domain.TreatmentBundle{
		ID:        "bnd_102",
		Shortname: "clean",
		Name:      "Adult Prophy Bundle",
		TotalFee:  15000,
	}
	if err := repo.Create(ctx, secondBundle); err != nil {
		t.Fatalf("Failed to create second bundle: %v", err)
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list bundles: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Expected 2 bundles in list, got %d", len(list))
	}
	// List should be sorted by shortname ASC ("clean", then "crwn")
	if list[0].Shortname != "clean" || list[1].Shortname != "crwn" {
		t.Errorf("Expected shortname ordering ('clean', 'crwn'), got ('%s', '%s')", list[0].Shortname, list[1].Shortname)
	}

	// 7. Delete
	if err := repo.Delete(ctx, "bnd_101"); err != nil {
		t.Fatalf("Failed to delete bundle: %v", err)
	}
	if _, err := repo.GetByID(ctx, "bnd_101"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got: %v", err)
	}
	if err := repo.Delete(ctx, "bnd_101"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound deleting already deleted bundle, got: %v", err)
	}
}
