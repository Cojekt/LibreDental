package services_test

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage"
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

	auditRepo := sqlite.NewAuditRepository(db)
	configRepo := sqlite.NewPracticeConfigRepository(db)
	if err := configRepo.SaveProvider(context.Background(), &domain.Provider{ID: "prov_1", Name: "Test Prov", Pin: "1234", IsActive: true}); err != nil {
		t.Fatalf("Failed to save provider: %v", err)
	}
	auditSvc := services.NewAuditService(auditRepo, configRepo)
	token, err := auditSvc.CreateSession("prov_1", "1234")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	service := services.NewDocumentService(repo, tempDir, auditSvc)

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

		doc, err := service.SaveDocumentBase64(token, patient.ID, "Test Doc", "Desc", "other", "text/plain", b64)
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
		doc, err := service.SaveDocumentBase64(token, patient.ID, "Doc to read", "", string(domain.DocumentTypePDF), "application/pdf", b64Content)
		if err != nil {
			t.Fatalf("Failed to setup document: %v", err)
		}

		b64, err := service.GetDocumentBase64(token, doc.ID)
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
		doc, err := service.SaveDocumentBase64(token, patient.ID, "Doc to delete", "", string(domain.DocumentTypePDF), "application/pdf", b64Content)
		if err != nil {
			t.Fatalf("Failed to setup document: %v", err)
		}

		err = service.DeleteDocument(token, doc.ID)
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
		_, err = repo.GetByID(doc.ID)
		if err == nil {
			t.Errorf("Expected DB record to be deleted, but it exists")
		}
	})

	t.Run("DeleteDocumentWhenDBRowAlreadyDeleted", func(t *testing.T) {
		b64Content := base64.StdEncoding.EncodeToString([]byte("delete me too"))
		doc, err := service.SaveDocumentBase64(token, patient.ID, "Doc concurrent delete", "", string(domain.DocumentTypePDF), "application/pdf", b64Content)
		if err != nil {
			t.Fatalf("Failed to setup document: %v", err)
		}

		fullFilePath := filepath.Join(tempDir, "documents", doc.FilePath)

		// Create a mock repo where GetByID returns doc, but Delete returns storage.ErrNotFound
		mockRepo := &notFoundOnDeleteRepo{
			DocumentRepository: repo,
			realRepo:           repo,
			doc:                doc,
		}
		mockService := services.NewDocumentService(mockRepo, tempDir, auditSvc)

		// Call service.DeleteDocument - should proceed to remove the file from disk without returning an error
		err = mockService.DeleteDocument(token, doc.ID)
		if err != nil {
			t.Fatalf("Expected DeleteDocument to succeed even if repo.Delete returns ErrNotFound, got: %v", err)
		}

		// Verify file was removed from disk
		_, err = os.Stat(fullFilePath)
		if err == nil {
			t.Errorf("Expected file to be deleted from disk, but it still exists")
		} else if !os.IsNotExist(err) {
			t.Fatalf("Unexpected error checking file existence: %v", err)
		}
	})

	t.Run("ListDocuments", func(t *testing.T) {
		// Patient document fixture
		b64PatientContent := base64.StdEncoding.EncodeToString([]byte("patient doc"))
		_, err := service.SaveDocumentBase64(token, patient.ID, "Patient Document", "", string(domain.DocumentTypePDF), "application/pdf", b64PatientContent)
		if err != nil {
			t.Fatalf("Failed to setup patient document: %v", err)
		}

		docs, err := service.ListPatientDocuments(token, domain.DocumentFilter{PatientID: &patient.ID})
		if err != nil {
			t.Fatalf("Failed to list patient docs: %v", err)
		}
		if len(docs) == 0 {
			t.Errorf("Expected patient documents, got 0")
		}

		// Clinic document
		b64Content := base64.StdEncoding.EncodeToString([]byte("clinic doc"))
		_, err = service.SaveDocumentBase64(token, "", "Clinic Policy", "", string(domain.DocumentTypePDF), "application/pdf", b64Content)
		if err != nil {
			t.Fatalf("Failed to setup clinic document: %v", err)
		}

		clinicDocs, err := service.ListClinicDocuments(token)
		if err != nil {
			t.Fatalf("Failed to list clinic docs: %v", err)
		}
		if len(clinicDocs) == 0 {
			t.Errorf("Expected clinic documents, got 0")
		}
	})

	t.Run("SaveDocumentWithNestedPatientID", func(t *testing.T) {
		nestedPatientID := "dept/pat_sub_123"
		nestedPatient := &domain.Patient{
			ID:        nestedPatientID,
			FirstName: "Nested",
			LastName:  "Patient",
		}
		if err := patientRepo.Create(context.Background(), nestedPatient); err != nil {
			t.Fatalf("Failed to create nested test patient: %v", err)
		}

		content := "Nested patient document content"
		b64 := base64.StdEncoding.EncodeToString([]byte(content))

		doc, err := service.SaveDocumentBase64(token, nestedPatientID, "Nested Doc", "Desc", "other", "text/plain", b64)
		if err != nil {
			t.Fatalf("Failed to save nested document: %v", err)
		}

		fileContent, err := os.ReadFile(filepath.Join(tempDir, "documents", doc.FilePath))
		if err != nil {
			t.Fatalf("Failed to read nested document file at %s: %v", doc.FilePath, err)
		}
		if string(fileContent) != content {
			t.Errorf("Expected content %q, got %q", content, string(fileContent))
		}
	})

	t.Run("SaveDocumentPathTraversalRejected", func(t *testing.T) {
		invalidIDs := []string{
			"../outside",
			"/tmp/malicious",
			"pat/../../escaped",
			"..",
			".. ",
			"pat/.. ",
			"... ",
			".. .",
			". ",
		}
		b64 := base64.StdEncoding.EncodeToString([]byte("malicious content"))

		for _, badID := range invalidIDs {
			_, err := service.SaveDocumentBase64(token, badID, "Bad Doc", "Desc", "other", "text/plain", b64)
			if err == nil {
				t.Errorf("Expected error when saving with invalid patient ID %q, but got nil", badID)
			}
		}
	})

	t.Run("SaveDocumentNormalizesPatientID", func(t *testing.T) {
		content := "Normalized Patient ID test"
		b64 := base64.StdEncoding.EncodeToString([]byte(content))

		doc, err := service.SaveDocumentBase64(token, patient.ID+"/.", "Normalized Doc", "Desc", "other", "text/plain", b64)
		if err != nil {
			t.Fatalf("Failed to save document with raw patient ID: %v", err)
		}
		if *doc.PatientID != patient.ID {
			t.Errorf("Expected normalized PatientID %q, got %q", patient.ID, *doc.PatientID)
		}
	})
}

type notFoundOnDeleteRepo struct {
	services.DocumentRepository
	realRepo services.DocumentRepository
	doc      *domain.Document
}

func (m *notFoundOnDeleteRepo) GetByID(id string) (*domain.Document, error) {
	return m.doc, nil
}

func (m *notFoundOnDeleteRepo) Delete(id string) error {
	_ = m.realRepo.Delete(id)
	return storage.ErrNotFound
}
