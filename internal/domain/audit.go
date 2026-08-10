package domain

import (
	"time"
)

// AuditAction represents the type of operation performed on ePHI.
type AuditAction string

const (
	AuditActionCreate AuditAction = "CREATE"
	AuditActionRead   AuditAction = "READ"
	AuditActionUpdate AuditAction = "UPDATE"
	AuditActionDelete AuditAction = "DELETE"
	AuditActionExport AuditAction = "EXPORT"
)

// AuditLogEntry represents an immutable HIPAA audit trail record.
type AuditLogEntry struct {
	ID         string      `json:"id"`
	Timestamp  time.Time   `json:"timestamp"`
	UserID     string      `json:"user_id"`
	UserName   string      `json:"user_name"`
	PatientID  string      `json:"patient_id,omitempty"`
	Action     AuditAction `json:"action"`
	Resource   string      `json:"resource"` // e.g. "patient_demographics", "xray_image", "dental_chart"
	ResourceID string      `json:"resource_id,omitempty"`
	Details    string      `json:"details,omitempty"`
	IPAddress  string      `json:"ip_address,omitempty"`
}
