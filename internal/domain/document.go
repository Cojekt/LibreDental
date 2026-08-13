package domain

import (
	"time"
)

// DocumentType represents the category of the document.
type DocumentType string

const (
	DocumentTypeXRay        DocumentType = "xray"
	DocumentTypeConsentForm DocumentType = "consent_form"
	DocumentTypePDF         DocumentType = "pdf"
	DocumentTypeImage       DocumentType = "image"
	DocumentTypeOther       DocumentType = "other"
)

// Document represents a file stored in the system (e.g., X-ray, consent form).
type Document struct {
	ID          string       `json:"id"`
	PatientID   *string      `json:"patient_id,omitempty"` // nil means it's a clinic-wide document
	Type        DocumentType `json:"type"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	FilePath    string       `json:"file_path"` // relative to the documents base folder
	SizeBytes   int64        `json:"size_bytes"`
	ContentType string       `json:"content_type"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// DocumentFilter specifies search parameters for documents.
type DocumentFilter struct {
	PatientID *string      `json:"patient_id,omitempty"`
	Type      DocumentType `json:"type,omitempty"`
	Limit     int          `json:"limit,omitempty"`
	Offset    int          `json:"offset,omitempty"`
}
