package services

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

// Value formatting shared by every clearinghouse that sends X12 837D claims, whether as
// raw X12 or as a JSON rendering of it.

var (
	cdtPattern      = regexp.MustCompile(`^D\d{4}$`)
	taxonomyPattern = regexp.MustCompile(`^[0-9A-Z]{9}X$`)
)

// x12Delimiters are reserved by the X12 envelope and cannot be escaped.
var x12Delimiters = strings.NewReplacer("~", " ", "*", " ", ":", " ", "^", " ", ">", " ")

func x12Text(s string) string {
	return strings.Join(strings.Fields(x12Delimiters.Replace(s)), " ")
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// x12Phone returns a 10-digit phone number, or "" when s isn't a US number.
func x12Phone(s string) string {
	d := digitsOnly(s)
	if len(d) == 11 && d[0] == '1' {
		d = d[1:]
	}
	if len(d) != 10 {
		return ""
	}
	return d
}

// x12Date converts a stored YYYY-MM-DD date to X12's CCYYMMDD.
func x12Date(isoDate string) (string, error) {
	t, err := time.Parse("2006-01-02", isoDate)
	if err != nil {
		return "", err
	}
	return t.Format("20060102"), nil
}

func x12Gender(s domain.Sex) string {
	switch s {
	case domain.SexMale:
		return "M"
	case domain.SexFemale:
		return "F"
	default:
		return "U"
	}
}

func centsToDecimal(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

// x12ToothCode converts LibreDental's internal Universal numbering (1-32 permanent,
// 101-120 for primary A-T) to the ADA Universal designation X12 expects.
func x12ToothCode(n int) (string, bool) {
	switch {
	case n >= 1 && n <= 32:
		return fmt.Sprint(n), true
	case n >= 101 && n <= 120:
		return string(rune('A' + n - 101)), true
	default:
		return "", false
	}
}

func validSubscriberRelationship(code string) bool {
	switch code {
	case domain.SubscriberRelationshipSpouse, domain.SubscriberRelationshipChild,
		domain.SubscriberRelationshipLifePartner, domain.SubscriberRelationshipOther:
		return true
	}
	return false
}

// validNPI checks the NPI's Luhn check digit (computed with the 80840 card-issuer prefix),
// which catches most typos before a payer rejects the claim for them.
func validNPI(npi string) bool {
	if len(npi) != 10 || digitsOnly(npi) != npi {
		return false
	}
	sum := 24 // the 80840 prefix's contribution to the Luhn sum
	for i := 0; i < 9; i++ {
		d := int(npi[8-i] - '0')
		if i%2 == 0 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return (10-sum%10)%10 == int(npi[9]-'0')
}

// splitProviderName splits a single display name like "Dr. Jane Smith, DDS" into
// first and last name, since providers are stored with one name field.
func splitProviderName(name string) (first, last string) {
	if i := strings.Index(name, ","); i >= 0 {
		name = name[:i]
	}
	fields := strings.Fields(name)
	if len(fields) > 0 && strings.EqualFold(strings.TrimSuffix(fields[0], "."), "dr") {
		fields = fields[1:]
	}
	switch len(fields) {
	case 0:
		return "", ""
	case 1:
		return "", fields[0]
	}
	return strings.Join(fields[:len(fields)-1], " "), fields[len(fields)-1]
}
