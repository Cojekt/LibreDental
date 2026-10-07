package services

import (
	"fmt"
	"strings"

	"github.com/nyaruka/phonenumbers"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// toE164 converts a phone number as staff typed it into E.164 (+ and country code), which
// messaging vendors require. Numbers without a country code are read as numbers in the
// practice's country. Anything that isn't a valid number is rejected rather than guessed at, so
// a message never goes to a mistyped number.
func toE164(raw string, country domain.CountryCode) (string, error) {
	num, err := parsePhone(raw, country)
	if err != nil {
		return "", err
	}
	return phonenumbers.Format(num, phonenumbers.E164), nil
}

// toSMSNumber is toE164 for text messages: it also rejects numbers that can't receive them.
func toSMSNumber(raw string, country domain.CountryCode) (string, error) {
	num, err := parsePhone(raw, country)
	if err != nil {
		return "", err
	}
	if num.GetExtension() != "" {
		return "", fmt.Errorf("%w: %q has an extension, so it can't receive text messages", storage.ErrInvalidInput, raw)
	}
	// Only some countries can tell landlines from mobiles (not the US or Canada, whose numbers
	// report FIXED_LINE_OR_MOBILE), so this only catches landlines where that's known.
	if phonenumbers.GetNumberType(num) == phonenumbers.FIXED_LINE {
		return "", fmt.Errorf("%w: %q is a landline, so it can't receive text messages", storage.ErrInvalidInput, raw)
	}
	return phonenumbers.Format(num, phonenumbers.E164), nil
}

func parsePhone(raw string, country domain.CountryCode) (*phonenumbers.PhoneNumber, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%w: phone number is empty", storage.ErrInvalidInput)
	}
	num, err := phonenumbers.Parse(raw, string(country))
	if err != nil {
		return nil, fmt.Errorf("%w: phone number %q could not be read", storage.ErrInvalidInput, raw)
	}
	if !phonenumbers.IsValidNumber(num) {
		return nil, fmt.Errorf("%w: %q is not a valid phone number", storage.ErrInvalidInput, raw)
	}
	return num, nil
}
