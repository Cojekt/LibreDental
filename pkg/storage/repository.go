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

// PracticeConfigRepository defines storage operations for practice configuration, providers, and operatories.
type PracticeConfigRepository interface {
	Get(ctx context.Context) (*domain.PracticeConfig, error)
	Save(ctx context.Context, config *domain.PracticeConfig) error

	ListProviders(ctx context.Context) ([]*domain.Provider, error)
	SaveProvider(ctx context.Context, provider *domain.Provider) error
	DeleteProvider(ctx context.Context, id string) error

	ListOperatories(ctx context.Context) ([]*domain.Operatory, error)
	SaveOperatory(ctx context.Context, operatory *domain.Operatory) error
	DeleteOperatory(ctx context.Context, id string) error

	ListCountryConfigs(ctx context.Context) ([]domain.CountryConfig, error)
	GetCountryConfig(ctx context.Context, code string) (*domain.CountryConfig, error)
	GetDefaultCountryConfig(ctx context.Context) (*domain.CountryConfig, error)
}

// SystemSettingsRepository defines storage operations for local system settings & user preferences.
type SystemSettingsRepository interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key string, value string) error
}

// ChartRepository defines storage operations for dental tooth conditions and patient charts.
type ChartRepository interface {
	GetChart(ctx context.Context, patientID string) (*domain.DentalChart, error)
	SaveCondition(ctx context.Context, condition *domain.ToothCondition) error
	DeleteCondition(ctx context.Context, id string) error
}


