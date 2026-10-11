package domain

import (
	"strings"
	"time"
)

// System actors are reserved, non-human identities for actions LibreDental takes on its own
// with no staff member logged in, such as sending automatic appointment reminders. Each
// background job gets its own ID so the audit trail shows which process acted. Staff IDs can
// never use the prefix.
const (
	SystemActorPrefix    = "system:"
	SystemActorReminders = "system:reminders"
	SystemActorName      = "LibreDental"
)

// IsSystemActorID reports whether id is a system actor ID.
func IsSystemActorID(id string) bool {
	return len(id) > len(SystemActorPrefix) && strings.HasPrefix(id, SystemActorPrefix)
}

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
	// SystemActor reports whether UserID is a system actor (IsSystemActorID). It's derived
	// when entries are read, not stored, so the frontend doesn't re-implement the rule.
	SystemActor bool `json:"system_actor"`
}
