package domain

import "time"

// ClaimStatus represents the lifecycle of an insurance claim.
type ClaimStatus string

const (
	ClaimStatusDraft     ClaimStatus = "draft"
	ClaimStatusSubmitted ClaimStatus = "submitted"
	ClaimStatusAccepted  ClaimStatus = "accepted"
	ClaimStatusRejected  ClaimStatus = "rejected"
	ClaimStatusPaid      ClaimStatus = "paid"
)

// PaymentMethod represents how a payment was made.
type PaymentMethod string

const (
	PaymentMethodCash       PaymentMethod = "cash"
	PaymentMethodCheck      PaymentMethod = "check"
	PaymentMethodCreditCard PaymentMethod = "credit_card"
	PaymentMethodInsurance  PaymentMethod = "insurance"
	PaymentMethodWriteOff   PaymentMethod = "write_off"
)

// ClaimLineItem represents a single CDT-coded procedure on an insurance claim.
type ClaimLineItem struct {
	ID               string         `json:"id"`
	ToothConditionID string         `json:"tooth_condition_id,omitempty"` // optional link to dental_conditions
	ToothNumber      int            `json:"tooth_number,omitempty"`
	Surfaces         []ToothSurface `json:"surfaces,omitempty"`
	ADACode          string         `json:"ada_code"`
	Description      string         `json:"description"`
	Fee              int64          `json:"fee"`
	InsuranceAllowed int64          `json:"insurance_allowed,omitempty"`
	PatientPortion   int64          `json:"patient_portion,omitempty"`
}

// Claim represents an insurance claim for one or more procedures.
type Claim struct {
	ID               string          `json:"id"`
	PatientID        string          `json:"patient_id"`
	ProviderID       string          `json:"provider_id"`
	AppointmentID    string          `json:"appointment_id,omitempty"`
	InsuranceCarrier string          `json:"insurance_carrier,omitempty"`
	PolicyNumber     string          `json:"policy_number,omitempty"`
	GroupNumber      string          `json:"group_number,omitempty"`
	DateOfService    string          `json:"date_of_service"` // stored as YYYY-MM-DD
	Status           ClaimStatus     `json:"status"`
	Notes            string          `json:"notes,omitempty"`
	LineItems        []ClaimLineItem `json:"line_items"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// TotalFee sums all line item fees on the claim.
func (c *Claim) TotalFee() int64 {
	var total int64
	for _, item := range c.LineItems {
		total += item.Fee
	}
	return total
}

// Payment represents a single payment event (from a patient or insurance).
type Payment struct {
	ID        string        `json:"id"`
	PatientID string        `json:"patient_id"`
	ClaimID   string        `json:"claim_id,omitempty"` // nullable — payment may not be tied to a specific claim
	Amount    int64         `json:"amount"`
	Method    PaymentMethod `json:"method"`
	Date      string        `json:"date"` // stored as YYYY-MM-DD
	Notes     string        `json:"notes,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}

// PatientBalance is a computed summary DTO for a patient's outstanding balance.
type PatientBalance struct {
	PatientID   string  `json:"patient_id"`
	TotalBilled int64   `json:"total_billed"`
	TotalPaid   int64   `json:"total_paid"`
	Outstanding int64   `json:"outstanding"`
}

// BundleItemTemplate is a CDT line item template stored inside a TreatmentBundle.
type BundleItemTemplate struct {
	ADACode     string  `json:"ada_code"`
	Description string  `json:"description"`
	DefaultFee  int64   `json:"default_fee"`
}

// TreatmentBundle is a practice-wide reusable template of CDT procedures.
// It is not tied to any specific patient — dentists configure these once
// and apply them to any patient during claim entry.
type TreatmentBundle struct {
	ID          string               `json:"id"`
	Shortname   string               `json:"shortname"` // e.g. "crwn", "rct-a" — unique mnemonic for fast data entry
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Items       []BundleItemTemplate `json:"items"`
	TotalFee    int64                `json:"total_fee"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// ProcedureCode represents a national standard dental procedure/billing code.
type ProcedureCode struct {
	CountryCode  CountryCode `json:"country_code"`
	Code         string      `json:"code"`
	Category     string      `json:"category"`
	Description  string      `json:"description"`
	DefaultFee   int64       `json:"default_fee"`
	EffectiveFee int64       `json:"effective_fee,omitempty"`
	IsActive     bool        `json:"is_active"`
}

// FeeSchedule represents a practice or provider-specific custom fee override for a procedure code.
type FeeSchedule struct {
	ID          string      `json:"id"`
	CountryCode CountryCode `json:"country_code"`
	Code        string      `json:"code"`
	ProviderID  string      `json:"provider_id,omitempty"`
	CustomFee   int64       `json:"custom_fee"`
	UpdatedAt   time.Time   `json:"updated_at"`
}
