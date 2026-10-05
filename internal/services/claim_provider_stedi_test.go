package services

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

// These tests check API compatibility offline: requests are validated against the property
// list in testdata/stedi_dental_claim_fields.txt (extracted from Stedi's published API
// reference), and responses are replayed from Stedi's documented examples. No API key or
// network access is needed. TestStediLiveTestMode exercises the real API in test mode when
// STEDI_API_KEY is set.

const (
	testBillingNPI   = "1999999992" // Stedi's documented test NPIs
	testRenderingNPI = "1999999984"
)

func stediTestSubmission() *domain.ClaimSubmission {
	return &domain.ClaimSubmission{
		Claim: &domain.Claim{
			ID:               "claim_1759632000000000000",
			PatientID:        "pat_1",
			ProviderID:       "prov_2",
			InsuranceCarrier: "United HealthCare Dental",
			PolicyNumber:     "123412345",
			GroupNumber:      "1234567890",
			PayerID:          "52133",
			DateOfService:    "2026-09-28",
			Status:           domain.ClaimStatusDraft,
			LineItems: []domain.ClaimLineItem{
				{ID: "li_1", ADACode: "D0120", Fee: 6500},
				{ID: "li_2", ADACode: "D2392", Fee: 18550, ToothNumber: 3, Surfaces: []domain.ToothSurface{domain.SurfaceMesial, domain.SurfaceOcclusal}},
				{ID: "li_3", ADACode: "D1351", Fee: 5500, ToothNumber: 103}, // primary tooth C
			},
		},
		Patient: &domain.Patient{
			ID:                    "pat_1",
			FirstName:             "John",
			LastName:              "Doe",
			DateOfBirth:           time.Date(1985, 6, 15, 0, 0, 0, 0, time.UTC),
			Sex:                   domain.SexMale,
			AddressLine1:          "1234 Some St",
			City:                  "Buckeye",
			StateProvince:         "AZ",
			PostalCode:            "85326",
			InsuranceIsSubscriber: true,
		},
		Practice: &domain.PracticeConfig{
			ClinicName:    "Test Dental Group",
			TaxID:         "12-3456789",
			NPI:           testBillingNPI,
			Phone:         "(313) 123-4567",
			AddressLine1:  "123 Main St",
			AddressLine2:  "Suite 4",
			City:          "Denver",
			StateProvince: "CO",
			PostalCode:    "80238-3000",
		},
		RenderingProvider: &domain.Provider{
			ID:           "prov_2",
			Name:         "Dr. Jane Q. Smith, DDS",
			NPI:          testRenderingNPI,
			TaxonomyCode: "1223G0001X",
		},
	}
}

// ─── Documented-schema checker ───────────────────────────────────────────────

type stediField struct {
	typ      string
	required bool
}

func loadStediFields(t *testing.T) map[string]stediField {
	t.Helper()
	f, err := os.Open("testdata/stedi_dental_claim_fields.txt")
	if err != nil {
		t.Fatalf("open field list: %v", err)
	}
	defer f.Close()
	fields := map[string]stediField{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		fields[parts[0]] = stediField{typ: parts[1], required: len(parts) > 2 && parts[2] == "required"}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read field list: %v", err)
	}
	return fields
}

// checkAgainstStediSchema returns every place the JSON document departs from the documented
// request schema: undocumented properties, wrong JSON types, and missing required properties
// of objects that are present.
func checkAgainstStediSchema(fields map[string]stediField, doc []byte) []string {
	var root any
	if err := json.Unmarshal(doc, &root); err != nil {
		return []string{"invalid JSON: " + err.Error()}
	}
	var problems []string
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch val := v.(type) {
		case map[string]any:
			for key, child := range val {
				p := key
				if path != "" {
					p = path + "." + key
				}
				f, ok := fields[p]
				if !ok {
					problems = append(problems, "undocumented property "+p)
					continue
				}
				if msg := checkStediType(p, f.typ, child); msg != "" {
					problems = append(problems, msg)
					continue
				}
				walk(p, child)
			}
			prefix := path + "."
			if path == "" {
				prefix = ""
			}
			for p, f := range fields {
				if !f.required || !strings.HasPrefix(p, prefix) {
					continue
				}
				key := strings.TrimPrefix(p, prefix)
				if strings.ContainsAny(key, ".[") {
					continue
				}
				if _, ok := val[key]; !ok {
					problems = append(problems, "missing required property "+p)
				}
			}
		case []any:
			for _, el := range val {
				walk(path+"[]", el)
			}
		}
	}
	walk("", root)
	sort.Strings(problems)
	return problems
}

func checkStediType(path, typ string, v any) string {
	ok := true
	switch {
	case strings.HasPrefix(typ, "array"):
		_, ok = v.([]any)
	case typ == "string":
		_, ok = v.(string)
	case typ == "number" || typ == "integer":
		_, ok = v.(float64)
	case typ == "boolean":
		_, ok = v.(bool)
	case typ == "object":
		_, ok = v.(map[string]any)
	}
	if !ok {
		return fmt.Sprintf("property %s should be %s, got %T", path, typ, v)
	}
	return ""
}

func TestStediSchemaChecker_AcceptsDocumentedExample(t *testing.T) {
	fields := loadStediFields(t)
	example, err := os.ReadFile("testdata/stedi_dental_claim_example_request.json")
	if err != nil {
		t.Fatal(err)
	}
	if problems := checkAgainstStediSchema(fields, example); len(problems) > 0 {
		t.Fatalf("documented example should pass the checker:\n%s", strings.Join(problems, "\n"))
	}

	// And the checker must actually catch problems.
	bad := []byte(`{"tradingPartnerServiceId": 52133, "madeUp": true, "submitter": {"organizationName": "X"}}`)
	got := strings.Join(checkAgainstStediSchema(fields, bad), "\n")
	for _, want := range []string{
		"undocumented property madeUp",
		"property tradingPartnerServiceId should be string",
		"missing required property submitter.contactInformation",
		"missing required property claimInformation",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("checker missed %q; got:\n%s", want, got)
		}
	}
}

// ─── Request mapping ─────────────────────────────────────────────────────────

func marshalStediRequest(t *testing.T, sub *domain.ClaimSubmission, testMode bool) ([]byte, *stediDentalClaimRequest) {
	t.Helper()
	req, err := buildStediDentalClaim(sub, testMode)
	if err != nil {
		t.Fatalf("buildStediDentalClaim: %v", err)
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	return body, req
}

func TestStediRequest_ConformsToDocumentedSchema(t *testing.T) {
	fields := loadStediFields(t)

	subscriberSub := stediTestSubmission()
	dependentSub := stediTestSubmission()
	dependentSub.Patient.InsuranceIsSubscriber = false
	dependentSub.Patient.InsuranceSubscriberFirstName = "Mary"
	dependentSub.Patient.InsuranceSubscriberLastName = "Doe"
	dependentSub.Patient.InsuranceSubscriberDOB = "1983-02-01"
	dependentSub.Patient.InsuranceSubscriberRelationship = domain.SubscriberRelationshipChild
	soloSub := stediTestSubmission()
	soloSub.RenderingProvider.NPI = testBillingNPI

	for name, sub := range map[string]*domain.ClaimSubmission{
		"patient is subscriber": subscriberSub,
		"patient is dependent":  dependentSub,
		"solo practice":         soloSub,
	} {
		t.Run(name, func(t *testing.T) {
			body, _ := marshalStediRequest(t, sub, true)
			if problems := checkAgainstStediSchema(fields, body); len(problems) > 0 {
				t.Fatalf("request does not match Stedi's schema:\n%s\n\nrequest: %s", strings.Join(problems, "\n"), body)
			}
		})
	}
}

func TestStediRequest_Mapping(t *testing.T) {
	_, req := marshalStediRequest(t, stediTestSubmission(), true)

	if req.UsageIndicator != "T" {
		t.Errorf("usageIndicator = %q, want T", req.UsageIndicator)
	}
	if req.TradingPartnerServiceID != "52133" || req.Receiver.OrganizationName != "United HealthCare Dental" {
		t.Errorf("payer = %q/%q", req.TradingPartnerServiceID, req.Receiver.OrganizationName)
	}
	b := req.Billing
	if b.NPI != testBillingNPI || b.EmployerID != "123456789" || b.Address.PostalCode != "802383000" || b.ContactInformation.PhoneNumber != "3131234567" {
		t.Errorf("billing provider = %+v", b)
	}
	if req.Rendering == nil || req.Rendering.NPI != testRenderingNPI || req.Rendering.TaxonomyCode != "1223G0001X" ||
		req.Rendering.FirstName != "Jane Q." || req.Rendering.LastName != "Smith" {
		t.Errorf("rendering provider = %+v", req.Rendering)
	}
	s := req.Subscriber
	if s.MemberID != "123412345" || s.FirstName != "John" || s.DateOfBirth != "19850615" || s.Gender != "M" || s.Address == nil || req.Dependent != nil {
		t.Errorf("subscriber = %+v, dependent = %+v", s, req.Dependent)
	}

	ci := req.ClaimInformation
	if ci.ClaimChargeAmount != "305.50" {
		t.Errorf("claimChargeAmount = %q, want 305.50 (sum of lines)", ci.ClaimChargeAmount)
	}
	if ci.PatientControlNumber != stediTestSubmission().Claim.PatientControlNumber() {
		t.Errorf("patientControlNumber = %q", ci.PatientControlNumber)
	}
	if len(ci.ServiceLines) != 3 {
		t.Fatalf("got %d service lines", len(ci.ServiceLines))
	}
	l0, l1, l2 := ci.ServiceLines[0], ci.ServiceLines[1], ci.ServiceLines[2]
	if l0.ServiceDate != "20260928" || l0.DentalService.ProcedureCode != "D0120" || l0.DentalService.LineItemChargeAmount != "65.00" || l0.TeethInformation != nil {
		t.Errorf("line 1 = %+v", l0)
	}
	if len(l1.TeethInformation) != 1 || l1.TeethInformation[0].ToothCode != "3" || strings.Join(l1.TeethInformation[0].ToothSurfaceCodes, "") != "MO" {
		t.Errorf("line 2 teeth = %+v", l1.TeethInformation)
	}
	if l2.TeethInformation[0].ToothCode != "C" {
		t.Errorf("primary tooth 103 should map to C, got %q", l2.TeethInformation[0].ToothCode)
	}
	if l0.ProviderControlNumber == l1.ProviderControlNumber || l0.ProviderControlNumber != domain.ControlNumber("li_1") {
		t.Errorf("line control numbers should be distinct and stable: %q %q", l0.ProviderControlNumber, l1.ProviderControlNumber)
	}
}

func TestStediRequest_DependentAndSoloPractice(t *testing.T) {
	sub := stediTestSubmission()
	sub.Patient.InsuranceIsSubscriber = false
	sub.Patient.InsuranceSubscriberFirstName = "Mary"
	sub.Patient.InsuranceSubscriberLastName = "Doe"
	sub.Patient.InsuranceSubscriberDOB = "1983-02-01"
	sub.Patient.InsuranceSubscriberRelationship = domain.SubscriberRelationshipChild
	sub.RenderingProvider.NPI = testBillingNPI
	sub.RenderingProvider.TaxonomyCode = ""

	_, req := marshalStediRequest(t, sub, false)
	if req.UsageIndicator != "" {
		t.Errorf("production claims should omit usageIndicator, got %q", req.UsageIndicator)
	}
	if req.Subscriber.FirstName != "Mary" || req.Subscriber.DateOfBirth != "19830201" || req.Subscriber.Address != nil {
		t.Errorf("subscriber = %+v", req.Subscriber)
	}
	d := req.Dependent
	if d == nil || d.FirstName != "John" || d.DateOfBirth != "19850615" || d.RelationshipToSubscriberCode != "19" || d.Address == nil {
		t.Errorf("dependent = %+v", d)
	}
	if req.Rendering != nil {
		t.Errorf("rendering loop must be omitted when it matches the billing NPI, got %+v", req.Rendering)
	}
}

func TestStediRequest_ReportsAllMissingData(t *testing.T) {
	sub := stediTestSubmission()
	sub.Claim.PayerID = ""
	sub.Practice.NPI = "1234567890" // bad check digit
	sub.Practice.PostalCode = "80238"
	sub.Patient.InsuranceIsSubscriber = false
	sub.Claim.LineItems[0].ADACode = "0120"
	sub.Claim.LineItems[1].ToothNumber = 33

	_, err := buildStediDentalClaim(sub, true)
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

func TestStediRequest_StripsX12Delimiters(t *testing.T) {
	sub := stediTestSubmission()
	sub.Practice.ClinicName = "Smile*Care ~ Dental:Group^>"
	_, req := marshalStediRequest(t, sub, true)
	if got := req.Billing.OrganizationName; strings.ContainsAny(got, "~*:^>") || got != "Smile Care Dental Group" {
		t.Errorf("organizationName = %q", got)
	}
}

func TestValidNPI(t *testing.T) {
	for npi, want := range map[string]bool{
		"1234567893":     true, // CMS's published example
		testBillingNPI:   true,
		testRenderingNPI: true,
		"1234567890":     false,
		"123456789":      false,
		"12345678a3":     false,
	} {
		if got := validNPI(npi); got != want {
			t.Errorf("validNPI(%q) = %v, want %v", npi, got, want)
		}
	}
}

func TestControlNumber_FitsX12Constraints(t *testing.T) {
	pcn := domain.ControlNumber("claim_1759632000000000000")
	if !regexp.MustCompile(`^[A-Z2-7]{16}$`).MatchString(pcn) {
		t.Errorf("control number %q should be 16 uppercase base32 characters", pcn)
	}
	if pcn != domain.ControlNumber("claim_1759632000000000000") || pcn == domain.ControlNumber("claim_1759632000000000001") {
		t.Error("control numbers must be deterministic and distinct per ID")
	}
}

// ─── HTTP round trip against a fake Stedi ────────────────────────────────────

func newFakeStedi(t *testing.T, handler func(w http.ResponseWriter, body []byte)) *StediClaimProvider {
	t.Helper()
	fields := loadStediFields(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/dental-claims/submission" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "test-key" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("bad headers: %v", r.Header)
		}
		if k := r.Header.Get("Idempotency-Key"); !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(k) {
			t.Errorf("Idempotency-Key = %q", k)
		}
		body, _ := io.ReadAll(r.Body)
		if problems := checkAgainstStediSchema(fields, body); len(problems) > 0 {
			t.Errorf("request does not match Stedi's schema:\n%s", strings.Join(problems, "\n"))
		}
		handler(w, body)
	}))
	t.Cleanup(srv.Close)
	return &StediClaimProvider{baseURL: srv.URL, client: srv.Client()}
}

func TestStediSubmitClaim_Success(t *testing.T) {
	documented, err := os.ReadFile("testdata/stedi_dental_claim_success_response.json")
	if err != nil {
		t.Fatal(err)
	}
	var sentUsage string
	p := newFakeStedi(t, func(w http.ResponseWriter, body []byte) {
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		sentUsage, _ = req["usageIndicator"].(string)
		w.Write(documented)
	})

	result, err := p.SubmitClaim(context.Background(), stediTestSubmission(), map[string]string{StediConfigAPIKey: "test-key"})
	if err != nil {
		t.Fatalf("SubmitClaim: %v", err)
	}
	if sentUsage != "T" {
		t.Errorf("claims must default to test mode, sent usageIndicator %q", sentUsage)
	}
	if result.Status != domain.ClaimStatusSubmitted || result.ExternalClaimID != "01JDQMX92Q1T561BH8NKX750TQ" || len(result.Messages) != 0 {
		t.Errorf("result = %+v", result)
	}
	if string(result.RawResponse) != string(documented) {
		t.Error("raw response should be kept verbatim for the audit trail")
	}
}

func TestStediSubmitClaim_ProductionMode(t *testing.T) {
	var sentUsage any = "unset"
	p := newFakeStedi(t, func(w http.ResponseWriter, body []byte) {
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		sentUsage = req["usageIndicator"]
		w.Write([]byte(`{"status":"SUCCESS","claimReference":{"correlationId":"X"}}`))
	})
	_, err := p.SubmitClaim(context.Background(), stediTestSubmission(),
		map[string]string{StediConfigAPIKey: "test-key", StediConfigTestMode: "false"})
	if err != nil {
		t.Fatal(err)
	}
	if sentUsage != nil {
		t.Errorf("production submissions should omit usageIndicator, sent %v", sentUsage)
	}
}

func TestStediSubmitClaim_ClaimEditRejection(t *testing.T) {
	p := newFakeStedi(t, func(w http.ResponseWriter, body []byte) {
		w.Write([]byte(`{
			"status": "SUCCESS",
			"claimReference": {"correlationId": "01KREJECT"},
			"errors": [{"code": "33", "description": "Subscriber/Insured ID is invalid", "location": "subscriber.memberId", "followupAction": "Please correct and resubmit"}]
		}`))
	})
	result, err := p.SubmitClaim(context.Background(), stediTestSubmission(), map[string]string{StediConfigAPIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.ClaimStatusRejected || len(result.Messages) != 1 ||
		result.Messages[0] != "33: Subscriber/Insured ID is invalid (subscriber.memberId) - Please correct and resubmit" {
		t.Errorf("result = %+v", result)
	}
}

func TestStediSubmitClaim_HTTPErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   string
	}{
		{400, `{"code":"ValidationException","message":"claimInformation.claimChargeAmount must balance"}`, "HTTP 400): claimInformation.claimChargeAmount must balance"},
		{403, `{"code":"AccessDeniedException","message":"Test API keys cannot submit production claims"}`, "HTTP 403): Test API keys cannot submit production claims"},
		{503, ``, "HTTP 503): Service Unavailable"},
	} {
		p := newFakeStedi(t, func(w http.ResponseWriter, body []byte) {
			w.WriteHeader(tc.status)
			w.Write([]byte(tc.body))
		})
		_, err := p.SubmitClaim(context.Background(), stediTestSubmission(), map[string]string{StediConfigAPIKey: "test-key"})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("HTTP %d: err = %v, want %q", tc.status, err, tc.want)
		}
	}
}

func TestStediSubmitClaim_NothingSentWithoutKeyOrData(t *testing.T) {
	p := &StediClaimProvider{baseURL: "http://127.0.0.1:0", client: http.DefaultClient}
	if _, err := p.SubmitClaim(context.Background(), stediTestSubmission(), map[string]string{}); err == nil || !strings.Contains(err.Error(), "API key") {
		t.Errorf("missing key: err = %v", err)
	}
	sub := stediTestSubmission()
	sub.Claim.PayerID = ""
	if _, err := p.SubmitClaim(context.Background(), sub, map[string]string{StediConfigAPIKey: "k"}); !errors.Is(err, ErrClaimDataIncomplete) {
		t.Errorf("incomplete claim: err = %v", err)
	}
}

// TestStediLiveTestMode submits a test claim to the real Stedi API. Test claims are processed
// by Stedi's test clearinghouse and never reach a payer. It needs a Stedi test API key, which
// requires a Stedi production (pay-as-you-go) account; sandbox accounts can't submit claims.
//
//	STEDI_API_KEY=... go test ./internal/services -run TestStediLiveTestMode -v
func TestStediLiveTestMode(t *testing.T) {
	key := os.Getenv("STEDI_API_KEY")
	if key == "" {
		t.Skip("STEDI_API_KEY not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sub := stediTestSubmission()
	sub.Claim.ID = fmt.Sprintf("claim_live_%d", time.Now().UnixNano())

	result, err := NewStediClaimProvider().SubmitClaim(ctx, sub, map[string]string{StediConfigAPIKey: key})
	if err != nil {
		t.Fatalf("SubmitClaim: %v", err)
	}
	t.Logf("status=%s reference=%s messages=%v", result.Status, result.ExternalClaimID, result.Messages)
	t.Logf("raw response: %s", result.RawResponse)
	if result.Status != domain.ClaimStatusSubmitted {
		t.Errorf("Stedi's claim edits rejected the test claim: %v", result.Messages)
	}
}
