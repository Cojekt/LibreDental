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

func TestDocumentRepository(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_libredental.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewDocumentRepository(db)
	patientRepo := sqlite.NewPatientRepository(db)
	ctx := context.Background()

	// Create a test patient
	patient := &domain.Patient{
		ID:          "pat_123",
		FirstName:   "John",
		LastName:    "Doe",
		DateOfBirth: time.Now(),
		Sex:         domain.SexMale,
		Status:      domain.StatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := patientRepo.Create(ctx, patient); err != nil {
		t.Fatalf("Failed to create test patient: %v", err)
	}

	t.Run("Create and GetByID", func(t *testing.T) {
		doc := &domain.Document{
			ID:          "doc_1",
			PatientID:   &patient.ID,
			Type:        domain.DocumentTypeXRay,
			Name:        "Bitewing",
			Description: "Annual bitewing xray",
			FilePath:    "some/path.png",
			SizeBytes:   1024,
			ContentType: "image/png",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := repo.Create(doc); err != nil {
			t.Fatalf("Failed to create document: %v", err)
		}

		fetched, err := repo.GetByID(doc.ID)
		if err != nil {
			t.Fatalf("Failed to get document: %v", err)
		}
		if fetched.ID != doc.ID {
			t.Errorf("Expected ID %s, got %s", doc.ID, fetched.ID)
		}
		if fetched.Name != doc.Name {
			t.Errorf("Expected Name %s, got %s", doc.Name, fetched.Name)
		}
		if *fetched.PatientID != *doc.PatientID {
			t.Errorf("Expected PatientID %s, got %s", *doc.PatientID, *fetched.PatientID)
		}
	})

	t.Run("ListByFilter", func(t *testing.T) {
		doc := &domain.Document{
			ID:          "doc_2",
			PatientID:   &patient.ID,
			Type:        domain.DocumentTypeConsentForm,
			Name:        "Consent",
			FilePath:    "consent.pdf",
			SizeBytes:   2048,
			ContentType: "application/pdf",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := repo.Create(doc); err != nil {
			t.Fatalf("Failed to create document: %v", err)
		}

		docs, err := repo.ListByFilter(domain.DocumentFilter{PatientID: &patient.ID})
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}
		if len(docs) < 1 {
			t.Errorf("Expected at least 1 document, got %d", len(docs))
		}

		docsFiltered, err := repo.ListByFilter(domain.DocumentFilter{PatientID: &patient.ID, Type: domain.DocumentTypeConsentForm})
		if err != nil {
			t.Fatalf("Failed to list filtered documents: %v", err)
		}
		if len(docsFiltered) != 1 {
			t.Fatalf("Expected 1 filtered document, got %d", len(docsFiltered))
		}
		if docsFiltered[0].ID != doc.ID {
			t.Errorf("Expected ID %s, got %s", doc.ID, docsFiltered[0].ID)
		}
	})

	t.Run("ListClinicDocuments", func(t *testing.T) {
		doc := &domain.Document{
			ID:          "doc_3",
			Type:        domain.DocumentTypePDF,
			Name:        "Office Policy",
			FilePath:    "policy.pdf",
			SizeBytes:   100,
			ContentType: "application/pdf",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := repo.Create(doc); err != nil {
			t.Fatalf("Failed to create clinic document: %v", err)
		}

		docs, err := repo.ListClinicDocuments()
		if err != nil {
			t.Fatalf("Failed to list clinic documents: %v", err)
		}
		found := false
		for _, d := range docs {
			if d.ID == doc.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find clinic document %s", doc.ID)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		doc := &domain.Document{
			ID:          "doc_4",
			Type:        domain.DocumentTypeOther,
			Name:        "To Delete",
			FilePath:    "delete.me",
			SizeBytes:   10,
			ContentType: "text/plain",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := repo.Create(doc); err != nil {
			t.Fatalf("Failed to create document for deletion: %v", err)
		}

		if err := repo.Delete(doc.ID); err != nil {
			t.Fatalf("Failed to delete document: %v", err)
		}

		_, err = repo.GetByID(doc.ID)
		if err != storage.ErrNotFound {
			t.Errorf("Expected storage.ErrNotFound getting deleted document, got %v", err)
		}

		err = repo.Delete("non_existent_id")
		if err != storage.ErrNotFound {
			t.Errorf("Expected storage.ErrNotFound deleting non-existent document, got %v", err)
		}
	})
}
