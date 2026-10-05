package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/LibreDental/libredental/internal/domain"
)

// StediClaimProvider submits US dental claims (X12 837D) through Stedi's Dental Claims JSON
// API. Stedi translates the JSON into X12 and routes it to the payer named by the claim's
// payer ID; the payer's 277CA acknowledgment and 835 ERA arrive later, asynchronously.
//
// API reference: https://www.stedi.com/docs/healthcare/api-reference/post-healthcare-dental-claims
//
// Claims are sent as test data (never forwarded to a payer) unless the provider config sets
// test_mode to "false", so a misconfigured install can't bill a payer by accident.
type StediClaimProvider struct {
	baseURL string
	client  *http.Client
}

const (
	stediDefaultBaseURL = "https://healthcare.us.stedi.com/2024-04-01"
	stediMaxResponse    = 1 << 20

	// StediConfigAPIKey and StediConfigTestMode are the keys this provider reads from its
	// SecretsService config.
	StediConfigAPIKey   = "api_key"
	StediConfigTestMode = "test_mode"
)

func NewStediClaimProvider() *StediClaimProvider {
	return &StediClaimProvider{baseURL: stediDefaultBaseURL, client: http.DefaultClient}
}

func (p *StediClaimProvider) Name() string { return "stedi" }

func (p *StediClaimProvider) SupportedCountries() []domain.CountryCode {
	return []domain.CountryCode{domain.CountryUS}
}

func (p *StediClaimProvider) SubmitClaim(ctx context.Context, sub *domain.ClaimSubmission, config map[string]string) (*domain.ClaimSubmissionResult, error) {
	apiKey := config[StediConfigAPIKey]
	if apiKey == "" {
		return nil, fmt.Errorf("stedi API key is not configured")
	}
	req, err := buildStediDentalClaim(sub, config[StediConfigTestMode] != "false")
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode stedi claim: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/dental-claims/submission", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build stedi request: %w", err)
	}
	httpReq.Header.Set("Authorization", apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	// Keyed on the exact payload so a retried request is deduplicated by Stedi, while a
	// corrected resubmission (different payload) goes through as a new claim.
	sum := sha256.Sum256(body)
	httpReq.Header.Set("Idempotency-Key", hex.EncodeToString(sum[:]))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stedi request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, stediMaxResponse))
	if err != nil {
		return nil, fmt.Errorf("failed to read stedi response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var e stediErrorResponse
		_ = json.Unmarshal(raw, &e)
		msg := e.Message
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		if len(e.Errors) > 0 {
			msg += ": " + strings.Join(stediErrorMessages(e.Errors), "; ")
		}
		return nil, fmt.Errorf("stedi rejected the request (HTTP %d): %s", resp.StatusCode, msg)
	}

	var out stediSubmissionResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to decode stedi response: %w", err)
	}
	result := &domain.ClaimSubmissionResult{
		ExternalClaimID: out.ClaimReference.CorrelationID,
		Status:          domain.ClaimStatusSubmitted,
		RawResponse:     raw,
	}
	// A 200 only means Stedi built the X12; claim edit failures come back in errors.
	if len(out.Errors) > 0 || !strings.EqualFold(out.Status, "SUCCESS") {
		result.Status = domain.ClaimStatusRejected
		result.Messages = stediErrorMessages(out.Errors)
		if len(result.Messages) == 0 {
			result.Messages = []string{"clearinghouse returned status " + out.Status}
		}
	}
	return result, nil
}

// ─── Request mapping ─────────────────────────────────────────────────────────

type stediDentalClaimRequest struct {
	UsageIndicator          string                  `json:"usageIndicator,omitempty"`
	TradingPartnerServiceID string                  `json:"tradingPartnerServiceId"`
	TradingPartnerName      string                  `json:"tradingPartnerName,omitempty"`
	Submitter               stediSubmitter          `json:"submitter"`
	Receiver                stediReceiver           `json:"receiver"`
	Subscriber              stediSubscriber         `json:"subscriber"`
	Dependent               *stediDependent         `json:"dependent,omitempty"`
	Billing                 stediBillingProvider    `json:"billing"`
	Rendering               *stediRenderingProvider `json:"rendering,omitempty"`
	ClaimInformation        stediClaimInformation   `json:"claimInformation"`
}

type stediContact struct {
	Name        string `json:"name,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	Email       string `json:"email,omitempty"`
}

type stediAddress struct {
	Address1   string `json:"address1"`
	Address2   string `json:"address2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
}

type stediSubmitter struct {
	OrganizationName   string       `json:"organizationName"`
	ContactInformation stediContact `json:"contactInformation"`
}

type stediReceiver struct {
	OrganizationName string `json:"organizationName"`
}

type stediSubscriber struct {
	PaymentResponsibilityLevelCode string        `json:"paymentResponsibilityLevelCode"`
	MemberID                       string        `json:"memberId"`
	FirstName                      string        `json:"firstName"`
	LastName                       string        `json:"lastName"`
	DateOfBirth                    string        `json:"dateOfBirth"`
	Gender                         string        `json:"gender,omitempty"`
	GroupNumber                    string        `json:"groupNumber,omitempty"`
	Address                        *stediAddress `json:"address,omitempty"`
}

type stediDependent struct {
	FirstName                    string        `json:"firstName"`
	LastName                     string        `json:"lastName"`
	DateOfBirth                  string        `json:"dateOfBirth"`
	Gender                       string        `json:"gender"`
	RelationshipToSubscriberCode string        `json:"relationshipToSubscriberCode"`
	Address                      *stediAddress `json:"address,omitempty"`
}

type stediBillingProvider struct {
	NPI                string       `json:"npi"`
	EmployerID         string       `json:"employerId"`
	OrganizationName   string       `json:"organizationName"`
	Address            stediAddress `json:"address"`
	ContactInformation stediContact `json:"contactInformation"`
}

type stediRenderingProvider struct {
	NPI          string `json:"npi"`
	TaxonomyCode string `json:"taxonomyCode"`
	FirstName    string `json:"firstName,omitempty"`
	LastName     string `json:"lastName"`
}

type stediClaimInformation struct {
	ClaimFilingCode                          string             `json:"claimFilingCode"`
	PatientControlNumber                     string             `json:"patientControlNumber"`
	ClaimChargeAmount                        string             `json:"claimChargeAmount"`
	PlaceOfServiceCode                       string             `json:"placeOfServiceCode"`
	ClaimFrequencyCode                       string             `json:"claimFrequencyCode"`
	SignatureIndicator                       string             `json:"signatureIndicator"`
	PlanParticipationCode                    string             `json:"planParticipationCode"`
	ReleaseInformationCode                   string             `json:"releaseInformationCode"`
	BenefitsAssignmentCertificationIndicator string             `json:"benefitsAssignmentCertificationIndicator"`
	ServiceLines                             []stediServiceLine `json:"serviceLines"`
}

type stediServiceLine struct {
	ServiceDate           string              `json:"serviceDate"`
	ProviderControlNumber string              `json:"providerControlNumber"`
	DentalService         stediDentalService  `json:"dentalService"`
	TeethInformation      []stediToothInfoRow `json:"teethInformation,omitempty"`
}

type stediDentalService struct {
	ProcedureCode        string `json:"procedureCode"`
	LineItemChargeAmount string `json:"lineItemChargeAmount"`
	ProcedureCount       int    `json:"procedureCount"`
}

type stediToothInfoRow struct {
	ToothCode         string   `json:"toothCode"`
	ToothSurfaceCodes []string `json:"toothSurfaceCodes,omitempty"`
}

type stediClaimsError struct {
	Code           string `json:"code"`
	Description    string `json:"description"`
	FollowupAction string `json:"followupAction"`
	Location       string `json:"location"`
	Value          string `json:"value"`
}

type stediSubmissionResponse struct {
	Status         string `json:"status"`
	ClaimReference struct {
		CorrelationID        string `json:"correlationId"`
		PatientControlNumber string `json:"patientControlNumber"`
	} `json:"claimReference"`
	Errors []stediClaimsError `json:"errors"`
}

type stediErrorResponse struct {
	Code    string             `json:"code"`
	Message string             `json:"message"`
	Errors  []stediClaimsError `json:"errors"`
}

func stediErrorMessages(errs []stediClaimsError) []string {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		msg := strings.TrimSpace(strings.Join(nonEmpty(e.Code, e.Description), ": "))
		if e.Location != "" {
			msg += " (" + e.Location + ")"
		}
		if e.FollowupAction != "" {
			msg += " - " + e.FollowupAction
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func nonEmpty(vals ...string) []string {
	var out []string
	for _, v := range vals {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// buildStediDentalClaim validates the submission against the shared 837D requirements, then
// maps it to Stedi's JSON rendering of the 837D.
func buildStediDentalClaim(sub *domain.ClaimSubmission, testMode bool) (*stediDentalClaimRequest, error) {
	if err := ValidateDentalClaim(sub); err != nil {
		return nil, err
	}
	c, pt, pr := sub.Claim, sub.Patient, sub.Practice
	serviceDate, _ := x12Date(c.DateOfService)
	practicePhone := x12Phone(pr.Phone)

	var rendering *stediRenderingProvider
	if rp := separateRenderingProvider(sub); rp != nil {
		first, last := splitProviderName(rp.Name)
		rendering = &stediRenderingProvider{
			NPI:          rp.NPI,
			TaxonomyCode: rp.TaxonomyCode,
			FirstName:    x12Text(first),
			LastName:     x12Text(last),
		}
	}

	patientDOB := pt.DateOfBirth.Format("20060102")
	patientAddr := &stediAddress{
		Address1:   x12Text(pt.AddressLine1),
		Address2:   x12Text(pt.AddressLine2),
		City:       x12Text(pt.City),
		State:      x12Text(pt.StateProvince),
		PostalCode: digitsOnly(pt.PostalCode),
	}

	subscriber := stediSubscriber{
		PaymentResponsibilityLevelCode: "P",
		MemberID:                       x12Text(c.PolicyNumber),
		GroupNumber:                    x12Text(c.GroupNumber),
	}
	var dependent *stediDependent
	if pt.InsuranceIsSubscriber {
		subscriber.FirstName = x12Text(pt.FirstName)
		subscriber.LastName = x12Text(pt.LastName)
		subscriber.DateOfBirth = patientDOB
		subscriber.Gender = x12Gender(pt.Sex)
		subscriber.Address = patientAddr
	} else {
		subscriber.FirstName = x12Text(pt.InsuranceSubscriberFirstName)
		subscriber.LastName = x12Text(pt.InsuranceSubscriberLastName)
		subscriber.DateOfBirth, _ = x12Date(pt.InsuranceSubscriberDOB)
		if pt.InsuranceSubscriberSex != "" {
			subscriber.Gender = x12Gender(pt.InsuranceSubscriberSex)
		}
		if hasSubscriberAddress(pt) {
			subscriber.Address = &stediAddress{
				Address1:   x12Text(pt.InsuranceSubscriberAddressLine1),
				Address2:   x12Text(pt.InsuranceSubscriberAddressLine2),
				City:       x12Text(pt.InsuranceSubscriberCity),
				State:      x12Text(pt.InsuranceSubscriberState),
				PostalCode: digitsOnly(pt.InsuranceSubscriberPostalCode),
			}
		}
		dependent = &stediDependent{
			FirstName:                    x12Text(pt.FirstName),
			LastName:                     x12Text(pt.LastName),
			DateOfBirth:                  patientDOB,
			Gender:                       x12Gender(pt.Sex),
			RelationshipToSubscriberCode: pt.InsuranceSubscriberRelationship,
			Address:                      patientAddr,
		}
	}

	lines := make([]stediServiceLine, 0, len(c.LineItems))
	for _, li := range c.LineItems {
		line := stediServiceLine{
			ServiceDate:           serviceDate,
			ProviderControlNumber: domain.ControlNumber(li.ID),
			DentalService: stediDentalService{
				ProcedureCode:        li.ADACode,
				LineItemChargeAmount: centsToDecimal(li.Fee),
				ProcedureCount:       1,
			},
		}
		if li.ToothNumber != 0 {
			tooth, _ := x12ToothCode(li.ToothNumber)
			row := stediToothInfoRow{ToothCode: tooth}
			for _, s := range li.Surfaces {
				row.ToothSurfaceCodes = append(row.ToothSurfaceCodes, string(s))
			}
			line.TeethInformation = []stediToothInfoRow{row}
		}
		lines = append(lines, line)
	}

	req := &stediDentalClaimRequest{
		TradingPartnerServiceID: x12Text(c.PayerID),
		TradingPartnerName:      x12Text(c.InsuranceCarrier),
		Submitter: stediSubmitter{
			OrganizationName:   x12Text(pr.ClinicName),
			ContactInformation: stediContact{Name: x12Text(pr.ClinicName), PhoneNumber: practicePhone},
		},
		Receiver:   stediReceiver{OrganizationName: x12Text(c.InsuranceCarrier)},
		Subscriber: subscriber,
		Dependent:  dependent,
		Billing: stediBillingProvider{
			NPI:              pr.NPI,
			EmployerID:       digitsOnly(pr.TaxID),
			OrganizationName: x12Text(pr.ClinicName),
			Address: stediAddress{
				Address1:   x12Text(pr.AddressLine1),
				Address2:   x12Text(pr.AddressLine2),
				City:       x12Text(pr.City),
				State:      x12Text(pr.StateProvince),
				PostalCode: digitsOnly(pr.PostalCode),
			},
			ContactInformation: stediContact{Name: x12Text(pr.ClinicName), PhoneNumber: practicePhone},
		},
		Rendering: rendering,
		ClaimInformation: stediClaimInformation{
			ClaimFilingCode:                          "CI", // commercial insurance
			PatientControlNumber:                     c.PatientControlNumber(),
			ClaimChargeAmount:                        centsToDecimal(c.TotalFee()),
			PlaceOfServiceCode:                       "11", // office
			ClaimFrequencyCode:                       "1",  // original claim
			SignatureIndicator:                       "Y",
			PlanParticipationCode:                    "A",
			ReleaseInformationCode:                   "Y",
			BenefitsAssignmentCertificationIndicator: "Y",
			ServiceLines:                             lines,
		},
	}
	if testMode {
		req.UsageIndicator = "T"
	}
	return req, nil
}
