package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

type PatientRepository struct {
	db *DB
}

func NewPatientRepository(db *DB) *PatientRepository {
	return &PatientRepository{db: db}
}

func (r *PatientRepository) Create(ctx context.Context, p *domain.Patient) error {
	if p.ID == "" {
		return fmt.Errorf("%w: patient ID is required", storage.ErrInvalidInput)
	}

	alertsJSON, _ := json.Marshal(p.MedicalAlerts)
	allergiesJSON, _ := json.Marshal(p.Allergies)
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.Version == 0 {
		p.Version = 1
	}
	if p.Status == "" {
		p.Status = domain.StatusActive
	}

	query := `
	INSERT INTO patients (
		id, first_name, last_name, middle_name, preferred_name,
		date_of_birth, sex, email, phone_primary, phone_secondary,
		emergency_contact_name, emergency_contact_rel, emergency_contact_phone,
		guarantor_name, guarantor_rel, guarantor_phone,
		insurance_carrier, insurance_policy_number, insurance_group_number,
		preferred_contact_method, preferred_language, reminder_opt_in,
		preferred_provider_id, referral_source,
		address_line1, address_line2, city, state_province, postal_code, country_code,
		national_id_type, national_id,
		medical_alerts, allergies, notes, created_at, updated_at, version, status
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	reminderInt := 1
	if !p.ReminderOptIn && p.ID != "" {
		// handle boolean bool
		if p.ReminderOptIn {
			reminderInt = 1
		} else {
			reminderInt = 0
		}
	}

	_, err := r.db.ExecContext(
		ctx, query,
		p.ID, p.FirstName, p.LastName, p.MiddleName, p.PreferredName,
		p.DateOfBirth.Format(time.RFC3339), p.Sex, p.Email, p.PhonePrimary, p.PhoneSecondary,
		p.EmergencyContactName, p.EmergencyContactRel, p.EmergencyContactPhone,
		p.GuarantorName, p.GuarantorRel, p.GuarantorPhone,
		p.InsuranceCarrier, p.InsurancePolicyNumber, p.InsuranceGroupNumber,
		p.PreferredContactMethod, p.PreferredLanguage, reminderInt,
		p.PreferredProviderID, p.ReferralSource,
		p.AddressLine1, p.AddressLine2, p.City, p.StateProvince, p.PostalCode, p.CountryCode,
		p.NationalIDType, p.NationalID,
		string(alertsJSON), string(allergiesJSON), p.Notes, p.CreatedAt, p.UpdatedAt, p.Version, p.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to insert patient: %w", err)
	}
	return nil
}

func (r *PatientRepository) GetByID(ctx context.Context, id string) (*domain.Patient, error) {
	query := `
	SELECT id, first_name, last_name, middle_name, preferred_name,
	       date_of_birth, sex, email, phone_primary, phone_secondary,
	       emergency_contact_name, emergency_contact_rel, emergency_contact_phone,
	       guarantor_name, guarantor_rel, guarantor_phone,
	       insurance_carrier, insurance_policy_number, insurance_group_number,
	       preferred_contact_method, preferred_language, reminder_opt_in,
	       preferred_provider_id, referral_source,
	       address_line1, address_line2, city, state_province, postal_code, country_code,
	       national_id_type, national_id,
	       medical_alerts, allergies, notes, created_at, updated_at, version, status
	FROM patients WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanPatient(row)
}

func (r *PatientRepository) Update(ctx context.Context, p *domain.Patient) error {
	now := time.Now().UTC()
	alertsJSON, _ := json.Marshal(p.MedicalAlerts)
	allergiesJSON, _ := json.Marshal(p.Allergies)

	reminderInt := 0
	if p.ReminderOptIn {
		reminderInt = 1
	}

	query := `
	UPDATE patients SET
		first_name = ?, last_name = ?, middle_name = ?, preferred_name = ?,
		date_of_birth = ?, sex = ?, email = ?, phone_primary = ?, phone_secondary = ?,
		emergency_contact_name = ?, emergency_contact_rel = ?, emergency_contact_phone = ?,
		guarantor_name = ?, guarantor_rel = ?, guarantor_phone = ?,
		insurance_carrier = ?, insurance_policy_number = ?, insurance_group_number = ?,
		preferred_contact_method = ?, preferred_language = ?, reminder_opt_in = ?,
		preferred_provider_id = ?, referral_source = ?,
		address_line1 = ?, address_line2 = ?, city = ?, state_province = ?, postal_code = ?, country_code = ?,
		national_id_type = ?, national_id = ?,
		medical_alerts = ?, allergies = ?, notes = ?, updated_at = ?, version = version + 1, status = ?
	WHERE id = ? AND version = ?`

	res, err := r.db.ExecContext(
		ctx, query,
		p.FirstName, p.LastName, p.MiddleName, p.PreferredName,
		p.DateOfBirth.Format(time.RFC3339), p.Sex, p.Email, p.PhonePrimary, p.PhoneSecondary,
		p.EmergencyContactName, p.EmergencyContactRel, p.EmergencyContactPhone,
		p.GuarantorName, p.GuarantorRel, p.GuarantorPhone,
		p.InsuranceCarrier, p.InsurancePolicyNumber, p.InsuranceGroupNumber,
		p.PreferredContactMethod, p.PreferredLanguage, reminderInt,
		p.PreferredProviderID, p.ReferralSource,
		p.AddressLine1, p.AddressLine2, p.City, p.StateProvince, p.PostalCode, p.CountryCode,
		p.NationalIDType, p.NationalID,
		string(alertsJSON), string(allergiesJSON), p.Notes, now, p.Status,
		p.ID, p.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to update patient: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return storage.ErrConflict
	}

	p.Version++
	p.UpdatedAt = now
	return nil
}

func (r *PatientRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM patients WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *PatientRepository) List(ctx context.Context, filter domain.PatientFilter) ([]*domain.Patient, int64, error) {
	var conditions []string
	var args []interface{}

	if filter.Query != "" {
		q := "%" + strings.ToLower(filter.Query) + "%"
		conditions = append(conditions, "(LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR phone_primary LIKE ? OR national_id LIKE ?)")
		args = append(args, q, q, q, q, q)
	}

	status := filter.Status
	if status == "" {
		status = string(domain.StatusActive)
	}
	conditions = append(conditions, "status = ?")
	args = append(args, status)

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM patients %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	selectQuery := fmt.Sprintf(`
	SELECT id, first_name, last_name, middle_name, preferred_name,
	       date_of_birth, sex, email, phone_primary, phone_secondary,
	       emergency_contact_name, emergency_contact_rel, emergency_contact_phone,
	       guarantor_name, guarantor_rel, guarantor_phone,
	       insurance_carrier, insurance_policy_number, insurance_group_number,
	       preferred_contact_method, preferred_language, reminder_opt_in,
	       preferred_provider_id, referral_source,
	       address_line1, address_line2, city, state_province, postal_code, country_code,
	       national_id_type, national_id,
	       medical_alerts, allergies, notes, created_at, updated_at, version, status
	FROM patients %s ORDER BY last_name, first_name LIMIT ? OFFSET ?`, whereClause)

	args = append(args, limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var patients []*domain.Patient
	for rows.Next() {
		p, err := scanPatient(rows)
		if err != nil {
			return nil, 0, err
		}
		patients = append(patients, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return patients, total, nil
}

func scanPatient(scanner rowScanner) (*domain.Patient, error) {
	var p domain.Patient
	var dobStr, alertsJSON, allergiesJSON string
	var sexStr, statusStr, countryStr string
	var reminderInt int

	err := scanner.Scan(
		&p.ID, &p.FirstName, &p.LastName, &p.MiddleName, &p.PreferredName,
		&dobStr, &sexStr, &p.Email, &p.PhonePrimary, &p.PhoneSecondary,
		&p.EmergencyContactName, &p.EmergencyContactRel, &p.EmergencyContactPhone,
		&p.GuarantorName, &p.GuarantorRel, &p.GuarantorPhone,
		&p.InsuranceCarrier, &p.InsurancePolicyNumber, &p.InsuranceGroupNumber,
		&p.PreferredContactMethod, &p.PreferredLanguage, &reminderInt,
		&p.PreferredProviderID, &p.ReferralSource,
		&p.AddressLine1, &p.AddressLine2, &p.City, &p.StateProvince, &p.PostalCode, &countryStr,
		&p.NationalIDType, &p.NationalID,
		&alertsJSON, &allergiesJSON, &p.Notes, &p.CreatedAt, &p.UpdatedAt, &p.Version, &statusStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}

	p.Sex = domain.Sex(sexStr)
	p.Status = domain.Status(statusStr)
	p.CountryCode = domain.CountryCode(countryStr)
	p.ReminderOptIn = reminderInt != 0
	p.DateOfBirth, _ = time.Parse(time.RFC3339, dobStr)
	json.Unmarshal([]byte(alertsJSON), &p.MedicalAlerts)
	json.Unmarshal([]byte(allergiesJSON), &p.Allergies)

	return &p, nil
}
