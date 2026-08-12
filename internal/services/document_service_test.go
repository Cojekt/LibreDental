package services_test

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestDocumentService(t *testing.T) {
	tempDir := t.TempDir()

	// Set up database for real repository
	dbPath := filepath.Join(tempDir, "test_documents.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewDocumentRepository(db)
	service := services.NewDocumentService(repo, tempDir)

	// Create a test patient since documents have a foreign key to patients
	patientRepo := sqlite.NewPatientRepository(db)
	patient := &domain.Patient{
		ID:        "pat_test_1",
		FirstName: "Test",
		LastName:  "Patient",
	}
	if err := patientRepo.Create(context.Background(), patient); err != nil {
		t.Fatalf("Failed to create test patient: %v", err)
	}

	t.Run("SaveDocumentBase64", func(t *testing.T) {
		content := "Hello World"
		b64 := base64.StdEncoding.EncodeToString([]byte(content))

		doc, err := service.SaveDocumentBase64(patient.ID, "Test Doc", "Desc", "other", "text/plain", b64)
		if err != nil {
			t.Fatalf("Failed to save document: %v", err)
		}
		if doc.Name != "Test Doc" {
			t.Errorf("Expected Name 'Test Doc', got %s", doc.Name)
		}
		if *doc.PatientID != patient.ID {
			t.Errorf("Expected PatientID '%s', got %s", patient.ID, *doc.PatientID)
		}

		// Verify file exists
		fileContent, err := os.ReadFile(filepath.Join(tempDir, "documents", doc.FilePath))
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(fileContent) != content {
			t.Errorf("Expected file content %q, got %q", content, string(fileContent))
		}
	})

	t.Run("GetDocumentBase64", func(t *testing.T) {
		content := "File Content"

		// directly create a document via service to ensure file is on disk
		b64Content := base64.StdEncoding.EncodeToString([]byte(content))
		doc, err := service.SaveDocumentBase64(patient.ID, "Doc to read", "", string(domain.DocumentTypePDF), "application/pdf", b64Content)
		if err != nil {
			t.Fatalf("Failed to setup document: %v", err)
		}

		b64, err := service.GetDocumentBase64(doc.ID)
		if err != nil {
			t.Fatalf("Failed to get document base64: %v", err)
		}

		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			t.Fatalf("Failed to decode base64: %v", err)
		}
		if string(decoded) != content {
			t.Errorf("Expected decoded content %q, got %q", content, string(decoded))
		}
	})

	t.Run("DeleteDocument", func(t *testing.T) {
		// create a document to delete
		b64Content := base64.StdEncoding.EncodeToString([]byte("delete me"))
		doc, err := service.SaveDocumentBase64(patient.ID, "Doc to delete", "", string(domain.DocumentTypePDF), "application/pdf", b64Content)
		if err != nil {
			t.Fatalf("Failed to setup document: %v", err)
		}

		err = service.DeleteDocument(doc.ID)
		if err != nil {
			t.Fatalf("Failed to delete document: %v", err)
		}

		// Verify file is gone
		_, err = os.Stat(filepath.Join(tempDir, "documents", doc.FilePath))
		if err == nil {
			t.Errorf("Expected file to be deleted, but it exists")
		} else if !os.IsNotExist(err) {
			t.Fatalf("Unexpected error checking if file exists: %v", err)
		}

		// Verify DB record is gone
		_, err = service.GetDocumentMeta(doc.ID)
		if err == nil {
			t.Errorf("Expected DB record to be deleted, but it exists")
		}
	})

	t.Run("ListDocuments", func(t *testing.T) {
		docs, err := service.ListPatientDocuments(patient.ID)
		if err != nil {
			t.Fatalf("Failed to list patient docs: %v", err)
		}
		if len(docs) == 0 {
			t.Errorf("Expected patient documents, got 0")
		}

		// Clinic document
		b64Content := base64.StdEncoding.EncodeToString([]byte("clinic doc"))
		_, err = service.SaveDocumentBase64("", "Clinic Policy", "", string(domain.DocumentTypePDF), "application/pdf", b64Content)
		if err != nil {
			t.Fatalf("Failed to setup clinic document: %v", err)
		}

		clinicDocs, err := service.ListClinicDocuments()
		if err != nil {
			t.Fatalf("Failed to list clinic docs: %v", err)
		}
		if len(clinicDocs) == 0 {
			t.Errorf("Expected clinic documents, got 0")
		}
	})
}
