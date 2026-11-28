package services

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestOpenableFileName(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        string // "" means refused
	}{
		{"xray.png", "image/png", "xray.png"},
		{"Scan.PDF", "application/pdf", "Scan.pdf"},
		{"Consent form", "application/pdf", "Consent form.pdf"},
		{"Consent v1.2", "Application/PDF; charset=binary", "Consent v1.2.pdf"},
		{"volume", "application/dicom", "volume.dcm"},
		{"Röntgen.jpg", "image/jpeg", "Röntgen.jpg"},
		{"x&calc&.pdf", "application/pdf", "x_calc_.pdf"},
		{"a%PATH%^|b.txt", "text/plain", "a_PATH___b.txt"},
		{".pdf", "", "document.pdf"},
		// A dangerous extension is never the last one.
		{"run.bat", "text/plain", "run.bat.txt"},
		{"setup.exe", "application/pdf", "setup.exe.pdf"},
		{"run.bat", "", ""},
		{"setup.exe", "application/x-msdownload", ""},
		{"run.bat.", "application/octet-stream", ""},
		{"run.cmd ", "", ""},
		{"page.html", "text/html", ""},
		{"image.svg", "image/svg+xml", ""},
		{"macro.docm", "", ""},
		{"link.lnk", "", ""},
		{"app.desktop", "", ""},
		{"noextension", "", ""},
	}
	for _, tt := range tests {
		got, err := openableFileName(&domain.Document{Name: tt.name, ContentType: tt.contentType})
		if tt.want == "" {
			if !errors.Is(err, ErrDocumentTypeNotOpenable) {
				t.Errorf("openableFileName(%q, %q) = %q, %v; want refusal", tt.name, tt.contentType, got, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("openableFileName(%q, %q) = %q, %v; want %q", tt.name, tt.contentType, got, err, tt.want)
		}
	}
}

func TestOpenableFileNameIsBounded(t *testing.T) {
	got, err := openableFileName(&domain.Document{Name: strings.Repeat("é", 300) + ".pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > maxOpenNameBytes+len(".pdf") || !strings.HasSuffix(got, "é.pdf") {
		t.Errorf("unexpected name of %d bytes: %q", len(got), got)
	}
}

// A refused document returns before anything is launched, so this needs no stand-in launcher.
func TestOpenDocumentRefusesScript(t *testing.T) {
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "documents.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auditDb, err := sqlite.OpenAudit(filepath.Join(dir, "documents_audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer auditDb.Close()
	configRepo := sqlite.NewPracticeConfigRepository(db)
	if err := configRepo.SaveProvider(context.Background(), &domain.Provider{ID: "prov_1", Name: "Test", Pin: "1234", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	auditSvc := NewAuditService(sqlite.NewAuditRepository(auditDb), configRepo)
	token, err := auditSvc.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatal(err)
	}

	svc := NewDocumentService(sqlite.NewDocumentRepository(db), dir, auditSvc)
	data := base64.StdEncoding.EncodeToString([]byte("content"))
	doc, err := svc.SaveDocumentBase64(token, "", "run.bat", "", "other", "", data)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.OpenDocument(token, doc.ID); !errors.Is(err, ErrDocumentTypeNotOpenable) {
		t.Errorf("OpenDocument = %v, want ErrDocumentTypeNotOpenable", err)
	}
}
