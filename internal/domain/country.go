package domain

// CountryCode represents an ISO 3166-1 alpha-2 country code.
type CountryCode string

const (
	CountryUS CountryCode = "US"
	CountryCA CountryCode = "CA"
	CountryGB CountryCode = "GB"
	CountryAU CountryCode = "AU"
	CountryDE CountryCode = "DE"
	CountryFR CountryCode = "FR"
)

// ToothSystem represents the dental tooth numbering notation standard.
type ToothSystem string

const (
	ToothSystemUniversal ToothSystem = "universal" // US standard 1-32 / A-T
	ToothSystemFDI       ToothSystem = "fdi"       // ISO 3950 / International two-digit 11-48 / 51-85
	ToothSystemPalmer    ToothSystem = "palmer"    // UK quadrant 1-8
)

// CountryConfig contains compliance and display rules for a country stored in SQL database.
type CountryConfig struct {
	Code                  CountryCode `json:"code"`
	Name                  string      `json:"name"`
	NationalIDName        string      `json:"national_id_name"`
	NationalIDType        string      `json:"national_id_type"`
	NationalIDPlaceholder string      `json:"national_id_placeholder"`
	DefaultToothSystem    ToothSystem `json:"default_tooth_system"`
	DefaultCurrency       string      `json:"default_currency"`
	StateProvinceLabel    string      `json:"state_province_label"`
	PostalCodeLabel       string      `json:"postal_code_label"`
	DateFormat            string      `json:"date_format"`
}
