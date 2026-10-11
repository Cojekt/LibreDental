package domain

import (
	"context"
	"errors"
	"time"
)

// ErrDeliveryUnknown marks a failed send whose message may still have reached the recipient,
// for example a timeout after the vendor received the request. Retrying it could deliver a
// duplicate, so callers must not retry automatically. Any other provider error means the
// message was not sent.
var ErrDeliveryUnknown = errors.New("delivery outcome unknown")

// NotificationChannel identifies how a notification is delivered.
type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "email"
	NotificationChannelSMS   NotificationChannel = "sms"
	NotificationChannelVoice NotificationChannel = "voice"
)

// NotificationStatus represents the delivery outcome of a notification attempt.
type NotificationStatus string

const (
	NotificationStatusSent   NotificationStatus = "sent"
	NotificationStatusFailed NotificationStatus = "failed"
)

// NotificationMessage is the payload handed to a NotificationProvider to deliver.
type NotificationMessage struct {
	Channel NotificationChannel `json:"channel"`
	To      string              `json:"to"`                // email address or phone number, per channel
	Subject string              `json:"subject,omitempty"` // email only
	Body    string              `json:"body"`
}

// NotificationResult represents the response from an external notification provider.
type NotificationResult struct {
	ExternalMessageID string             `json:"external_message_id"`
	Status            NotificationStatus `json:"status"`
	Messages          []string           `json:"messages,omitempty"`
	RawResponse       []byte             `json:"raw_response,omitempty"` // For provider-side debugging; not persisted
}

// NotificationProvider defines the contract for any external notification integration
// (an email, SMS, or voice-call vendor).
type NotificationProvider interface {
	// Name returns the unique identifier for this provider (e.g., "twilio_sms", "sendgrid_email", "mock").
	Name() string

	// Channel returns which delivery channel this provider handles.
	Channel() NotificationChannel

	// Send delivers the message via the external system.
	// config contains the decrypted credentials and settings for this provider.
	Send(ctx context.Context, msg *NotificationMessage, config map[string]string) (*NotificationResult, error)
}

// NotificationLog is a persisted record of a notification attempt, kept independently
// of the HIPAA audit trail so delivery history can be queried per patient/appointment.
type NotificationLog struct {
	ID            string              `json:"id"`
	PatientID     string              `json:"patient_id"`
	AppointmentID string              `json:"appointment_id,omitempty"`
	Channel       NotificationChannel `json:"channel"`
	ProviderName  string              `json:"provider_name"`
	// ExternalMessageID is the vendor's message ID, used to reconcile later delivery-status
	// updates and to reference a message when raising issues with the vendor.
	ExternalMessageID string             `json:"external_message_id,omitempty"`
	Recipient         string             `json:"recipient"`
	Subject           string             `json:"subject,omitempty"`
	Body              string             `json:"body"`
	Status            NotificationStatus `json:"status"`
	ErrorMessage      string             `json:"error_message,omitempty"`
	SentAt            time.Time          `json:"sent_at"`
}
