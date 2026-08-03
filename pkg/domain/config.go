package domain

import (
	"time"
)

// PracticeConfig represents the system-wide regional configuration for a LibreDental practice instance.
type PracticeConfig struct {
	ID          int64       `json:"id"`
	CountryCode CountryCode `json:"country_code"`
	Currency    string      `json:"currency"`
	ToothSystem ToothSystem `json:"tooth_system"`
	DateFormat  string      `json:"date_format"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// NewPracticeConfig creates a new PracticeConfig populated with regional defaults from the specified country code.
func NewPracticeConfig(code CountryCode) *PracticeConfig {
	meta, _ := GetCountryConfig(code)
	now := time.Now().UTC()

	return &PracticeConfig{
		ID:          1,
		CountryCode: meta.Code,
		Currency:    meta.DefaultCurrency,
		ToothSystem: meta.DefaultToothSystem,
		DateFormat:  meta.DateFormat,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
