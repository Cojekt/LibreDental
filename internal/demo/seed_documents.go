package demo

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LibreDental/libredental/internal/domain"
)

func seedDocuments(g *ServiceGraph, token string, demoDataDir string, summary *SeedSummary) error {
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

		data, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("demo document asset missing: %w", err)
		}

		patientID := s.patientID
		if s.isClinic {
			patientID = "" // clinic-wide document
		}

		b64 := base64.StdEncoding.EncodeToString(data)
		if _, err := g.Document.SaveDocumentBase64(token, patientID, s.name, s.desc, s.docType, s.mimeType, b64); err != nil {
			return fmt.Errorf("failed to seed document %s: %w", s.name, err)
		}
		summary.DocumentsCount++
	}

	return nil
}
