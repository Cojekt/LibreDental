package demo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
	"github.com/google/uuid"
)

func seedDocuments(ctx context.Context, db *sqlite.DB, appDir, demoDataDir string, now time.Time, summary *SeedSummary) error {
	docRepo := sqlite.NewDocumentRepository(db)

	// Setup sample definitions to seed
	seedFile := func(patientID, name, desc, docType, mimeType, path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		docID := uuid.New().String()
		var relDir string
		var pID *string

		if patientID != "" {
			relDir = patientID
			pID = &patientID
		} else {
			relDir = "clinic"
		}

		targetDir := filepath.Join(appDir, "documents", relDir)
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("failed to create document directory: %w", err)
		}

		filePathRelative := filepath.Join(relDir, docID)
		fullPath := filepath.Join(appDir, "documents", filePathRelative)

		if err := os.WriteFile(fullPath, data, 0o644); err != nil {
			return fmt.Errorf("failed to write document file: %w", err)
		}

		doc := &domain.Document{
			ID:          docID,
			PatientID:   pID,
			Type:        domain.DocumentType(docType),
			Name:        name,
			Description: desc,
			FilePath:    filePathRelative,
			SizeBytes:   int64(len(data)),
			ContentType: mimeType,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := docRepo.Create(doc); err != nil {
			_ = os.Remove(fullPath)
			return fmt.Errorf("failed to save document record: %w", err)
		}

		summary.DocumentsCount++
		return nil
	}

	type seedDef struct {
		name      string
		desc      string
		docType   string
		mimeType  string
		path      string
		isClinic  bool
		patientID string
	}

	seeds := []seedDef{
		{"DICOM Multi-frame Volume", "A single large DICOM file containing multiple frames.", string(domain.DocumentTypeXRay), "application/dicom", "dicom-multiframe-test-volume.dcm", false, "pat_101"},
		{"Embedded Images Test PDF", "A document with embedded images to test media extraction.", string(domain.DocumentTypePDF), "application/pdf", "document-embedded-images-test.pdf", false, "pat_101"},
		{"LibreOffice Writer Test PDF", "A PDF generated natively by LibreOffice Writer.", string(domain.DocumentTypePDF), "application/pdf", "document-libreoffice-writer-test.pdf", false, "pat_101"},
		{"Minimal Test PDF", "A basic, standard 1-page PDF.", string(domain.DocumentTypeConsentForm), "application/pdf", "document-minimal-test.pdf", true, ""},
		{"Multipage Test PDF", "A multi-page document to test pagination parsing.", string(domain.DocumentTypePDF), "application/pdf", "document-multipage-test.pdf", false, "pat_101"},
		{"Password Protected Test PDF", "A password-protected PDF (edge case).", string(domain.DocumentTypePDF), "application/pdf", "document-password-protected-test.pdf", false, "pat_101"},
		{"Geometry Test PNG", "A PNG file with drawn geometry.", string(domain.DocumentTypeXRay), "image/png", "image-geometry-test.png", false, "pat_101"},
		{"Lossless Test TIFF", "A lossless TIFF image.", string(domain.DocumentTypeXRay), "image/tiff", "image-lossless-test.tiff", false, "pat_102"},
		{"Modern Format Test WebP", "A modern WebP image.", string(domain.DocumentTypeXRay), "image/webp", "image-modern-format-test.webp", false, "pat_102"},
		{"Photo Test JPEG", "A standard compressed JPEG photograph.", string(domain.DocumentTypeXRay), "image/jpeg", "image-photo-test.jpeg", false, "pat_102"},
		{"Static Test GIF", "A static GIF file.", string(domain.DocumentTypeXRay), "image/gif", "image-static-test.gif", false, "pat_102"},
	}

	for _, s := range seeds {
		fullPath := filepath.Join(demoDataDir, s.path)

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("demo document asset missing: %w", err)
		}

		pID := s.patientID
		if s.isClinic {
			pID = "" // clinic-wide document
		}

		if err := seedFile(pID, s.name, s.desc, s.docType, s.mimeType, fullPath); err != nil {
			return err
		}
	}

	return nil
}
