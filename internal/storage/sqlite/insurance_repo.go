package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/google/uuid"
)

type queryExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type InsuranceRepository struct{ db *DB }

func NewInsuranceRepository(db *DB) *InsuranceRepository { return &InsuranceRepository{db: db} }

func insuranceName(name string) (string, string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if len(name) == 0 || len([]rune(name)) > 160 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", "", fmt.Errorf("%w: name must contain 1–160 characters", storage.ErrInvalidInput)
	}
	return name, strings.ToLower(name), nil
}
func (r *InsuranceRepository) ListProviders(ctx context.Context) ([]*domain.InsuranceProvider, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,archived FROM insurance_providers ORDER BY normalized_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []*domain.InsuranceProvider{}
	for rows.Next() {
		p := new(domain.InsuranceProvider)
		if err := rows.Scan(&p.ID, &p.Name, &p.Archived); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}
func (r *InsuranceRepository) SaveProvider(ctx context.Context, p *domain.InsuranceProvider) error {
	if p == nil {
		return storage.ErrInvalidInput
	}
	name, normalized, err := insuranceName(p.Name)
	if err != nil {
		return err
	}
	p.Name = name
	now := time.Now().UTC()
	if p.ID == "" {
		p.ID = uuid.NewString()
		_, err = r.db.ExecContext(ctx, `INSERT INTO insurance_providers(id,name,normalized_name,archived,created_at,updated_at) VALUES(?,?,?,?,?,?)`, p.ID, name, normalized, p.Archived, now, now)
	} else {
		var res sql.Result
		res, err = r.db.ExecContext(ctx, `UPDATE insurance_providers SET name=?,normalized_name=?,archived=?,updated_at=? WHERE id=?`, name, normalized, p.Archived, now, p.ID)
		if err == nil {
			n, _ := res.RowsAffected()
			if n == 0 {
				return storage.ErrNotFound
			}
		}
	}
	return insuranceConflict(err)
}
func insuranceConflict(err error) error {
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return fmt.Errorf("%w: %w", storage.ErrConflict, storage.ErrInsuranceDuplicate)
	}
	return err
}
func (r *InsuranceRepository) ListFees(ctx context.Context, plan string, cc domain.CountryCode) ([]*domain.InsuranceFee, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT provider_id,country_code,code,fee FROM insurance_fee_overrides WHERE provider_id=? AND country_code=? ORDER BY code`, plan, cc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []*domain.InsuranceFee{}
	for rows.Next() {
		f := new(domain.InsuranceFee)
		if err := rows.Scan(&f.ProviderID, &f.CountryCode, &f.Code, &f.Fee); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}
func (r *InsuranceRepository) SaveFee(ctx context.Context, f *domain.InsuranceFee) error {
	if f == nil || f.ProviderID == "" || f.Code == "" || f.CountryCode == "" || !validFee(f.Fee) {
		return storage.ErrInvalidInput
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO insurance_fee_overrides(provider_id,country_code,code,fee,updated_at) SELECT i.id,?,?,?,? FROM insurance_providers i WHERE i.id=? AND i.archived=0 ON CONFLICT(provider_id,country_code,code) DO UPDATE SET fee=excluded.fee,updated_at=excluded.updated_at`, f.CountryCode, f.Code, f.Fee, time.Now().UTC(), f.ProviderID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return storage.ErrInvalidInput
	}
	return nil
}
func validFee(n int64) bool { return n >= 0 && n <= 9007199254740991 }
func (r *InsuranceRepository) DeleteFee(ctx context.Context, plan string, cc domain.CountryCode, code string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM insurance_fee_overrides WHERE provider_id=? AND country_code=? AND code=?`, plan, cc, code)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *InsuranceRepository) Preview(ctx context.Context, s *domain.PricingSource) (*domain.PricingSource, error) {
	return previewPricing(ctx, r.db, s)
}
func previewPricing(ctx context.Context, q queryExecutor, s *domain.PricingSource) (*domain.PricingSource, error) {
	if s == nil || s.PatientID == "" || s.Code == "" || s.CountryCode == "" {
		return nil, storage.ErrInvalidInput
	}
	var exists bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM patients WHERE id=?)`, s.PatientID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, storage.ErrNotFound
	}
	if s.ClinicianID != "" {
		var active bool
		if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM providers WHERE id=? AND is_active=1)`, s.ClinicianID).Scan(&active); err != nil || !active {
			return nil, storage.ErrInvalidInput
		}
	}
	p := &domain.PricingSource{Kind: s.Kind, PatientID: s.PatientID, CountryCode: s.CountryCode, Code: s.Code, ClinicianID: s.ClinicianID}
	err := q.QueryRowContext(ctx, `SELECT default_fee FROM procedure_codes WHERE country_code=? AND code=? AND is_active=1`, s.CountryCode, s.Code).Scan(&p.BaseFee)
	if err != nil && !(errors.Is(err, sql.ErrNoRows) && s.Kind == "manual") {
		return nil, fmt.Errorf("%w: procedure unavailable", storage.ErrInvalidInput)
	}
	p.HasBase = err == nil
	p.StandardFee = p.BaseFee
	version := []any{p.BaseFee}

	if p.HasBase {
		p.StandardFee, err = (&ProcedureRepository{db: q}).GetEffectiveFee(ctx, s.CountryCode, s.Code, s.ClinicianID)
		if err != nil {
			return nil, err
		}
	}
	rows, err := q.QueryContext(ctx, `SELECT id,custom_fee,updated_at FROM fee_schedules WHERE country_code=? AND code=? AND (provider_id IS NULL OR provider_id=?) ORDER BY id`, s.CountryCode, s.Code, s.ClinicianID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, stamp string
		var fee int64
		if err := rows.Scan(&id, &fee, &stamp); err != nil {
			rows.Close()
			return nil, err
		}
		version = append(version, id, stamp, fee)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	switch s.Kind {
	case "standard":
		p.Amount = p.StandardFee
	case "manual":
		if !validFee(s.Amount) {
			return nil, storage.ErrInvalidInput
		}
		p.Amount = s.Amount
	case "insurance":
		var archived bool
		var providerStamp, feeStamp string
		err := q.QueryRowContext(ctx, `SELECT id,name,archived,updated_at FROM insurance_providers WHERE id=?`, s.ProviderID).Scan(&p.ProviderID, &p.ProviderName, &archived, &providerStamp)
		if err != nil || archived {
			return nil, fmt.Errorf("%w: %w", storage.ErrInvalidInput, storage.ErrInsuranceUnavailable)
		}
		err = q.QueryRowContext(ctx, `SELECT fee,updated_at FROM insurance_fee_overrides WHERE provider_id=? AND country_code=? AND code=?`, s.ProviderID, s.CountryCode, s.Code).Scan(&p.Amount, &feeStamp)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", storage.ErrNotFound, storage.ErrInsuranceFeeMissing)
		}
		if err != nil {
			return nil, err
		}
		version = append(version, p.ProviderID, p.ProviderName, providerStamp, feeStamp, p.Amount)
	default:
		return nil, storage.ErrInvalidInput
	}
	b, _ := json.Marshal([]any{p, version})
	h := sha256.Sum256(b)
	p.Version = hex.EncodeToString(h[:])
	return p, nil
}
func checkPricing(ctx context.Context, q queryExecutor, s *domain.PricingSource, patient, code string, amount int64) (*domain.PricingSource, error) {
	if s == nil || s.PatientID != patient || s.Code != code || s.Amount != amount || s.Version == "" {
		return nil, fmt.Errorf("%w: confirm pricing before saving", storage.ErrInvalidInput)
	}
	current, err := previewPricing(ctx, q, s)
	if err != nil {
		return nil, err
	}
	if current.Version != s.Version {
		return nil, fmt.Errorf("%w: %w", storage.ErrConflict, storage.ErrPricingChanged)
	}
	// Version covers the recomputed amount, not the client's copy, so a saved fee must match it.
	if current.Amount != amount {
		return nil, fmt.Errorf("%w: confirm pricing before saving", storage.ErrInvalidInput)
	}
	current.ConfirmedAt = time.Now().UTC()

	return current, nil
}
func (r *InsuranceRepository) ConfirmCondition(ctx context.Context, c *domain.ToothCondition) (bool, error) {
	if c == nil || c.ID == "" || c.PatientID == "" || c.ToothNumber <= 0 {
		return false, storage.ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	// Acquire the write lock before reading the rule, across desktop and server processes.
	if _, err = tx.ExecContext(ctx, `UPDATE dental_conditions SET id=id WHERE id=?`, c.ID); err != nil {
		return false, err
	}
	var patient, code, description, status, surfaces string
	var fee int64
	var raw sql.NullString
	var updated time.Time
	var created time.Time
	var tooth int
	billable := c.Status == domain.ToothStatusTreatmentPlanned || c.Status == domain.ToothStatusCompleted
	err = tx.QueryRowContext(ctx, `SELECT patient_id,COALESCE(ada_code,''),fee,pricing_source,updated_at,created_at,tooth_number,COALESCE(description,''),status,COALESCE(surfaces,'[]') FROM dental_conditions WHERE id=?`, c.ID).Scan(&patient, &code, &fee, &raw, &updated, &created, &tooth, &description, &status, &surfaces)
	if err == nil {
		if patient != c.PatientID {
			return false, storage.ErrConflict
		}
		if c.ExpectedUpdatedAt.IsZero() {
			var oldSurfaces []domain.ToothSurface
			_ = json.Unmarshal([]byte(surfaces), &oldSurfaces)
			a, _ := json.Marshal(oldSurfaces)
			b, _ := json.Marshal(c.Surfaces)
			if code != c.ADACode || fee != c.Fee || tooth != c.ToothNumber || description != c.Description || status != string(c.Status) || string(a) != string(b) {
				return false, storage.ErrConflict
			}
			if raw.Valid {
				var previous domain.PricingSource
				if err := json.Unmarshal([]byte(raw.String), &previous); err != nil {
					return false, err
				}
				if c.Pricing != nil && c.Pricing.Version != previous.Version {
					return false, storage.ErrConflict
				}
				c.Pricing = &previous
			} else if c.Pricing != nil {
				return false, storage.ErrConflict
			}
			c.CreatedAt, c.UpdatedAt = created, updated
			return false, tx.Commit()
		} else if !updated.Equal(c.ExpectedUpdatedAt) {
			return false, storage.ErrConflict
		}
		c.CreatedAt = created
		if raw.Valid {
			var old domain.PricingSource
			if e := json.Unmarshal([]byte(raw.String), &old); e != nil {
				return false, e
			}
			if !billable {
				c.Pricing = &old
			} else if c.Pricing != nil && c.Pricing.Version == old.Version && code == c.ADACode && fee == c.Fee {
				if c.ToothNumber == 0 {
					return false, storage.ErrInvalidInput
				}
				c.Pricing = &old
			} else if c.Pricing != nil && !c.Pricing.ConfirmedAt.IsZero() {
				if old.Version != c.Pricing.Version || old.ConfirmedAt != c.Pricing.ConfirmedAt || code != c.ADACode || fee != c.Fee {
					return false, storage.ErrConflict
				}
				c.Pricing = &old
			} else if code == c.ADACode && fee == c.Fee && c.Pricing == nil {
				c.Pricing = &old
			} else {
				c.Pricing, err = checkPricing(ctx, tx, c.Pricing, c.PatientID, c.ADACode, c.Fee)
				if err == nil {
					retainInitialPricing(c.Pricing, &old)
				}
			}
		} else if billable && (code != c.ADACode || fee != c.Fee || c.Pricing != nil || (status != string(domain.ToothStatusTreatmentPlanned) && status != string(domain.ToothStatusCompleted))) {
			c.Pricing, err = checkPricing(ctx, tx, c.Pricing, c.PatientID, c.ADACode, c.Fee)
			if err == nil && (status == string(domain.ToothStatusTreatmentPlanned) || status == string(domain.ToothStatusCompleted)) {
				retainInitialPricing(c.Pricing, nil)
			}
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		if billable {
			c.Pricing, err = checkPricing(ctx, tx, c.Pricing, c.PatientID, c.ADACode, c.Fee)
		} else {
			c.Pricing = nil
			err = nil
		}
	}
	if err != nil {
		return false, err
	}
	inserted, err := saveConditionOn(ctx, tx, c)
	if err != nil {
		return false, err
	}
	return inserted, tx.Commit()
}
func (r *InsuranceRepository) ConfirmClaim(ctx context.Context, c *domain.Claim, update bool) error {
	if c == nil || c.ID == "" || c.PatientID == "" {
		return storage.ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE claims SET id=id WHERE id=?`, c.ID); err != nil {
		return err
	}
	repo := &ClaimRepository{db: tx}
	old, err := repo.GetByID(ctx, c.ID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	if old != nil && !update {
		normalized := func(items []domain.ClaimLineItem) string {
			b, _ := json.Marshal(items)
			var cp []domain.ClaimLineItem
			_ = json.Unmarshal(b, &cp)
			for i := range cp {
				if cp[i].Pricing != nil {
					cp[i].Pricing.ConfirmedAt = time.Time{}
				}
			}
			out, _ := json.Marshal(cp)
			return string(out)
		}
		a, b := normalized(old.LineItems), normalized(c.LineItems)
		status := c.Status
		if status == "" {
			status = domain.ClaimStatusDraft
		}
		if old.PatientID == c.PatientID && old.ProviderID == c.ProviderID && old.DateOfService == c.DateOfService && old.AppointmentID == c.AppointmentID && old.InsuranceCarrier == c.InsuranceCarrier && old.PolicyNumber == c.PolicyNumber && old.GroupNumber == c.GroupNumber && old.PayerID == c.PayerID && old.Notes == c.Notes && old.Status == status && a == b {
			*c = *old
			return nil
		}
		return storage.ErrConflict
	}
	if update && old == nil {
		return storage.ErrNotFound
	}
	linked := map[string]bool{}
	lineIDs := map[string]bool{}
	for i := range c.LineItems {
		li := &c.LineItems[i]
		if li.ID == "" || lineIDs[li.ID] {
			return storage.ErrInvalidInput
		}
		lineIDs[li.ID] = true
		preserved := false
		if old != nil {
			for _, prev := range old.LineItems {
				// A provider change must not keep a line priced on another clinician's fee schedule.
				clinicianOK := prev.Pricing == nil || prev.Pricing.ClinicianID == "" || prev.Pricing.ClinicianID == c.ProviderID
				if prev.ID == li.ID && old.PatientID == c.PatientID && prev.ADACode == li.ADACode && prev.Fee == li.Fee && prev.ToothConditionID == li.ToothConditionID && clinicianOK {
					li.Pricing = prev.Pricing
					preserved = true
					break
				}
			}
		}
		if preserved {
			continue
		}
		if li.ToothConditionID != "" {
			if linked[li.ToothConditionID] {
				return storage.ErrConflict
			}
			linked[li.ToothConditionID] = true
			var billed bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM claims c,json_each(c.line_items) j WHERE json_extract(j.value,'$.tooth_condition_id')=? AND (c.id!=? OR COALESCE(json_extract(j.value,'$.id'),'')!=?))`, li.ToothConditionID, c.ID, li.ID).Scan(&billed); err != nil {
				return err
			}
			if billed {
				return storage.ErrConflict
			}
			var raw sql.NullString
			var patient, code, status string
			var amount int64
			e := tx.QueryRowContext(ctx, `SELECT patient_id,COALESCE(NULLIF(ada_code,''),'PROC'),fee,pricing_source,status FROM dental_conditions WHERE id=?`, li.ToothConditionID).Scan(&patient, &code, &amount, &raw, &status)
			if e != nil || patient != c.PatientID || code != li.ADACode || amount != li.Fee || (status != string(domain.ToothStatusTreatmentPlanned) && status != string(domain.ToothStatusCompleted)) {
				return storage.ErrInvalidInput
			}
			li.Pricing = nil
			if raw.Valid {
				if e := json.Unmarshal([]byte(raw.String), &li.Pricing); e != nil {
					return e
				}
				// An empty clinician means no clinician-specific fee was considered.
				if li.Pricing.ClinicianID != "" && li.Pricing.ClinicianID != c.ProviderID {
					return fmt.Errorf("%w: %w", storage.ErrInvalidInput, storage.ErrPricingClinician)
				}
			}
			continue
		}
		if li.Pricing != nil && li.Pricing.ClinicianID != c.ProviderID {
			return storage.ErrInvalidInput
		}
		li.Pricing, err = checkPricing(ctx, tx, li.Pricing, c.PatientID, li.ADACode, li.Fee)
		if err != nil {
			return err
		}
	}
	if update {
		err = repo.Update(ctx, c)
	} else {
		err = repo.Create(ctx, c)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func retainInitialPricing(next, previous *domain.PricingSource) {
	if next == nil {
		return
	}
	next.InitialSelection = &domain.PricingSource{}
	if previous != nil {
		if previous.InitialSelection != nil {
			next.InitialSelection = previous.InitialSelection
		} else {
			next.InitialSelection = previous
		}
	}
}
