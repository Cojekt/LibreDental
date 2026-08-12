package services

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/google/uuid"
)

// DocumentRepository defines the data access interface for documents.
type DocumentRepository interface {
	Create(doc *domain.Document) error
	GetByID(id string) (*domain.Document, error)
	ListByFilter(filter domain.DocumentFilter) ([]domain.Document, error)
	ListClinicDocuments() ([]domain.Document, error)
	Delete(id string) error
}

// DocumentService handles business logic and file storage for documents.
type DocumentService struct {
	repo   DocumentRepository
	appDir string // Base application data directory
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(repo DocumentRepository, appDir string) *DocumentService {
	return &DocumentService{
		repo:   repo,
		appDir: appDir,
	}
}

// getDocumentsBasePath returns the base directory where all documents are stored.
func (s *DocumentService) getDocumentsBasePath() string {
	return filepath.Join(s.appDir, "documents")
}

// getPatientDocumentsPath returns the directory where a specific patient's documents are stored.
func (s *DocumentService) getPatientDocumentsPath(patientID string) string {
	return filepath.Join(s.getDocumentsBasePath(), patientID)
}

// getClinicDocumentsPath returns the directory where clinic-wide documents are stored.
func (s *DocumentService) getClinicDocumentsPath() string {
	return filepath.Join(s.getDocumentsBasePath(), "clinic")
}

// SaveDocumentBase64 saves a document from a base64 encoded string.
// If patientID is empty, it saves as a clinic document.
func (s *DocumentService) SaveDocumentBase64(patientID, name, description, docType, contentType, base64Data string) (*domain.Document, error) {
	// Decode base64
	// Allow frontend to pass "data:image/png;base64,iVBORw0KGgo..." or just raw base64
	b64str := base64Data
	if strings.Contains(base64Data, ",") {
		parts := strings.SplitN(base64Data, ",", 2)
		b64str = parts[1]
	}

	data, err := base64.StdEncoding.DecodeString(b64str)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 data: %w", err)
	}

	return s.SaveDocumentBytes(patientID, name, description, docType, contentType, data)
}

// SaveDocumentBytes saves a document from a byte array.
func (s *DocumentService) SaveDocumentBytes(patientID, name, description, docType, contentType string, data []byte) (*domain.Document, error) {
	docID := uuid.New().String()
	
	var targetDir string
	var pID *string
	
	if patientID != "" {
		targetDir = s.getPatientDocumentsPath(patientID)
		pID = &patientID
	} else {
		targetDir = s.getClinicDocumentsPath()
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create document directory: %w", err)
	}

	// Assuming we keep the original extension from contentType or name if needed, but for now just use docID as filename without extension, or guess extension.
	// For safety, let's store it purely as docID in the filesystem to prevent traversal attacks.
	filePathRelative := filepath.Join(filepath.Base(targetDir), docID)
	if patientID == "" {
		filePathRelative = filepath.Join("clinic", docID)
	}
	
	fullPath := filepath.Join(s.getDocumentsBasePath(), filePathRelative)

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write document file: %w", err)
	}

	now := time.Now()
	doc := &domain.Document{
		ID:          docID,
		PatientID:   pID,
		Type:        domain.DocumentType(docType),
		Name:        name,
		Description: description,
		FilePath:    filePathRelative,
		SizeBytes:   int64(len(data)),
		ContentType: contentType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(doc); err != nil {
		// Clean up file if db save fails
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("failed to save document record: %w", err)
	}

	return doc, nil
}

// GetDocumentMeta retrieves document metadata.
func (s *DocumentService) GetDocumentMeta(id string) (*domain.Document, error) {
	return s.repo.GetByID(id)
}

// GetDocumentBase64 retrieves a document and returns its contents as a base64 encoded string.
func (s *DocumentService) GetDocumentBase64(id string) (string, error) {
	doc, err := s.repo.GetByID(id)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(s.getDocumentsBasePath(), doc.FilePath)
	
	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to open document file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read document file: %w", err)
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// ListPatientDocuments lists documents for a specific patient.
func (s *DocumentService) ListPatientDocuments(patientID string) ([]domain.Document, error) {
	return s.repo.ListByFilter(domain.DocumentFilter{PatientID: &patientID})
}

// ListClinicDocuments lists clinic-wide documents.
func (s *DocumentService) ListClinicDocuments() ([]domain.Document, error) {
	return s.repo.ListClinicDocuments()
}

// DeleteDocument deletes a document and its associated file.
func (s *DocumentService) DeleteDocument(id string) error {
	doc, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}

	fullPath := filepath.Join(s.getDocumentsBasePath(), doc.FilePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove document file: %w", err)
	}

	return nil
}
