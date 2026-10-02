package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestProgramBridgeRepository(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test_program_bridge_repo.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := sqlite.NewProgramBridgeRepository(db)

	if _, err := repo.Get(ctx, "weasis"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unconfigured bridge, got %v", err)
	}
	if err := repo.Save(ctx, &domain.BridgeConfig{}); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for empty name, got %v", err)
	}

	cfg := &domain.BridgeConfig{Name: "weasis", Enabled: true, Path: "/opt/weasis/bin/Weasis", Args: `'$dicom:get -l "{dir}"'`}
	if err := repo.Save(ctx, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	created := cfg.CreatedAt

	got, err := repo.Get(ctx, "weasis")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.Enabled || got.Path != cfg.Path || got.Args != cfg.Args || !got.CreatedAt.Equal(created) {
		t.Errorf("unexpected config: %+v", got)
	}

	time.Sleep(time.Millisecond)
	if err := repo.Save(ctx, &domain.BridgeConfig{Name: "weasis", Path: "/other"}); err != nil {
		t.Fatalf("Save update: %v", err)
	}
	got, err = repo.Get(ctx, "weasis")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Enabled || got.Path != "/other" || got.Args != "" {
		t.Errorf("update not applied: %+v", got)
	}
	if !got.CreatedAt.Equal(created) || !got.UpdatedAt.After(created) {
		t.Errorf("timestamps wrong after update: created %v -> %v, updated %v", created, got.CreatedAt, got.UpdatedAt)
	}

	if err := repo.Save(ctx, &domain.BridgeConfig{Name: "custom"}); err != nil {
		t.Fatalf("Save custom: %v", err)
	}
	all, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 || all[0].Name != "custom" || all[1].Name != "weasis" {
		t.Errorf("unexpected list: %+v", all)
	}
}
