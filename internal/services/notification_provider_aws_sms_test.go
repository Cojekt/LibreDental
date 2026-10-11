package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// The SMS tests run against an httptest server standing in for the AWS endpoint. The SDK talks
// to it with the AWS JSON 1.0 protocol; error responses use the exception names and reason
// codes from AWS's API model.

type fakeAWSRequest struct {
	target        string
	contentType   string
	authorization string
	body          map[string]any
}

type fakeAWSSMS struct {
	server   *httptest.Server
	attempts atomic.Int32
	mu       sync.Mutex
	last     fakeAWSRequest
}

func newFakeAWSSMS(t *testing.T, respond func(w http.ResponseWriter)) *fakeAWSSMS {
	t.Helper()
	f := &fakeAWSSMS{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.attempts.Add(1)
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		f.mu.Lock()
		f.last = fakeAWSRequest{
			target:        r.Header.Get("X-Amz-Target"),
			contentType:   r.Header.Get("Content-Type"),
			authorization: r.Header.Get("Authorization"),
			body:          body,
		}
		f.mu.Unlock()
		respond(w)
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeAWSSMS) lastRequest() fakeAWSRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last
}

func awsJSONResponse(status int, body string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func testAWSSMSConfig() map[string]string {
	return map[string]string{
		"access_key_id":        "AKIATESTKEY",
		"secret_access_key":    "test-secret",
		"region":               "us-east-1",
		"origination_identity": "+12065550100",
	}
}

func testSMSMessage() *domain.NotificationMessage {
	return &domain.NotificationMessage{
		Channel: domain.NotificationChannelSMS,
		To:      "+12025550123",
		Body:    "Smile Dental: reminder of your appointment tomorrow at 2:00 PM. Reply STOP to opt out.",
	}
}

func TestParseAWSSMSConfig(t *testing.T) {
	if _, err := parseAWSSMSConfig(testAWSSMSConfig()); err != nil {
		t.Fatalf("Expected valid config to parse, got %v", err)
	}
	_, err := parseAWSSMSConfig(map[string]string{"configuration_set": "reminders"})
	if !errors.Is(err, storage.ErrInvalidInput) {
		t.Fatalf("Expected empty config to be rejected, got %v", err)
	}
	for _, want := range []string{"access key ID", "secret access key", "region", "origination identity"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected all problems reported at once; %q missing from %q", want, err)
		}
	}
}

func TestAWSSMSProvider_SendRequest(t *testing.T) {
	// The provider must use only its saved settings, never the environment's AWS login.
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAENVIRONMENT")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "env-secret")
	t.Setenv("AWS_PROFILE", "developer")
	t.Setenv("AWS_REGION", "eu-west-1")

	fake := newFakeAWSSMS(t, awsJSONResponse(200, `{"MessageId":"msg-0123"}`))
	provider := &AWSSMSProvider{endpoint: fake.server.URL}

	result, err := provider.Send(context.Background(), testSMSMessage(), testAWSSMSConfig())
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if result.Status != domain.NotificationStatusSent || result.ExternalMessageID != "msg-0123" {
		t.Errorf("Unexpected result: %+v", result)
	}

	req := fake.lastRequest()
	if req.target != "PinpointSMSVoiceV2.SendTextMessage" {
		t.Errorf("X-Amz-Target = %q", req.target)
	}
	if req.contentType != "application/x-amz-json-1.0" {
		t.Errorf("Content-Type = %q", req.contentType)
	}
	if !strings.HasPrefix(req.authorization, "AWS4-HMAC-SHA256 Credential=AKIATESTKEY/") ||
		!strings.Contains(req.authorization, "/us-east-1/sms-voice/aws4_request") {
		t.Errorf("Expected a SigV4 signature with the saved key and region, got %q", req.authorization)
	}
	want := map[string]any{
		"DestinationPhoneNumber": "+12025550123",
		"OriginationIdentity":    "+12065550100",
		"MessageBody":            testSMSMessage().Body,
		"MessageType":            "TRANSACTIONAL",
	}
	for k, v := range want {
		if req.body[k] != v {
			t.Errorf("%s = %v; want %v", k, req.body[k], v)
		}
	}
	for _, k := range []string{"ConfigurationSetName", "Context", "MaxPrice"} {
		if _, ok := req.body[k]; ok {
			t.Errorf("Expected %s to be omitted, got %v", k, req.body[k])
		}
	}
	if dry, ok := req.body["DryRun"]; ok && dry != false {
		t.Errorf("Expected a real send, got DryRun=%v", dry)
	}

	config := testAWSSMSConfig()
	config["configuration_set"] = "reminders"
	if _, err := provider.send(context.Background(), testSMSMessage(), config, true); err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}
	req = fake.lastRequest()
	if req.body["ConfigurationSetName"] != "reminders" || req.body["DryRun"] != true {
		t.Errorf("Expected configuration set and DryRun to be sent, got %v", req.body)
	}
}

func TestAWSSMSProvider_SendErrors(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantErr     string
		wantUnknown bool
	}{
		{"opted out", 400, `{"__type":"ConflictException","Message":"opted out","Reason":"DESTINATION_PHONE_NUMBER_OPTED_OUT","ResourceType":"opt-out-list","ResourceId":"Default"}`, "opted out", false},
		{"sandbox", 400, `{"__type":"ConflictException","Message":"not verified","Reason":"DESTINATION_PHONE_NUMBER_NOT_VERIFIED"}`, "SMS sandbox", false},
		{"other conflict", 400, `{"__type":"ConflictException","Message":"wrong message type","Reason":"MESSAGE_TYPE_MISMATCH"}`, "MESSAGE_TYPE_MISMATCH", false},
		{"spend limit", 400, `{"__type":"ServiceQuotaExceededException","Message":"limit","Reason":"MONTHLY_SPEND_LIMIT_REACHED_FOR_TEXT"}`, "spend limit", false},
		{"account disabled", 400, `{"__type":"AccessDeniedException","Message":"disabled","Reason":"ACCOUNT_DISABLED"}`, "ACCOUNT_DISABLED", false},
		{"blocked country", 400, `{"__type":"ValidationException","Message":"blocked","Reason":"DESTINATION_COUNTRY_BLOCKED","Fields":[]}`, "DESTINATION_COUNTRY_BLOCKED", false},
		{"unknown identity", 400, `{"__type":"ResourceNotFoundException","Message":"not found","ResourceType":"phone-number","ResourceId":"+12065550100"}`, "could not find", false},
		{"throttled", 400, `{"__type":"ThrottlingException","Message":"slow down"}`, "limiting the sending rate", false},
		{"bad credentials", 400, `{"__type":"UnrecognizedClientException","message":"The security token included in the request is invalid."}`, "rejected the request", false},
		{"internal error", 500, `{"__type":"InternalServerException","Message":"oops","RequestId":"r-1"}`, "server error", true},
		{"unavailable", 503, `{"__type":"ServiceUnavailableException","message":"unavailable"}`, "server error", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeAWSSMS(t, awsJSONResponse(tt.status, tt.body))
			provider := &AWSSMSProvider{endpoint: fake.server.URL}
			result, err := provider.Send(context.Background(), testSMSMessage(), testAWSSMSConfig())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Expected error containing %q, got %v", tt.wantErr, err)
			}
			if got := errors.Is(err, domain.ErrDeliveryUnknown); got != tt.wantUnknown {
				t.Errorf("errors.Is(err, ErrDeliveryUnknown) = %v; want %v (err: %v)", got, tt.wantUnknown, err)
			}
			if result == nil || result.Status != domain.NotificationStatusFailed {
				t.Errorf("Expected a failed result, got %+v", result)
			}
			// SendTextMessage has no idempotency token, so nothing may be retried.
			if n := fake.attempts.Load(); n != 1 {
				t.Errorf("Expected exactly 1 attempt, got %d", n)
			}
		})
	}
}

func TestAWSSMSProvider_SendWithoutResponse(t *testing.T) {
	// The request reaches the server, which drops the connection without answering: the text
	// may have been sent.
	fake := newFakeAWSSMS(t, func(w http.ResponseWriter) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
	})
	provider := &AWSSMSProvider{endpoint: fake.server.URL}
	_, err := provider.Send(context.Background(), testSMSMessage(), testAWSSMSConfig())
	if !errors.Is(err, domain.ErrDeliveryUnknown) {
		t.Errorf("Expected a dropped connection to be uncertain, got %v", err)
	}
	if n := fake.attempts.Load(); n != 1 {
		t.Errorf("Expected exactly 1 attempt, got %d", n)
	}

	// Nothing listening: the request never left, so it's definitely not sent.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closedURL := "http://" + ln.Addr().String()
	ln.Close()
	provider = &AWSSMSProvider{endpoint: closedURL}
	_, err = provider.Send(context.Background(), testSMSMessage(), testAWSSMSConfig())
	if err == nil || errors.Is(err, domain.ErrDeliveryUnknown) {
		t.Errorf("Expected a connection failure that is not uncertain, got %v", err)
	}
}

func TestAWSSMSProvider_RejectsBeforeRequest(t *testing.T) {
	fake := newFakeAWSSMS(t, awsJSONResponse(200, `{"MessageId":"msg-0123"}`))
	provider := &AWSSMSProvider{endpoint: fake.server.URL}
	ctx := context.Background()

	for _, to := range []string{"2025550123", "+0123", "", "+1 202 555 0123"} {
		msg := testSMSMessage()
		msg.To = to
		if _, err := provider.Send(ctx, msg, testAWSSMSConfig()); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected destination %q to be rejected, got %v", to, err)
		}
	}
	for _, body := range []string{"", strings.Repeat("a", 1531)} {
		msg := testSMSMessage()
		msg.Body = body
		if _, err := provider.Send(ctx, msg, testAWSSMSConfig()); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected a %d-character body to be rejected, got %v", len(body), err)
		}
	}
	if _, err := provider.Send(ctx, testSMSMessage(), map[string]string{}); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected missing settings to be rejected, got %v", err)
	}
	if n := fake.attempts.Load(); n != 0 {
		t.Errorf("Expected no requests to AWS, got %d", n)
	}
}

// TestAWSSMSLive calls AWS End User Messaging SMS. It only runs when AWS_SMS_LIVE_TEST=1, and by
// default only sends a DryRun request, which AWS validates without sending or charging for.
// Set AWS_SMS_LIVE_SEND=1 to also send a real message; use a simulator origination number and
// a simulator destination number so nothing reaches a carrier.
//
//	AWS_SMS_LIVE_TEST=1 AWS_SMS_ACCESS_KEY_ID=... AWS_SMS_SECRET_ACCESS_KEY=... AWS_SMS_REGION=us-east-1 \
//	AWS_SMS_ORIGINATION_IDENTITY=... AWS_SMS_DESTINATION=+1... go test ./internal/services -run TestAWSSMSLive -v
func TestAWSSMSLive(t *testing.T) {
	if os.Getenv("AWS_SMS_LIVE_TEST") != "1" {
		t.Skip("AWS_SMS_LIVE_TEST not set")
	}
	config := map[string]string{
		"access_key_id":        os.Getenv("AWS_SMS_ACCESS_KEY_ID"),
		"secret_access_key":    os.Getenv("AWS_SMS_SECRET_ACCESS_KEY"),
		"region":               os.Getenv("AWS_SMS_REGION"),
		"origination_identity": os.Getenv("AWS_SMS_ORIGINATION_IDENTITY"),
		"configuration_set":    os.Getenv("AWS_SMS_CONFIGURATION_SET"),
	}
	msg := &domain.NotificationMessage{
		Channel: domain.NotificationChannelSMS,
		To:      os.Getenv("AWS_SMS_DESTINATION"),
		Body:    "LibreDental SMS live test.",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	provider := NewAWSSMSProvider()
	if _, err := provider.send(ctx, msg, config, true); err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}
	if os.Getenv("AWS_SMS_LIVE_SEND") != "1" {
		return
	}
	result, err := provider.Send(ctx, msg, config)
	if err != nil {
		t.Fatalf("Live send failed: %v", err)
	}
	t.Logf("Sent; message ID %s", result.ExternalMessageID)
}

// The SDK reports a cancelled context the same way whether it happened before or after the
// request was sent; only a request that reached the network can leave the outcome unknown.
func TestAWSSMSProvider_CancellationBeforeAndAfterSending(t *testing.T) {
	fake := newFakeAWSSMS(t, awsJSONResponse(200, `{"MessageId":"msg-0123"}`))
	provider := &AWSSMSProvider{endpoint: fake.server.URL}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := provider.Send(cancelled, testSMSMessage(), testAWSSMSConfig())
	if err == nil || errors.Is(err, domain.ErrDeliveryUnknown) {
		t.Errorf("Expected a definite failure for a request cancelled before sending, got %v", err)
	}
	if n := fake.attempts.Load(); n != 0 {
		t.Errorf("Expected no request to reach AWS, got %d", n)
	}

	// The request reaches AWS, then the context is cancelled while waiting for the response.
	// Cancelling only once the server has the request keeps this independent of timing.
	received := make(chan struct{})
	release := make(chan struct{})
	slow := newFakeAWSSMS(t, func(w http.ResponseWriter) {
		close(received)
		<-release
	})
	t.Cleanup(func() { close(release) })
	provider = &AWSSMSProvider{endpoint: slow.server.URL}
	ctx, cancelSlow := context.WithCancel(context.Background())
	defer cancelSlow()
	go func() {
		<-received
		cancelSlow()
	}()
	_, err = provider.Send(ctx, testSMSMessage(), testAWSSMSConfig())
	if !errors.Is(err, domain.ErrDeliveryUnknown) {
		t.Errorf("Expected a cancellation after sending to be uncertain, got %v", err)
	}
	if n := slow.attempts.Load(); n != 1 {
		t.Errorf("Expected exactly 1 attempt, got %d", n)
	}
}

// The SDK's client attempts HTTP/2, which writes requests on a separate goroutine; a request
// cancelled after it reached AWS must still be reported as uncertain there.
func TestAWSSMSProvider_CancellationAfterSendingOverHTTP2(t *testing.T) {
	received := make(chan int, 1)
	release := make(chan struct{})
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		received <- r.ProtoMajor
		<-release
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) })

	provider := &AWSSMSProvider{endpoint: server.URL, httpClient: server.Client()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	proto := make(chan int, 1)
	go func() {
		p := <-received
		proto <- p
		cancel()
	}()
	_, err := provider.Send(ctx, testSMSMessage(), testAWSSMSConfig())
	if p := <-proto; p != 2 {
		t.Fatalf("Expected the request over HTTP/2, got HTTP/%d", p)
	}
	if !errors.Is(err, domain.ErrDeliveryUnknown) {
		t.Errorf("Expected a cancellation after sending over HTTP/2 to be uncertain, got %v", err)
	}
}
