package domain

import "time"

type InsuranceProvider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

type InsuranceFee struct {
	ProviderID  string      `json:"provider_id"`
	CountryCode CountryCode `json:"country_code"`
	Code        string      `json:"code"`
	Fee         int64       `json:"fee"`
}

// PricingSource is a review before saving and a historical snapshot afterwards.
// ConfirmedAt is set only by the backend after a successful pricing validation.
type PricingSource struct {
	InitialSelection *PricingSource `json:"initial_selection,omitempty"`
	HasBase          bool           `json:"has_base"`
	Kind             string         `json:"kind"`
	PatientID        string         `json:"patient_id"`
	CountryCode      CountryCode    `json:"country_code"`
	Code             string         `json:"code"`
	ClinicianID      string         `json:"clinician_id"`
	ProviderID       string         `json:"provider_id"`
	ProviderName     string         `json:"provider_name"`
	Amount           int64          `json:"amount"`
	BaseFee          int64          `json:"base_fee"`
	StandardFee      int64          `json:"standard_fee"`
	Version          string         `json:"version"`
	ConfirmedAt      time.Time      `json:"confirmed_at"`
}
