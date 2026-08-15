package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type PracticeConfigRepository struct {
	db *DB
}

func NewPracticeConfigRepository(db *DB) *PracticeConfigRepository {
	return &PracticeConfigRepository{db: db}
}

func (r *PracticeConfigRepository) Get(ctx context.Context) (*domain.PracticeConfig, error) {
	query := `
	SELECT id, clinic_name, tagline, tax_id, license_number, phone, email, website,
	       address_line1, address_line2, city, state_province, postal_code,
	       country_code, currency, tooth_system, date_format, business_hours,
	       created_at, updated_at
	FROM practice_config WHERE id = 1`

	row := r.db.QueryRowContext(ctx, query)

	var cfg domain.PracticeConfig
	var countryStr, toothStr, hoursJSON string

	err := row.Scan(
		&cfg.ID,
		&cfg.ClinicName,
		&cfg.Tagline,
		&cfg.TaxID,
		&cfg.LicenseNumber,
		&cfg.Phone,
		&cfg.Email,
		&cfg.Website,
		&cfg.AddressLine1,
		&cfg.AddressLine2,
		&cfg.City,
		&cfg.StateProvince,
		&cfg.PostalCode,
		&countryStr,
		&cfg.Currency,
		&toothStr,
		&cfg.DateFormat,
		&hoursJSON,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to fetch practice config: %w", err)
	}

	cfg.CountryCode = domain.CountryCode(countryStr)
	cfg.ToothSystem = domain.ToothSystem(toothStr)

	if hoursJSON != "" && hoursJSON != "[]" {
		_ = json.Unmarshal([]byte(hoursJSON), &cfg.BusinessHours)
	}
	if len(cfg.BusinessHours) == 0 {
		cfg.BusinessHours = domain.DefaultBusinessHours()
	}

	return &cfg, nil
}

func (r *PracticeConfigRepository) Save(ctx context.Context, cfg *domain.PracticeConfig) error {
	now := time.Now().UTC()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
	cfg.ID = 1

	if len(cfg.BusinessHours) == 0 {
		cfg.BusinessHours = domain.DefaultBusinessHours()
	}

	hoursJSON, err := json.Marshal(cfg.BusinessHours)
	if err != nil {
		return fmt.Errorf("failed to marshal business hours: %w", err)
	}

	query := `
	INSERT INTO practice_config (
		id, clinic_name, tagline, tax_id, license_number, phone, email, website,
		address_line1, address_line2, city, state_province, postal_code,
		country_code, currency, tooth_system, date_format, business_hours,
		created_at, updated_at
	) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		clinic_name = excluded.clinic_name,
		tagline = excluded.tagline,
		tax_id = excluded.tax_id,
		license_number = excluded.license_number,
		phone = excluded.phone,
		email = excluded.email,
		website = excluded.website,
		address_line1 = excluded.address_line1,
		address_line2 = excluded.address_line2,
		city = excluded.city,
		state_province = excluded.state_province,
		postal_code = excluded.postal_code,
		country_code = excluded.country_code,
		currency = excluded.currency,
		tooth_system = excluded.tooth_system,
		date_format = excluded.date_format,
		business_hours = excluded.business_hours,
		updated_at = excluded.updated_at`

	_, err = r.db.ExecContext(
		ctx, query,
		cfg.ClinicName, cfg.Tagline, cfg.TaxID, cfg.LicenseNumber, cfg.Phone, cfg.Email, cfg.Website,
		cfg.AddressLine1, cfg.AddressLine2, cfg.City, cfg.StateProvince, cfg.PostalCode,
		cfg.CountryCode, cfg.Currency, cfg.ToothSystem, cfg.DateFormat, string(hoursJSON),
		cfg.CreatedAt, cfg.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save practice config: %w", err)
	}

	return nil
}

// Providers CRUD

func (r *PracticeConfigRepository) ListProviders(ctx context.Context) ([]*domain.Provider, error) {
	query := `
	SELECT id, name, role, specialty, license_number, email, phone, color, is_active, hourly_rate, created_at, updated_at
	FROM providers ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	defer rows.Close()

	var providers []*domain.Provider
	for rows.Next() {
		var p domain.Provider
		var roleStr string
		var isActiveInt int

		err := rows.Scan(
			&p.ID, &p.Name, &roleStr, &p.Specialty, &p.LicenseNumber,
			&p.Email, &p.Phone, &p.Color, &isActiveInt, &p.HourlyRate, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}

		p.Role = domain.ProviderRole(roleStr)
		p.IsActive = isActiveInt == 1
		providers = append(providers, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating providers: %w", err)
	}

	return providers, nil
}

func (r *PracticeConfigRepository) SaveProvider(ctx context.Context, p *domain.Provider) error {
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	isActiveInt := 0
	if p.IsActive {
		isActiveInt = 1
	}

	query := `
	INSERT INTO providers (
		id, name, role, specialty, license_number, email, phone, color, is_active, hourly_rate, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		role = excluded.role,
		specialty = excluded.specialty,
		license_number = excluded.license_number,
		email = excluded.email,
		phone = excluded.phone,
		color = excluded.color,
		is_active = excluded.is_active,
		hourly_rate = excluded.hourly_rate,
		updated_at = excluded.updated_at`

	_, err := r.db.ExecContext(
		ctx, query,
		p.ID, p.Name, p.Role, p.Specialty, p.LicenseNumber,
		p.Email, p.Phone, p.Color, isActiveInt, p.HourlyRate, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save provider: %w", err)
	}

	return nil
}

func (r *PracticeConfigRepository) DeleteProvider(ctx context.Context, id string) error {
	query := `DELETE FROM providers WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// Operatories CRUD

func (r *PracticeConfigRepository) ListOperatories(ctx context.Context) ([]*domain.Operatory, error) {
	query := `
	SELECT id, name, room_code, type, is_active, created_at, updated_at
	FROM operatories ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list operatories: %w", err)
	}
	defer rows.Close()

	var operatories []*domain.Operatory
	for rows.Next() {
		var op domain.Operatory
		var typeStr string
		var isActiveInt int

		err := rows.Scan(
			&op.ID, &op.Name, &op.RoomCode, &typeStr, &isActiveInt, &op.CreatedAt, &op.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan operatory: %w", err)
		}

		op.Type = domain.OperatoryType(typeStr)
		op.IsActive = isActiveInt == 1
		operatories = append(operatories, &op)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating operatories: %w", err)
	}

	return operatories, nil
}

func (r *PracticeConfigRepository) SaveOperatory(ctx context.Context, op *domain.Operatory) error {
	now := time.Now().UTC()
	if op.CreatedAt.IsZero() {
		op.CreatedAt = now
	}
	op.UpdatedAt = now

	isActiveInt := 0
	if op.IsActive {
		isActiveInt = 1
	}

	query := `
	INSERT INTO operatories (
		id, name, room_code, type, is_active, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		room_code = excluded.room_code,
		type = excluded.type,
		is_active = excluded.is_active,
		updated_at = excluded.updated_at`

	_, err := r.db.ExecContext(
		ctx, query,
		op.ID, op.Name, op.RoomCode, op.Type, isActiveInt, op.CreatedAt, op.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save operatory: %w", err)
	}

	return nil
}

func (r *PracticeConfigRepository) DeleteOperatory(ctx context.Context, id string) error {
	query := `DELETE FROM operatories WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete operatory: %w", err)
	}

	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// CountryConfigs CRUD / Queries

func (r *PracticeConfigRepository) ListCountryConfigs(ctx context.Context) ([]domain.CountryConfig, error) {
	query := `
	SELECT code, name, national_id_name, national_id_type, national_id_placeholder,
	       default_tooth_system, default_currency, state_province_label, postal_code_label, date_format
	FROM country_configs ORDER BY is_default DESC, name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list country configs: %w", err)
	}
	defer rows.Close()

	var configs []domain.CountryConfig
	for rows.Next() {
		var cfg domain.CountryConfig
		var codeStr, toothStr string
		err := rows.Scan(
			&codeStr,
			&cfg.Name,
			&cfg.NationalIDName,
			&cfg.NationalIDType,
			&cfg.NationalIDPlaceholder,
			&toothStr,
			&cfg.DefaultCurrency,
			&cfg.StateProvinceLabel,
			&cfg.PostalCodeLabel,
			&cfg.DateFormat,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country config: %w", err)
		}
		cfg.Code = domain.CountryCode(codeStr)
		cfg.DefaultToothSystem = domain.ToothSystem(toothStr)
		configs = append(configs, cfg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating country configs: %w", err)
	}

	return configs, nil
}

func (r *PracticeConfigRepository) GetCountryConfig(ctx context.Context, code string) (*domain.CountryConfig, error) {
	query := `
	SELECT code, name, national_id_name, national_id_type, national_id_placeholder,
	       default_tooth_system, default_currency, state_province_label, postal_code_label, date_format
	FROM country_configs WHERE code = ?`

	row := r.db.QueryRowContext(ctx, query, code)

	var cfg domain.CountryConfig
	var codeStr, toothStr string
	err := row.Scan(
		&codeStr,
		&cfg.Name,
		&cfg.NationalIDName,
		&cfg.NationalIDType,
		&cfg.NationalIDPlaceholder,
		&toothStr,
		&cfg.DefaultCurrency,
		&cfg.StateProvinceLabel,
		&cfg.PostalCodeLabel,
		&cfg.DateFormat,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to fetch country config for %s: %w", code, err)
	}
	cfg.Code = domain.CountryCode(codeStr)
	cfg.DefaultToothSystem = domain.ToothSystem(toothStr)
	return &cfg, nil
}

func (r *PracticeConfigRepository) GetDefaultCountryConfig(ctx context.Context) (*domain.CountryConfig, error) {
	query := `
	SELECT code, name, national_id_name, national_id_type, national_id_placeholder,
	       default_tooth_system, default_currency, state_province_label, postal_code_label, date_format
	FROM country_configs WHERE is_default = 1 LIMIT 1`

	row := r.db.QueryRowContext(ctx, query)

	var cfg domain.CountryConfig
	var codeStr, toothStr string
	err := row.Scan(
		&codeStr,
		&cfg.Name,
		&cfg.NationalIDName,
		&cfg.NationalIDType,
		&cfg.NationalIDPlaceholder,
		&toothStr,
		&cfg.DefaultCurrency,
		&cfg.StateProvinceLabel,
		&cfg.PostalCodeLabel,
		&cfg.DateFormat,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to fetch default country config: %w", err)
	}
	cfg.Code = domain.CountryCode(codeStr)
	cfg.DefaultToothSystem = domain.ToothSystem(toothStr)
	return &cfg, nil
}
