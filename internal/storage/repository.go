package storage

import (
	"context"
	"errors"

	"github.com/LibreDental/libredental/internal/domain"
)

var (
	ErrNotFound     = errors.New("record not found")
	ErrConflict     = errors.New("optimistic concurrency conflict: record has been updated by another user")
	ErrInvalidInput = errors.New("invalid input data")
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

// ClaimRepository defines storage operations for insurance claims.
type ClaimRepository interface {
	Create(ctx context.Context, claim *domain.Claim) error
	GetByID(ctx context.Context, id string) (*domain.Claim, error)
	Update(ctx context.Context, claim *domain.Claim) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, patientID string) ([]*domain.Claim, error)
	// GetTotalBilled returns the sum of all line item fees for a patient's claims.
	GetTotalBilled(ctx context.Context, patientID string) (float64, error)
}

// PaymentRepository defines storage operations for patient payment records.
type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByID(ctx context.Context, id string) (*domain.Payment, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, patientID string) ([]*domain.Payment, error)
	// GetTotalPaid returns the sum of all payments received for a patient.
	GetTotalPaid(ctx context.Context, patientID string) (float64, error)
}

// TreatmentBundleRepository defines storage operations for clinic-wide procedure bundle templates.
type TreatmentBundleRepository interface {
	Create(ctx context.Context, bundle *domain.TreatmentBundle) error
	GetByID(ctx context.Context, id string) (*domain.TreatmentBundle, error)
	GetByShortname(ctx context.Context, shortname string) (*domain.TreatmentBundle, error)
	Update(ctx context.Context, bundle *domain.TreatmentBundle) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*domain.TreatmentBundle, error)
}

// ProcedureCodeRepository defines storage operations for procedure/billing code catalogs.
type ProcedureCodeRepository interface {
	List(ctx context.Context, countryCode domain.CountryCode) ([]*domain.ProcedureCode, error)
	GetByCode(ctx context.Context, countryCode domain.CountryCode, code string) (*domain.ProcedureCode, error)
}

// FeeScheduleRepository defines storage operations for practice and provider fee schedules.
type FeeScheduleRepository interface {
	Save(ctx context.Context, schedule *domain.FeeSchedule) error
	Delete(ctx context.Context, id string) error
	ListFeeSchedules(ctx context.Context, countryCode domain.CountryCode, providerID string) ([]*domain.FeeSchedule, error)
	GetEffectiveFee(ctx context.Context, countryCode domain.CountryCode, code string, providerID string) (float64, error)
}
