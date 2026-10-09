package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/aws/smithy-go"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

var e164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{1,14}$`)

// AWSSMSProvider sends text messages through AWS End User Messaging SMS (SendTextMessage),
// using credentials from the practice's own AWS account.
//
// The client is built from the stored settings only. The SDK's config package is deliberately
// not used: its default credential chain reads environment variables and ~/.aws, which would
// let the app silently send with a developer's own AWS login.
//
// SendTextMessage has no idempotency token, and the SDK's default retryer repeats requests
// after server errors and timeouts, when the first attempt may already have texted the patient.
// Each send is therefore attempted exactly once.
type AWSSMSProvider struct {
	// endpoint and httpClient replace the AWS endpoint and HTTP client; only tests set them.
	endpoint   string
	httpClient pinpointsmsvoicev2.HTTPClient
}

func NewAWSSMSProvider() *AWSSMSProvider {
	return &AWSSMSProvider{}
}

func (p *AWSSMSProvider) Name() string { return "aws_sms" }

func (p *AWSSMSProvider) Channel() domain.NotificationChannel {
	return domain.NotificationChannelSMS
}

type awsSMSConfig struct {
	accessKeyID         string
	secretAccessKey     string
	region              string
	originationIdentity string
	configurationSet    string
}

// parseAWSSMSConfig validates the stored settings and reports every problem at once.
func parseAWSSMSConfig(config map[string]string) (*awsSMSConfig, error) {
	cfg := &awsSMSConfig{
		accessKeyID:         strings.TrimSpace(config["access_key_id"]),
		secretAccessKey:     strings.TrimSpace(config["secret_access_key"]),
		region:              strings.TrimSpace(config["region"]),
		originationIdentity: strings.TrimSpace(config["origination_identity"]),
		configurationSet:    strings.TrimSpace(config["configuration_set"]),
	}
	var problems []string
	if cfg.accessKeyID == "" {
		problems = append(problems, "access key ID is required")
	}
	if cfg.secretAccessKey == "" {
		problems = append(problems, "secret access key is required")
	}
	if cfg.region == "" {
		problems = append(problems, "region is required")
	}
	if cfg.originationIdentity == "" {
		problems = append(problems, "origination identity (phone number, pool, or their ID or ARN) is required")
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("%w: AWS SMS settings: %s", storage.ErrInvalidInput, strings.Join(problems, "; "))
	}
	return cfg, nil
}

func (p *AWSSMSProvider) Send(ctx context.Context, msg *domain.NotificationMessage, config map[string]string) (*domain.NotificationResult, error) {
	return p.send(ctx, msg, config, false)
}

// send is Send with AWS's DryRun option, which validates the request without sending or
// charging for it. The opt-in live test uses it.
func (p *AWSSMSProvider) send(ctx context.Context, msg *domain.NotificationMessage, config map[string]string, dryRun bool) (*domain.NotificationResult, error) {
	cfg, err := parseAWSSMSConfig(config)
	if err != nil {
		return nil, err
	}
	if !e164Pattern.MatchString(msg.To) {
		return nil, fmt.Errorf("%w: %q is not a phone number in international format", storage.ErrInvalidInput, msg.To)
	}
	if err := checkSMSBody(msg.Body); err != nil {
		return nil, err
	}

	var base pinpointsmsvoicev2.HTTPClient = awshttp.NewBuildableClient()
	if p.httpClient != nil {
		base = p.httpClient
	}
	tracker := &connectionTracker{client: base}
	opts := pinpointsmsvoicev2.Options{
		Region:      cfg.region,
		Credentials: credentials.NewStaticCredentialsProvider(cfg.accessKeyID, cfg.secretAccessKey, ""),
		Retryer:     aws.NopRetryer{},
		HTTPClient:  tracker,
	}
	if p.endpoint != "" {
		opts.BaseEndpoint = aws.String(p.endpoint)
	}
	client := pinpointsmsvoicev2.New(opts)

	input := &pinpointsmsvoicev2.SendTextMessageInput{
		DestinationPhoneNumber: aws.String(msg.To),
		OriginationIdentity:    aws.String(cfg.originationIdentity),
		MessageBody:            aws.String(msg.Body),
		MessageType:            types.MessageTypeTransactional,
		DryRun:                 dryRun,
	}
	if cfg.configurationSet != "" {
		input.ConfigurationSetName = aws.String(cfg.configurationSet)
	}

	out, err := client.SendTextMessage(ctx, input)
	if err != nil {
		return &domain.NotificationResult{Status: domain.NotificationStatusFailed}, describeAWSSMSError(err, tracker.connected.Load())
	}
	return &domain.NotificationResult{ExternalMessageID: aws.ToString(out.MessageId), Status: domain.NotificationStatusSent}, nil
}

// connectionTracker wraps the SDK's HTTP client to record whether a connection to AWS was
// obtained. The SDK reports a cancelled or failed request the same way whether it happened
// before or after sending, but nothing can reach AWS without a connection.
//
// GotConn is used rather than WroteRequest because both HTTP/1 and HTTP/2 call it on the
// requesting goroutine before writing anything, so it has always run by the time Do returns.
// HTTP/2 (which the SDK's client attempts) writes the request on another goroutine, and a
// cancelled request can return before WroteRequest fires, which would make a sent text look
// unsent.
type connectionTracker struct {
	client    pinpointsmsvoicev2.HTTPClient
	connected atomic.Bool
}

func (t *connectionTracker) Do(req *http.Request) (*http.Response, error) {
	trace := &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) { t.connected.Store(true) },
	}
	return t.client.Do(req.WithContext(httptrace.WithClientTrace(req.Context(), trace)))
}

// describeAWSSMSError turns an SDK error into a message staff can act on, and marks it with
// domain.ErrDeliveryUnknown when the text may have been sent anyway. AWS answers every
// rejected request with HTTP 400, so only server errors, and requests that may have been sent
// but got no response, are uncertain. connected reports whether a connection to AWS was made.
func describeAWSSMSError(err error, connected bool) error {
	var (
		conflict   *types.ConflictException
		quota      *types.ServiceQuotaExceededException
		denied     *types.AccessDeniedException
		validation *types.ValidationException
		notFound   *types.ResourceNotFoundException
		throttled  *types.ThrottlingException
	)
	switch {
	case errors.As(err, &conflict) && conflict.Reason == types.ConflictExceptionReasonDestinationPhoneNumberOptedOut:
		return errors.New("the recipient has opted out of text messages from this number (replied STOP)")
	case errors.As(err, &conflict) && conflict.Reason == types.ConflictExceptionReasonDestinationPhoneNumberNotVerified:
		return errors.New("the AWS account is in the SMS sandbox and this number is not a verified destination")
	case errors.As(err, &quota) && quota.Reason == types.ServiceQuotaExceededExceptionReasonMonthlySpendLimitReachedForText:
		return errors.New("the AWS account's monthly SMS spend limit has been reached")
	case errors.As(err, &denied):
		return fmt.Errorf("AWS has restricted this account's SMS sending (%s): %s", denied.Reason, aws.ToString(denied.Message))
	case errors.As(err, &validation):
		return fmt.Errorf("AWS rejected the message as invalid (%s): %s", validation.Reason, aws.ToString(validation.Message))
	case errors.As(err, &notFound):
		return fmt.Errorf("AWS could not find %s %q; check the origination identity, configuration set, and region",
			notFound.ResourceType, aws.ToString(notFound.ResourceId))
	case errors.As(err, &throttled):
		return errors.New("AWS is limiting the sending rate; the message was not sent")
	case errors.As(err, &conflict):
		return fmt.Errorf("AWS rejected the message (%s): %s", conflict.Reason, aws.ToString(conflict.Message))
	case errors.As(err, &quota):
		return fmt.Errorf("AWS quota exceeded (%s): %s", quota.Reason, aws.ToString(quota.Message))
	}

	// The SDK rejects malformed input before building a request.
	var invalidParams smithy.InvalidParamsError
	var serialization *smithy.SerializationError
	if errors.As(err, &invalidParams) || errors.As(err, &serialization) {
		return fmt.Errorf("the SMS request could not be built: %w", err)
	}

	// A request that got no response is also reported as a ResponseError, with status 0.
	var respErr *awshttp.ResponseError
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() != 0 {
		if respErr.HTTPStatusCode() < 500 {
			return fmt.Errorf("AWS rejected the request: %w", err)
		}
		return fmt.Errorf("%w: AWS returned a server error: %v", domain.ErrDeliveryUnknown, err)
	}

	// No response at all. Without a connection (the context was cancelled first, or connecting
	// failed), AWS can't have received the request.
	if !connected {
		return fmt.Errorf("the request was not sent to AWS: %w", err)
	}
	return fmt.Errorf("%w: no response from AWS: %v", domain.ErrDeliveryUnknown, err)
}
