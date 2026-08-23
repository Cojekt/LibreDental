package domain

import "context"

// ClaimSubmissionResult represents the response from an external clearinghouse.
type ClaimSubmissionResult struct {
	ExternalClaimID string      `json:"external_claim_id"`
	Status          ClaimStatus `json:"status"`
	Messages        []string    `json:"messages"`
	RawResponse     []byte      `json:"raw_response"` // Kept for audit purposes
}

// ClaimProvider defines the contract for any external insurance integration.
type ClaimProvider interface {
	// Name returns the unique identifier for this provider (e.g., "dentalxchange", "cdanet", "manual_pdf", "mock")
	Name() string

	// SupportedCountries returns the countries this provider can process claims for
	SupportedCountries() []CountryCode

	// SubmitClaim sends the claim to the external system
	// config contains the decrypted credentials and settings for this provider
	SubmitClaim(ctx context.Context, claim *Claim, config map[string]string) (*ClaimSubmissionResult, error)
}
