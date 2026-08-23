package claims

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

// MockProvider is a dummy provider for testing claim submissions without
// needing actual clearinghouse credentials.
type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Name() string {
	return "mock"
}

func (p *MockProvider) SupportedCountries() []domain.CountryCode {
	// Supports any country for testing
	return []domain.CountryCode{domain.CountryUS, domain.CountryCA, domain.CountryGB, domain.CountryAU, domain.CountryDE, domain.CountryFR}
}

func (p *MockProvider) SubmitClaim(ctx context.Context, claim *domain.Claim, config map[string]string) (*domain.ClaimSubmissionResult, error) {
	// Simulate network delay
	select {
	case <-time.After(1 * time.Second):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// We can use a simple rule for the mock:
	// If the claim has no line items, reject it.
	// Otherwise, accept it.
	if len(claim.LineItems) == 0 {
		return &domain.ClaimSubmissionResult{
			ExternalClaimID: fmt.Sprintf("mock-err-%d", time.Now().UnixNano()),
			Status:          domain.ClaimStatusRejected,
			Messages:        []string{"Mock provider rejected claim: no line items"},
			RawResponse:     []byte(`{"error": "no line items"}`),
		}, nil
	}

	return &domain.ClaimSubmissionResult{
		ExternalClaimID: fmt.Sprintf("mock-ok-%d", time.Now().UnixNano()),
		Status:          domain.ClaimStatusSubmitted,
		Messages:        []string{"Mock provider successfully received claim"},
		RawResponse:     fmt.Appendf(nil, `{"success": true, "claim_id": "%s"}`, claim.ID),
	}, nil
}
