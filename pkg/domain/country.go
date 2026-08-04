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

// CountryConfig contains static compliance and display rules for a country.
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

var supportedCountries = map[CountryCode]CountryConfig{
	CountryUS: {
		Code:                  CountryUS,
		Name:                  "United States",
		NationalIDName:        "Social Security Number (SSN)",
		NationalIDType:        "ssn",
		NationalIDPlaceholder: "000-00-0000",
		DefaultToothSystem:    ToothSystemUniversal,
		DefaultCurrency:       "USD",
		StateProvinceLabel:    "State",
		PostalCodeLabel:       "ZIP Code",
		DateFormat:            "MM/DD/YYYY",
	},
	CountryCA: {
		Code:                  CountryCA,
		Name:                  "Canada",
		NationalIDName:        "Social Insurance Number (SIN)",
		NationalIDType:        "sin",
		NationalIDPlaceholder: "000-000-000",
		DefaultToothSystem:    ToothSystemFDI,
		DefaultCurrency:       "CAD",
		StateProvinceLabel:    "Province",
		PostalCodeLabel:       "Postal Code",
		DateFormat:            "YYYY-MM-DD",
	},
	CountryGB: {
		Code:                  CountryGB,
		Name:                  "United Kingdom",
		NationalIDName:        "NHS Number",
		NationalIDType:        "nhs_number",
		NationalIDPlaceholder: "000 000 0000",
		DefaultToothSystem:    ToothSystemFDI,
		DefaultCurrency:       "GBP",
		StateProvinceLabel:    "County",
		PostalCodeLabel:       "Postcode",
		DateFormat:            "DD/MM/YYYY",
	},
	CountryAU: {
		Code:                  CountryAU,
		Name:                  "Australia",
		NationalIDName:        "Medicare Card Number",
		NationalIDType:        "medicare_num",
		NationalIDPlaceholder: "0000 00000 0",
		DefaultToothSystem:    ToothSystemFDI,
		DefaultCurrency:       "AUD",
		StateProvinceLabel:    "State",
		PostalCodeLabel:       "Postcode",
		DateFormat:            "DD/MM/YYYY",
	},
	CountryDE: {
		Code:                  CountryDE,
		Name:                  "Germany",
		NationalIDName:        "Tax / Health Insurance ID",
		NationalIDType:        "tax_id",
		NationalIDPlaceholder: "X000000000",
		DefaultToothSystem:    ToothSystemFDI,
		DefaultCurrency:       "EUR",
		StateProvinceLabel:    "State / Bundesland",
		PostalCodeLabel:       "PLZ (Postal Code)",
		DateFormat:            "DD.MM.YYYY",
	},
	CountryFR: {
		Code:                  CountryFR,
		Name:                  "France",
		NationalIDName:        "NIR (NIR / Numéro SS)",
		NationalIDType:        "nir",
		NationalIDPlaceholder: "1 00 00 00 000 000 00",
		DefaultToothSystem:    ToothSystemFDI,
		DefaultCurrency:       "EUR",
		StateProvinceLabel:    "Region / Department",
		PostalCodeLabel:       "Code Postal",
		DateFormat:            "DD/MM/YYYY",
	},
}

// GetSupportedCountries returns all configured country profiles.
func GetSupportedCountries() []CountryConfig {
	list := []CountryConfig{
		supportedCountries[CountryUS],
		supportedCountries[CountryCA],
		supportedCountries[CountryGB],
		supportedCountries[CountryAU],
		supportedCountries[CountryDE],
		supportedCountries[CountryFR],
	}
	return list
}

// GetCountryConfig retrieves the static catalog entry for a country code.
func GetCountryConfig(code CountryCode) (CountryConfig, bool) {
	cfg, ok := supportedCountries[code]
	if !ok {
		return CountryConfig{}, false
	}
	return cfg, true
}
