package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

func TestValidateDentalClaim_AcceptsCompleteClaim(t *testing.T) {
	if err := ValidateDentalClaim(testClaimSubmission()); err != nil {
		t.Fatalf("complete claim rejected: %v", err)
	}
}

func TestValidateDentalClaim_ReportsAllMissingData(t *testing.T) {
	sub := testClaimSubmission()
	sub.Claim.PayerID = ""
	sub.Practice.NPI = "1234567890" // bad check digit
	sub.Practice.PostalCode = "80238"
	sub.Patient.InsuranceIsSubscriber = false
	sub.Claim.LineItems[0].ADACode = "0120"
	sub.Claim.LineItems[1].ToothNumber = 33

	err := ValidateDentalClaim(sub)
	if !errors.Is(err, ErrClaimDataIncomplete) {
		t.Fatalf("want ErrClaimDataIncomplete, got %v", err)
	}
	for _, want := range []string{
		"payer ID", "valid practice NPI", "ZIP+4", "policyholder name",
		"policyholder date of birth", "relationship to the policyholder",
		`valid CDT code on line 1 (got "0120")`, "valid tooth number on line 2",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
}

func TestValidateDentalClaim_PolicyholderAddressAllOrNothing(t *testing.T) {
	sub := testClaimSubmission()
	sub.Patient.InsuranceIsSubscriber = false
	sub.Patient.InsuranceSubscriberFirstName = "Mary"
	sub.Patient.InsuranceSubscriberLastName = "Doe"
	sub.Patient.InsuranceSubscriberDOB = "1983-02-01"
	sub.Patient.InsuranceSubscriberRelationship = domain.SubscriberRelationshipSpouse
	if err := ValidateDentalClaim(sub); err != nil {
		t.Fatalf("policyholder address is optional for dependents: %v", err)
	}

	sub.Patient.InsuranceSubscriberCity = "Phoenix"
	if err := ValidateDentalClaim(sub); err == nil || !strings.Contains(err.Error(), "complete policyholder address") {
		t.Errorf("partial policyholder address should be rejected, got %v", err)
	}
}

func TestValidateDentalClaim_RenderingProvider(t *testing.T) {
	sub := testClaimSubmission()
	sub.RenderingProvider.TaxonomyCode = ""
	if err := ValidateDentalClaim(sub); err == nil || !strings.Contains(err.Error(), "taxonomy code for provider") {
		t.Errorf("a separately reported rendering provider needs a taxonomy code, got %v", err)
	}

	sub.RenderingProvider.NPI = sub.Practice.NPI
	if err := ValidateDentalClaim(sub); err != nil {
		t.Errorf("taxonomy isn't needed when the rendering NPI is the billing NPI: %v", err)
	}
	if separateRenderingProvider(sub) != nil {
		t.Error("rendering provider sharing the billing NPI should not be reported separately")
	}
}

func TestValidateDentalClaim_RejectsMalformedValues(t *testing.T) {
	sub := testClaimSubmission()
	sub.Patient.FirstName = "***"    // empty once X12 delimiters are stripped
	sub.Practice.TaxID = "12a456789" // a typo must not be dropped into a different valid EIN
	sub.Patient.PostalCode = "K1A 0B1"
	sub.Patient.DateOfBirth = time.Now().AddDate(0, 0, 2)
	sub.Patient.InsuranceIsSubscriber = false
	sub.Patient.InsuranceSubscriberFirstName = "Mary"
	sub.Patient.InsuranceSubscriberLastName = "Doe"
	sub.Patient.InsuranceSubscriberDOB = "0001-01-01"
	sub.RenderingProvider.Name = "Dr."
	sub.Patient.InsuranceSubscriberRelationship = domain.SubscriberRelationshipSpouse

	err := ValidateDentalClaim(sub)
	for _, want := range []string{
		"patient name", "practice tax ID", "5- or 9-digit ZIP", "valid patient date of birth",
		"policyholder date of birth", "name for the provider on the claim",
	} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
}
