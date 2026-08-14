package demo

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func seedDocuments(ctx context.Context, db *sqlite.DB, appDir, demoDataDir string, now time.Time, summary *SeedSummary) error {
	docRepo := sqlite.NewDocumentRepository(db)
	docService := services.NewDocumentService(docRepo, appDir)
	patientRepo := sqlite.NewPatientRepository(db)

	patients, _, err := patientRepo.List(ctx, domain.PatientFilter{Limit: 2})
	if err != nil || len(patients) == 0 {
		return nil // skip if no patients
	}

	// Read test files from demoDataDir
	pdfPath := filepath.Join(demoDataDir, "Test.pdf")
	xray1Path := filepath.Join(demoDataDir, "x-ray-1.jpeg")
	xray2Path := filepath.Join(demoDataDir, "x-ray-2.jpeg")

	seedFile := func(patientID, name, desc, docType, mimeType, path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		b64Data := base64.StdEncoding.EncodeToString(data)
		_, err = docService.SaveDocumentBase64(patientID, name, desc, docType, mimeType, b64Data)
		if err != nil {
			return fmt.Errorf("failed to save doc %s: %w", name, err)
		}
		summary.DocumentsCount++
		return nil
	}

	if err := seedFile(patients[0].ID, "Annual Bitewing X-Ray (Right)", "Standard annual bitewing x-ray", string(domain.DocumentTypeXRay), "image/jpeg", xray1Path); err != nil {
		return err
	}
	if err := seedFile(patients[0].ID, "Annual Bitewing X-Ray (Left)", "Standard annual bitewing x-ray", string(domain.DocumentTypeXRay), "image/jpeg", xray2Path); err != nil {
		return err
	}

	var secondPatientID string
	if len(patients) > 1 {
		secondPatientID = patients[1].ID
	} else {
		secondPatientID = patients[0].ID
	}

	if err := seedFile(secondPatientID, "Patient Consent Form", "Signed consent form for general dentistry", string(domain.DocumentTypeConsentForm), "application/pdf", pdfPath); err != nil {
		return err
	}

	return nil
}
