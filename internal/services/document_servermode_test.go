package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

// In LAN server mode, document actions that touch the server's own filesystem
// (opening in the OS app, exporting to a local path) must be refused: the
// machine running the backend is the server, not the caller's workstation, so a
// bound client must not be able to write arbitrary files there.
func TestDocumentService_LocalActionsRefusedInServerMode(t *testing.T) {
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	adb, err := sqlite.OpenAudit(filepath.Join(dir, "a.db"))
	if err != nil {
		t.Fatalf("open audit db: %v", err)
	}
	defer adb.Close()

	cfgRepo := sqlite.NewPracticeConfigRepository(db)
	ctx := context.Background()
	if err := cfgRepo.SaveProvider(ctx, &domain.Provider{ID: "p", Name: "P", Pin: "1357", IsActive: true}); err != nil {
		t.Fatalf("save provider: %v", err)
	}
	audit := NewAuditService(sqlite.NewAuditRepository(adb), cfgRepo)
	token, err := audit.CreateSession("p", "1357")
	if err != nil {
		t.Fatalf("session: %v", err)
	}

	svc := NewDocumentService(sqlite.NewDocumentRepository(db), dir, audit)
	doc, err := svc.SaveDocumentBase64(token, "", "notes.txt", "", "other", "text/plain", "aGVsbG8=")
	if err != nil {
		t.Fatalf("save document: %v", err)
	}
	dest := filepath.Join(dir, "exported.txt")

	svc.serverMode = true
	if err := svc.OpenDocument(token, doc.ID); !errors.Is(err, ErrDocumentActionDesktopOnly) {
		t.Errorf("OpenDocument in server mode = %v, want ErrDocumentActionDesktopOnly", err)
	}
	if err := svc.ExportDocumentToPath(token, doc.ID, dest); !errors.Is(err, ErrDocumentActionDesktopOnly) {
		t.Errorf("ExportDocumentToPath in server mode = %v, want ErrDocumentActionDesktopOnly", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("server-mode export wrote a file to the server filesystem (stat err = %v)", err)
	}

	// Desktop mode still exports normally.
	svc.serverMode = false
	if err := svc.ExportDocumentToPath(token, doc.ID, dest); err != nil {
		t.Fatalf("desktop export: %v", err)
	}
	if b, err := os.ReadFile(dest); err != nil || string(b) != "hello" {
		t.Errorf("desktop export content = %q (err %v), want \"hello\"", b, err)
	}
}
