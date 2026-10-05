package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

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

// ErrClaimDataIncomplete is returned before anything is sent when the claim, patient, or
// practice records are missing data the clearinghouse requires.
var ErrClaimDataIncomplete = errors.New("claim is missing data required for electronic submission")

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

// missingFields collects every problem with a submission so staff can fix them all at once
// instead of discovering them one failed submission at a time.
type missingFields []string

func (m *missingFields) add(format string, args ...any) {
	*m = append(*m, fmt.Sprintf(format, args...))
}

func (m missingFields) err() error {
	if len(m) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrClaimDataIncomplete, strings.Join(m, "; "))
}

func buildStediDentalClaim(sub *domain.ClaimSubmission, testMode bool) (*stediDentalClaimRequest, error) {
	if sub == nil || sub.Claim == nil || sub.Patient == nil || sub.Practice == nil {
		return nil, fmt.Errorf("%w: claim, patient, and practice records are required", ErrClaimDataIncomplete)
	}
	c, pt, pr := sub.Claim, sub.Patient, sub.Practice
	var missing missingFields

	if c.PayerID == "" {
		missing.add("payer ID")
	}
	if c.InsuranceCarrier == "" {
		missing.add("insurance carrier name")
	}
	if c.PolicyNumber == "" {
		missing.add("subscriber member/policy ID")
	}

	serviceDate, err := x12Date(c.DateOfService)
	if err != nil {
		missing.add("valid date of service")
	}

	// Billing provider (the practice).
	if !validNPI(pr.NPI) {
		missing.add("valid practice NPI")
	}
	ein := digitsOnly(pr.TaxID)
	if len(ein) != 9 {
		missing.add("practice tax ID (9-digit EIN)")
	}
	if pr.ClinicName == "" {
		missing.add("practice name")
	}
	if pr.AddressLine1 == "" || pr.City == "" || pr.StateProvince == "" {
		missing.add("practice street address, city, and state")
	}
	billingZip := digitsOnly(pr.PostalCode)
	if len(billingZip) != 9 {
		missing.add("practice ZIP+4 (payers require all 9 digits for the billing address)")
	}
	practicePhone := x12Phone(pr.Phone)
	if practicePhone == "" {
		missing.add("practice phone number")
	}

	// Rendering provider: only sent when it differs from the billing NPI (X12 loop 2310B).
	var rendering *stediRenderingProvider
	if rp := sub.RenderingProvider; rp != nil {
		switch {
		case !validNPI(rp.NPI):
			missing.add("valid NPI for provider %s", rp.Name)
		case rp.NPI != pr.NPI:
			if !taxonomyPattern.MatchString(rp.TaxonomyCode) {
				missing.add("taxonomy code for provider %s", rp.Name)
			}
			first, last := splitProviderName(rp.Name)
			rendering = &stediRenderingProvider{
				NPI:          rp.NPI,
				TaxonomyCode: rp.TaxonomyCode,
				FirstName:    x12Text(first),
				LastName:     x12Text(last),
			}
		}
	}

	// Patient and, for dependents, the policyholder.
	patientDOB := ""
	if pt.DateOfBirth.IsZero() {
		missing.add("patient date of birth")
	} else {
		patientDOB = pt.DateOfBirth.Format("20060102")
	}
	var patientAddr *stediAddress
	if pt.AddressLine1 == "" || pt.City == "" || pt.StateProvince == "" || pt.PostalCode == "" {
		missing.add("patient address")
	} else {
		patientAddr = &stediAddress{
			Address1:   x12Text(pt.AddressLine1),
			Address2:   x12Text(pt.AddressLine2),
			City:       x12Text(pt.City),
			State:      x12Text(pt.StateProvince),
			PostalCode: digitsOnly(pt.PostalCode),
		}
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
		if pt.InsuranceSubscriberFirstName == "" || pt.InsuranceSubscriberLastName == "" {
			missing.add("policyholder name")
		}
		subDOB, err := x12Date(pt.InsuranceSubscriberDOB)
		if err != nil {
			missing.add("policyholder date of birth")
		}
		if !validSubscriberRelationship(pt.InsuranceSubscriberRelationship) {
			missing.add("patient's relationship to the policyholder")
		}
		subscriber.FirstName = x12Text(pt.InsuranceSubscriberFirstName)
		subscriber.LastName = x12Text(pt.InsuranceSubscriberLastName)
		subscriber.DateOfBirth = subDOB
		dependent = &stediDependent{
			FirstName:                    x12Text(pt.FirstName),
			LastName:                     x12Text(pt.LastName),
			DateOfBirth:                  patientDOB,
			Gender:                       x12Gender(pt.Sex),
			RelationshipToSubscriberCode: pt.InsuranceSubscriberRelationship,
			Address:                      patientAddr,
		}
	}

	// Service lines.
	if len(c.LineItems) == 0 {
		missing.add("at least one procedure")
	}
	lines := make([]stediServiceLine, 0, len(c.LineItems))
	var total int64
	for i, li := range c.LineItems {
		if !cdtPattern.MatchString(li.ADACode) {
			missing.add("valid CDT code on line %d (got %q)", i+1, li.ADACode)
		}
		if li.Fee < 0 {
			missing.add("non-negative fee on line %d", i+1)
		}
		total += li.Fee
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
			tooth, ok := x12ToothCode(li.ToothNumber)
			if !ok {
				missing.add("valid tooth number on line %d", i+1)
			}
			row := stediToothInfoRow{ToothCode: tooth}
			for _, s := range li.Surfaces {
				row.ToothSurfaceCodes = append(row.ToothSurfaceCodes, string(s))
			}
			line.TeethInformation = []stediToothInfoRow{row}
		}
		lines = append(lines, line)
	}

	if err := missing.err(); err != nil {
		return nil, err
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
			EmployerID:       ein,
			OrganizationName: x12Text(pr.ClinicName),
			Address: stediAddress{
				Address1:   x12Text(pr.AddressLine1),
				Address2:   x12Text(pr.AddressLine2),
				City:       x12Text(pr.City),
				State:      x12Text(pr.StateProvince),
				PostalCode: billingZip,
			},
			ContactInformation: stediContact{Name: x12Text(pr.ClinicName), PhoneNumber: practicePhone},
		},
		Rendering: rendering,
		ClaimInformation: stediClaimInformation{
			ClaimFilingCode:                          "CI", // commercial insurance
			PatientControlNumber:                     c.PatientControlNumber(),
			ClaimChargeAmount:                        centsToDecimal(total),
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

// ─── X12 value helpers ───────────────────────────────────────────────────────

var (
	cdtPattern      = regexp.MustCompile(`^D\d{4}$`)
	taxonomyPattern = regexp.MustCompile(`^[0-9A-Z]{9}X$`)
)

// x12Delimiters are reserved by the X12 envelope Stedi generates and cannot be escaped.
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
