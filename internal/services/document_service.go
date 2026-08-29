package services

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
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
	repo         DocumentRepository
	appDir       string // Base application data directory
	auditService *AuditService
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(repo DocumentRepository, appDir string, auditService *AuditService) *DocumentService {
	return &DocumentService{
		repo:         repo,
		appDir:       appDir,
		auditService: auditService,
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
func (s *DocumentService) SaveDocumentBase64(token string, patientID, name, description, docType, contentType, base64Data string) (*domain.Document, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

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

	doc, err := s.saveDocumentBytes(patientID, name, description, docType, contentType, data)
	if err == nil && doc.PatientID != nil {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionCreate, *doc.PatientID, "document", "Created document")
	}
	return doc, err
}

func cleanPatientID(patientID string) (string, error) {
	if filepath.IsAbs(patientID) || strings.HasPrefix(patientID, "/") || strings.HasPrefix(patientID, "\\") || strings.Contains(patientID, ":") {
		return "", errors.New("invalid patient ID: path traversal or absolute path detected")
	}

	rawSegments := strings.FieldsFunc(patientID, func(r rune) bool {
		return r == '/' || r == '\\'
	})

	if len(rawSegments) == 0 {
		return "", errors.New("invalid patient ID: empty")
	}

	var cleanedSegments []string
	for _, seg := range rawSegments {
		trimmed := strings.TrimRight(strings.TrimSpace(seg), " .")
		if trimmed == "" {
			if strings.Contains(seg, "..") {
				return "", errors.New("invalid patient ID: path traversal detected")
			}
			// Single dot or dot/space equivalent to '.' -> skip
			continue
		}
		if trimmed == "." || trimmed == ".." {
			return "", errors.New("invalid patient ID: path traversal detected")
		}
		cleanedSegments = append(cleanedSegments, trimmed)
	}

	if len(cleanedSegments) == 0 {
		return "", errors.New("invalid patient ID: path traversal detected")
	}

	cleanID := filepath.Join(cleanedSegments...)
	if filepath.IsAbs(cleanID) {
		return "", errors.New("invalid patient ID: path traversal detected")
	}

	return cleanID, nil
}

// saveDocumentBytes saves a document from a byte array.
func (s *DocumentService) saveDocumentBytes(patientID, name, description, docType, contentType string, data []byte) (*domain.Document, error) {
	docID := uuid.New().String()

	var relDir string
	var targetDir string
	var pID *string

	if patientID != "" {
		cleanID, err := cleanPatientID(patientID)
		if err != nil {
			return nil, err
		}
		basePath := s.getDocumentsBasePath()
		targetDir = s.getPatientDocumentsPath(cleanID)
		rel, err := filepath.Rel(basePath, targetDir)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "..\\") {
			return nil, errors.New("invalid patient ID: path traversal detected")
		}
		relDir = cleanID
		pID = &cleanID
	} else {
		relDir = "clinic"
		targetDir = s.getClinicDocumentsPath()
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create document directory: %w", err)
	}

	// For safety, let's store it purely as docID in the filesystem to prevent traversal attacks.
	filePathRelative := filepath.Join(relDir, docID)
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

// GetDocumentBase64 retrieves a document and returns its contents as a base64 encoded string.
func (s *DocumentService) GetDocumentBase64(token string, id string) (string, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return "", ErrUnauthorized
	}

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

	b64 := base64.StdEncoding.EncodeToString(data)
	if doc.PatientID != nil {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, *doc.PatientID, "document", "Viewed document")
	}
	return b64, nil
}

// GetDocumentImagesBase64 retrieves an image document and returns a slice of data URLs.
// For DICOM files, it extracts all frames. For regular images, it returns a single data URL.
func (s *DocumentService) GetDocumentImagesBase64(token string, id string) ([]string, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	doc, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(s.getDocumentsBasePath(), doc.FilePath)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open document file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read document file: %w", err)
	}

	lowerName := strings.ToLower(doc.Name)
	isDicom := strings.HasSuffix(lowerName, ".dcm") ||
		strings.HasSuffix(lowerName, ".dicom") ||
		strings.Contains(strings.ToLower(doc.ContentType), "dicom")

	if !isDicom && len(data) >= 132 {
		// Magic bytes check for DICOM
		if string(data[128:132]) == "DICM" {
			isDicom = true
		}
	}

	if isDicom {
		dataURLs, err := ParseDicomDataURLs(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DICOM frames: %w", err)
		}

		if doc.PatientID != nil {
			_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, *doc.PatientID, "document", "Viewed DICOM images")
		}
		return dataURLs, nil
	}

	// For standard images, return a single data URL
	b64 := base64.StdEncoding.EncodeToString(data)
	contentType := doc.ContentType
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, b64)
	if doc.PatientID != nil {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, *doc.PatientID, "document", "Viewed document image")
	}
	return []string{dataURL}, nil
}

// OpenDocument opens the document in the OS default application.
func (s *DocumentService) OpenDocument(token string, id string) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}

	doc, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(s.getDocumentsBasePath(), doc.FilePath)

	// Since the internal file might not have an extension, we copy it to a temp file with its original name
	// to ensure the OS opens it with the correct application.
	tempDir := os.TempDir()

	// Clean the file name to prevent traversal issues
	safeName := filepath.Base(doc.Name)
	if safeName == "." || safeName == "/" || safeName == "\\" {
		safeName = "document_" + id
	}

	// Ensure the file has a proper extension so the OS knows how to open it
	ext := filepath.Ext(safeName)
	if ext == "" && doc.ContentType != "" {
		if exts, err := mime.ExtensionsByType(doc.ContentType); err == nil && len(exts) > 0 {
			safeName += exts[0]
		}
	}

	tempFilePath := filepath.Join(tempDir, "libredental_temp_"+id+"_"+safeName)

	input, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open source document: %w", err)
	}
	defer input.Close()

	output, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return fmt.Errorf("failed to copy to temp file: %w", err)
	}

	output.Close() // Ensure file is closed before opening

	// Open the file with the default OS application
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	args = append(args, tempFilePath)

	// #nosec G204 - cmd and args are controlled and safe
	if err := exec.Command(cmd, args...).Start(); err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	if doc.PatientID != nil {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, *doc.PatientID, "document", "Opened document")
	}
	return nil
}

// ExportDocumentToPath exports a document to a specific file path.
func (s *DocumentService) ExportDocumentToPath(token string, id string, destPath string) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}

	doc, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(s.getDocumentsBasePath(), doc.FilePath)

	input, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open source document: %w", err)
	}
	defer input.Close()

	output, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return fmt.Errorf("failed to write to destination file: %w", err)
	}

	if doc.PatientID != nil {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionExport, *doc.PatientID, "document", "Exported document")
	}
	return nil
}

// ListPatientDocuments lists documents for a specific patient using a filter.
func (s *DocumentService) ListPatientDocuments(token string, filter domain.DocumentFilter) ([]domain.Document, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	docs, err := s.repo.ListByFilter(filter)
	if err == nil && filter.PatientID != nil {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionRead, *filter.PatientID, "document", "Listed patient documents")
	}
	return docs, err
}

// ListClinicDocuments lists clinic-wide documents.
func (s *DocumentService) ListClinicDocuments(token string) ([]domain.Document, error) {
	if s.auditService.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}

	return s.repo.ListClinicDocuments()
}

// DeleteDocument deletes a document and its associated file.
func (s *DocumentService) DeleteDocument(token string, id string) error {
	if s.auditService.GetSessionUser(token) == nil {
		return ErrUnauthorized
	}

	doc, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(id); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}

	fullPath := filepath.Join(s.getDocumentsBasePath(), doc.FilePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove document file: %w", err)
	}

	if doc.PatientID != nil {
		_ = s.auditService.LogPatientAction(token, domain.AuditActionDelete, *doc.PatientID, "document", "Deleted document")
	}
	return nil
}
