package demo

import (
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/LibreDental/libredental/internal/domain"
)

// seedDocuments uploads the fixture documents, then a realistic spread of everyday paperwork
// across the generated roster. It returns the generated documents' IDs.
func seedDocuments(g *ServiceGraph, token string, rng *rand.Rand, patients []*domain.Patient, demoDataDir string, summary *SeedSummary) ([]string, error) {
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

		// Everyday patient paperwork, reusing the fixtures above under realistic names.
		{"Signed Consent - Crown Preparation", "Informed consent for crown prep on #30, signed at chairside.", string(domain.DocumentTypeConsentForm), "application/pdf", "document-minimal-test.pdf", false, "pat_101"},
		{"Medical History Questionnaire", "New patient medical history intake form.", string(domain.DocumentTypePDF), "application/pdf", "document-multipage-test.pdf", false, "pat_103"},
		{"Referral Letter - Periodontist", "Referral for periodontal evaluation of lower anteriors.", string(domain.DocumentTypePDF), "application/pdf", "document-libreoffice-writer-test.pdf", false, "pat_103"},
		{"Insurance Card - Front", "Scan of the primary insurance card.", string(domain.DocumentTypeImage), "image/png", "image-geometry-test.png", false, "pat_104"},
		{"Treatment Plan Estimate", "Printed treatment plan and insurance estimate given to the patient.", string(domain.DocumentTypePDF), "application/pdf", "document-embedded-images-test.pdf", false, "pat_104"},
		{"Signed Consent - Whitening", "Consent for in-office whitening.", string(domain.DocumentTypeConsentForm), "application/pdf", "document-minimal-test.pdf", false, "pat_105"},
		{"Intraoral Photos", "Pre-treatment intraoral photographs.", string(domain.DocumentTypeImage), "image/jpeg", "image-photo-test.jpeg", false, "pat_106"},
		{"Lab Prescription - Night Guard", "Lab slip sent with impressions for an occlusal guard.", string(domain.DocumentTypePDF), "application/pdf", "document-libreoffice-writer-test.pdf", false, "pat_106"},

		// Clinic-wide forms and policies.
		{"Notice of Privacy Practices", "HIPAA notice given to every new patient.", string(domain.DocumentTypePDF), "application/pdf", "document-multipage-test.pdf", true, ""},
		{"Financial Policy", "Payment, insurance and cancellation policy.", string(domain.DocumentTypePDF), "application/pdf", "document-libreoffice-writer-test.pdf", true, ""},
		{"Blank Consent - Extraction", "Printable consent form for extractions.", string(domain.DocumentTypeConsentForm), "application/pdf", "document-minimal-test.pdf", true, ""},
	}

	// Everyday paperwork for a sample of the generated roster, using the small fixtures only.
	paperwork := []seedDef{
		{"Bitewing Radiographs", "Four bitewing images from recall visit.", string(domain.DocumentTypeXRay), "image/jpeg", "image-photo-test.jpeg", false, ""},
		{"Periapical Radiograph", "Single PA taken for a problem-focused exam.", string(domain.DocumentTypeXRay), "image/png", "image-geometry-test.png", false, ""},
		{"Insurance Card - Front", "Scan of the primary insurance card.", string(domain.DocumentTypeImage), "image/png", "image-geometry-test.png", false, ""},
		{"Signed HIPAA Acknowledgement", "Signed acknowledgement of the privacy notice.", string(domain.DocumentTypeConsentForm), "application/pdf", "document-minimal-test.pdf", false, ""},
		{"Signed Consent - Treatment", "Informed consent signed before treatment.", string(domain.DocumentTypeConsentForm), "application/pdf", "document-minimal-test.pdf", false, ""},
		{"Medical History Update", "Annual medical history review.", string(domain.DocumentTypePDF), "application/pdf", "document-multipage-test.pdf", false, ""},
		{"Referral Letter - Oral Surgeon", "Referral for third molar evaluation.", string(domain.DocumentTypePDF), "application/pdf", "document-libreoffice-writer-test.pdf", false, ""},
		{"Intraoral Photo", "Clinical photograph for the treatment plan.", string(domain.DocumentTypeImage), "image/webp", "image-modern-format-test.webp", false, ""},
	}
	firstGenerated := len(seeds)
	for _, i := range rng.Perm(len(patients))[:min(60, len(patients))] {
		for _, j := range rng.Perm(len(paperwork))[:1+rng.IntN(3)] {
			d := paperwork[j]
			d.patientID = patients[i].ID
			seeds = append(seeds, d)
		}
	}

	var generated []string
	for n, s := range seeds {
		fullPath := filepath.Join(demoDataDir, s.path)

		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("demo document asset missing: %w", err)
		}

		patientID := s.patientID
		if s.isClinic {
			patientID = "" // clinic-wide document
		}

		b64 := base64.StdEncoding.EncodeToString(data)
		doc, err := g.Document.SaveDocumentBase64(token, patientID, s.name, s.desc, s.docType, s.mimeType, b64)
		if err != nil {
			return nil, fmt.Errorf("failed to seed document %s: %w", s.name, err)
		}
		summary.DocumentsCount++
		if n >= firstGenerated {
			generated = append(generated, doc.ID)
		}
	}

	return generated, nil
}
