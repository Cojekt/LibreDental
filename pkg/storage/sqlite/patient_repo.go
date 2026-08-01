package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage"
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

	query := `
	INSERT INTO patients (
		id, first_name, last_name, middle_name, preferred_name,
		date_of_birth, gender, email, phone_primary, phone_secondary,
		address_line1, address_line2, city, state, zip_code,
		medical_alerts, allergies, notes, created_at, updated_at, version
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.FirstName, p.LastName, p.MiddleName, p.PreferredName,
		p.DateOfBirth.Format(time.RFC3339), p.Gender, p.Email, p.PhonePrimary, p.PhoneSecondary,
		p.AddressLine1, p.AddressLine2, p.City, p.State, p.ZipCode,
		string(alertsJSON), string(allergiesJSON), p.Notes, p.CreatedAt, p.UpdatedAt, p.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to insert patient: %w", err)
	}
	return nil
}

func (r *PatientRepository) GetByID(ctx context.Context, id string) (*domain.Patient, error) {
	query := `
	SELECT id, first_name, last_name, middle_name, preferred_name,
	       date_of_birth, gender, email, phone_primary, phone_secondary,
	       address_line1, address_line2, city, state, zip_code,
	       medical_alerts, allergies, notes, created_at, updated_at, version
	FROM patients WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanPatient(row)
}

func (r *PatientRepository) Update(ctx context.Context, p *domain.Patient) error {
	now := time.Now().UTC()
	alertsJSON, _ := json.Marshal(p.MedicalAlerts)
	allergiesJSON, _ := json.Marshal(p.Allergies)

	query := `
	UPDATE patients SET
		first_name = ?, last_name = ?, middle_name = ?, preferred_name = ?,
		date_of_birth = ?, gender = ?, email = ?, phone_primary = ?, phone_secondary = ?,
		address_line1 = ?, address_line2 = ?, city = ?, state = ?, zip_code = ?,
		medical_alerts = ?, allergies = ?, notes = ?, updated_at = ?, version = version + 1
	WHERE id = ? AND version = ?`

	res, err := r.db.ExecContext(ctx, query,
		p.FirstName, p.LastName, p.MiddleName, p.PreferredName,
		p.DateOfBirth.Format(time.RFC3339), p.Gender, p.Email, p.PhonePrimary, p.PhoneSecondary,
		p.AddressLine1, p.AddressLine2, p.City, p.State, p.ZipCode,
		string(alertsJSON), string(allergiesJSON), p.Notes, now,
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
		conditions = append(conditions, "(LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR phone_primary LIKE ?)")
		args = append(args, q, q, q, q)
	}

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
	       date_of_birth, gender, email, phone_primary, phone_secondary,
	       address_line1, address_line2, city, state, zip_code,
	       medical_alerts, allergies, notes, created_at, updated_at, version
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

	return patients, total, nil
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanPatient(scanner scannable) (*domain.Patient, error) {
	var p domain.Patient
	var dobStr, alertsJSON, allergiesJSON string
	var genderStr string

	err := scanner.Scan(
		&p.ID, &p.FirstName, &p.LastName, &p.MiddleName, &p.PreferredName,
		&dobStr, &genderStr, &p.Email, &p.PhonePrimary, &p.PhoneSecondary,
		&p.AddressLine1, &p.AddressLine2, &p.City, &p.State, &p.ZipCode,
		&alertsJSON, &allergiesJSON, &p.Notes, &p.CreatedAt, &p.UpdatedAt, &p.Version,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}

	p.Gender = domain.Gender(genderStr)
	p.DateOfBirth, _ = time.Parse(time.RFC3339, dobStr)
	json.Unmarshal([]byte(alertsJSON), &p.MedicalAlerts)
	json.Unmarshal([]byte(allergiesJSON), &p.Allergies)

	return &p, nil
}
