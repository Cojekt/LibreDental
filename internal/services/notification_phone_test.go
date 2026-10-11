package services

import (
	"errors"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

func TestToSMSNumber(t *testing.T) {
	valid := []struct {
		raw     string
		country domain.CountryCode
		want    string
	}{
		{"(202) 555-0123", domain.CountryUS, "+12025550123"},
		{"202.555.0123", domain.CountryUS, "+12025550123"},
		{"1-202-555-0123", domain.CountryUS, "+12025550123"},
		{" +1 202 555 0123 ", domain.CountryGB, "+12025550123"},
		{"011 61 412 345 678", domain.CountryUS, "+61412345678"},
		{"416-555-0199", domain.CountryCA, "+14165550199"},
		{"07400 123456", domain.CountryGB, "+447400123456"},
		{"+44 7400 123456", domain.CountryUS, "+447400123456"},
		{"0412 345 678", domain.CountryAU, "+61412345678"},
		{"0151 23456789", domain.CountryDE, "+4915123456789"},
		{"06 12 34 56 78", domain.CountryFR, "+33612345678"},
		{"00 33 6 12 34 56 78", domain.CountryGB, "+33612345678"},
		{"+12025550123", "", "+12025550123"},
	}
	for _, tt := range valid {
		got, err := toSMSNumber(tt.raw, tt.country)
		if err != nil || got != tt.want {
			t.Errorf("toSMSNumber(%q, %s) = %q, %v; want %q", tt.raw, tt.country, got, err, tt.want)
		}
	}

	invalid := []struct {
		raw     string
		country domain.CountryCode
		why     string
	}{
		{"", domain.CountryUS, "empty"},
		{"call me", domain.CountryUS, "letters"},
		{"555-0123", domain.CountryUS, "too short"},
		{"202 555 0123 4567", domain.CountryUS, "too long"},
		{"+1 555 555 0100", domain.CountryUS, "area code 555 doesn't exist"},
		{"07700 900123", domain.CountryGB, "reserved fictional range"},
		{"202-555-0123 ext 12", domain.CountryUS, "extension"},
		{"020 7946 0018", domain.CountryGB, "landline"},
		{"02 9876 5432", domain.CountryAU, "landline"},
		{"030 1234567", domain.CountryDE, "landline"},
		{"01 23 45 67 89", domain.CountryFR, "landline"},
		{"2025550123", "", "no country to read a national number in"},
	}
	for _, tt := range invalid {
		if got, err := toSMSNumber(tt.raw, tt.country); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("toSMSNumber(%q, %s) = %q, %v; want rejection (%s)", tt.raw, tt.country, got, err, tt.why)
		}
	}
}

// Voice calls can reach landlines and extensions, so only the number itself is checked.
func TestToE164AllowsLandlines(t *testing.T) {
	got, err := toE164("020 7946 0018", domain.CountryGB)
	if err != nil || got != "+442079460018" {
		t.Errorf("toE164(GB landline) = %q, %v; want +442079460018", got, err)
	}
	if _, err := toE164("+1 555 555 0100", domain.CountryUS); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected an invalid number to be rejected for voice too, got %v", err)
	}
}
