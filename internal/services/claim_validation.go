package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

// ErrClaimDataIncomplete is returned before anything is sent when the claim, patient, or
// practice records are missing data the clearinghouse requires.
var ErrClaimDataIncomplete = errors.New("claim is missing data required for electronic submission")

// ValidateDentalClaim checks that a submission has everything a US electronic dental claim
// (X12 837D, 005010X224A2) requires. The requirements come from the X12 standard, not from
// any one clearinghouse, so every ClaimProvider that sends 837D claims should call this
// before mapping the submission to its own format.
//
// Every problem is reported at once so staff can fix them in one pass instead of
// discovering them one failed submission at a time.
func ValidateDentalClaim(sub *domain.ClaimSubmission) error {
	if sub == nil || sub.Claim == nil || sub.Patient == nil || sub.Practice == nil {
		return fmt.Errorf("%w: claim, patient, and practice records are required", ErrClaimDataIncomplete)
	}
	c, pt, pr := sub.Claim, sub.Patient, sub.Practice
	var missing []string
	add := func(format string, args ...any) { missing = append(missing, fmt.Sprintf(format, args...)) }
	// Text is checked after X12 sanitizing: a value made only of delimiters ("***") would
	// otherwise pass here and reach the clearinghouse empty.
	blank := func(vals ...string) bool {
		for _, v := range vals {
			if x12Text(v) == "" {
				return true
			}
		}
		return false
	}
	today := time.Now().Format("2006-01-02")

	// Payer and coverage.
	if blank(c.PayerID) {
		add("payer ID")
	}
	if blank(c.InsuranceCarrier) {
		add("insurance carrier name")
	}
	if blank(c.PolicyNumber) {
		add("subscriber member/policy ID")
	}
	if _, err := x12Date(c.DateOfService); err != nil {
		add("valid date of service")
	}

	// Billing provider (the practice).
	if !validNPI(pr.NPI) {
		add("valid practice NPI")
	}
	if len(digitsOnly(pr.TaxID)) != 9 {
		add("practice tax ID (9-digit EIN)")
	}
	if blank(pr.ClinicName) {
		add("practice name")
	}
	if blank(pr.AddressLine1, pr.City, pr.StateProvince) {
		add("practice street address, city, and state")
	}
	if len(digitsOnly(pr.PostalCode)) != 9 {
		add("practice ZIP+4 (payers require all 9 digits for the billing address)")
	}
	if x12Phone(pr.Phone) == "" {
		add("practice phone number")
	}

	// Rendering provider. Only reported separately when the NPI differs from the practice's,
	// and then X12 requires the taxonomy code too.
	if rp := sub.RenderingProvider; rp != nil {
		if !validNPI(rp.NPI) {
			add("valid NPI for provider %s", rp.Name)
		} else if rp.NPI != pr.NPI && !taxonomyPattern.MatchString(rp.TaxonomyCode) {
			add("taxonomy code for provider %s", rp.Name)
		}
	}

	// Patient and, for dependents, the policyholder.
	if pt.DateOfBirth.IsZero() || pt.DateOfBirth.Format("2006-01-02") > today {
		add("valid patient date of birth")
	}
	if blank(pt.FirstName, pt.LastName) {
		add("patient name")
	}
	if blank(pt.AddressLine1, pt.City, pt.StateProvince) || !validZIP(pt.PostalCode) {
		add("patient address with a 5- or 9-digit ZIP code")
	}
	if !pt.InsuranceIsSubscriber {
		if blank(pt.InsuranceSubscriberFirstName, pt.InsuranceSubscriberLastName) {
			add("policyholder name")
		}
		if _, err := x12Date(pt.InsuranceSubscriberDOB); err != nil || pt.InsuranceSubscriberDOB > today {
			add("policyholder date of birth")
		}
		if !validSubscriberRelationship(pt.InsuranceSubscriberRelationship) {
			add("patient's relationship to the policyholder")
		}
		// The policyholder's address is optional in X12 for dependents, but a partial one
		// would be rejected.
		if hasSubscriberAddress(pt) && (blank(pt.InsuranceSubscriberAddressLine1, pt.InsuranceSubscriberCity,
			pt.InsuranceSubscriberState) || !validZIP(pt.InsuranceSubscriberPostalCode)) {
			add("complete policyholder address (street, city, state, and ZIP), or leave it blank")
		}
	}

	// Procedures.
	if len(c.LineItems) == 0 {
		add("at least one procedure")
	}
	for i, li := range c.LineItems {
		if !cdtPattern.MatchString(li.ADACode) {
			add("valid CDT code on line %d (got %q)", i+1, li.ADACode)
		}
		if li.Fee < 0 {
			add("non-negative fee on line %d", i+1)
		}
		if li.ToothNumber != 0 {
			if _, ok := x12ToothCode(li.ToothNumber); !ok {
				add("valid tooth number on line %d", i+1)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrClaimDataIncomplete, strings.Join(missing, "; "))
	}
	return nil
}

// separateRenderingProvider returns the provider to report as the rendering provider, or nil
// when the billing provider already identifies them (X12 loop 2310B is only sent when the
// rendering NPI differs from the billing NPI).
func separateRenderingProvider(sub *domain.ClaimSubmission) *domain.Provider {
	if rp := sub.RenderingProvider; rp != nil && rp.NPI != sub.Practice.NPI {
		return rp
	}
	return nil
}

func validZIP(zip string) bool {
	d := digitsOnly(zip)
	return len(d) == 5 || len(d) == 9
}

func hasSubscriberAddress(pt *domain.Patient) bool {
	return pt.InsuranceSubscriberAddressLine1 != "" || pt.InsuranceSubscriberAddressLine2 != "" ||
		pt.InsuranceSubscriberCity != "" || pt.InsuranceSubscriberState != "" || pt.InsuranceSubscriberPostalCode != ""
}
