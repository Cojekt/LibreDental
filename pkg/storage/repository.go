package storage

import (
	"context"
	"errors"

	"github.com/LibreDental/libredental/pkg/domain"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrConflict      = errors.New("optimistic concurrency conflict: record has been updated by another user")
	ErrInvalidInput  = errors.New("invalid input data")
)

// PatientRepository defines storage operations for patient demographic records.
type PatientRepository interface {
	Create(ctx context.Context, patient *domain.Patient) error
	GetByID(ctx context.Context, id string) (*domain.Patient, error)
	Update(ctx context.Context, patient *domain.Patient) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter domain.PatientFilter) ([]*domain.Patient, int64, error)
}

// AppointmentRepository defines storage operations for dental appointments.
type AppointmentRepository interface {
	Create(ctx context.Context, appt *domain.Appointment) error
	GetByID(ctx context.Context, id string) (*domain.Appointment, error)
	Update(ctx context.Context, appt *domain.Appointment) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter domain.AppointmentFilter) ([]*domain.Appointment, error)
}

// AuditRepository defines storage operations for immutable HIPAA audit logs.
type AuditRepository interface {
	Log(ctx context.Context, entry *domain.AuditLogEntry) error
	Query(ctx context.Context, patientID string, limit int, offset int) ([]*domain.AuditLogEntry, error)
}

// ConfigRepository defines storage operations for system practice configuration.
type ConfigRepository interface {
	Get(ctx context.Context) (*domain.PracticeConfig, error)
	Save(ctx context.Context, config *domain.PracticeConfig) error
}
