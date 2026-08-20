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
	patientRepo := sqlite.NewPatientRepository(db)

	patients, _, err := patientRepo.List(ctx, domain.PatientFilter{Limit: 2})
	if err != nil {
		return fmt.Errorf("failed to list patients for seeding documents: %w", err)
	}
	if len(patients) == 0 {
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

	if err := seedFile(patients[0].ID, "Annual Bitewing X-Ray (Right)", "Standard annual bitewing x-ray", string(domain.DocumentTypeXRay), "image/jpeg", xray1Path); err != nil {
		return err
	}
	if err := seedFile(patients[0].ID, "Annual Bitewing X-Ray (Left)", "Standard annual bitewing x-ray", string(domain.DocumentTypeXRay), "image/jpeg", xray2Path); err != nil {
		return err
	}

	// Seed clinic-wide document (patientID = "" stores patient_id = NULL in SQL)
	if err := seedFile("", "Practice Privacy & Consent Form", "Standard practice operational policies and general consent documentation", string(domain.DocumentTypeConsentForm), "application/pdf", pdfPath); err != nil {
		return err
	}

	return nil
}
