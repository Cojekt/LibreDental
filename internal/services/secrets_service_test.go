package services

import (
	"testing"

	"github.com/zalando/go-keyring"
)

// init is intentionally shared with billing_service_test.go — keyring.MockInit
// is idempotent (calling it again is safe).

func TestSecretsService_GetProviderConfig_Empty(t *testing.T) {
	keyring.MockInit()
	svc := NewSecretsService()

	cfg, err := svc.GetProviderConfig("nonexistent_provider")
	if err != nil {
		t.Fatalf("Expected no error for missing provider, got: %v", err)
	}
	if len(cfg) != 0 {
		t.Errorf("Expected empty map for missing provider, got: %v", cfg)
	}
}

func TestSecretsService_SetAndGetProviderConfig(t *testing.T) {
	keyring.MockInit()
	svc := NewSecretsService()

	providerName := "test_integration"
	input := map[string]string{
		"api_key": "super-secret-key-123",
		"region":  "us-east-1",
	}

	if err := svc.SetProviderConfig(providerName, input); err != nil {
		t.Fatalf("SetProviderConfig failed: %v", err)
	}

	cfg, err := svc.GetProviderConfig(providerName)
	if err != nil {
		t.Fatalf("GetProviderConfig failed: %v", err)
	}

	// API key must be redacted on read
	if cfg["api_key"] != "********" {
		t.Errorf("Expected api_key to be redacted, got %q", cfg["api_key"])
	}

	// Other fields must be preserved
	if cfg["region"] != "us-east-1" {
		t.Errorf("Expected region to be %q, got %q", "us-east-1", cfg["region"])
	}
}

func TestSecretsService_SetProviderConfig_RedactedSentinelRestoresKey(t *testing.T) {
	keyring.MockInit()
	svc := NewSecretsService()

	providerName := "test_integration_restore"
	original := map[string]string{
		"api_key": "original-secret-key",
		"mode":    "production",
	}

	if err := svc.SetProviderConfig(providerName, original); err != nil {
		t.Fatalf("Initial SetProviderConfig failed: %v", err)
	}

	// Simulate the frontend sending the redacted sentinel back
	withRedacted := map[string]string{
		"api_key": "********",
		"mode":    "staging",
	}
	if err := svc.SetProviderConfig(providerName, withRedacted); err != nil {
		t.Fatalf("SetProviderConfig with sentinel failed: %v", err)
	}

	// The raw config should still have the original key
	raw, err := svc.getRawProviderConfig(providerName)
	if err != nil {
		t.Fatalf("getRawProviderConfig failed: %v", err)
	}
	if raw["api_key"] != "original-secret-key" {
		t.Errorf("Expected original api_key to be restored, got %q", raw["api_key"])
	}
	if raw["mode"] != "staging" {
		t.Errorf("Expected mode to be updated to 'staging', got %q", raw["mode"])
	}
}

func TestSecretsService_DeleteProviderConfig(t *testing.T) {
	keyring.MockInit()
	svc := NewSecretsService()

	providerName := "test_integration_delete"
	if err := svc.SetProviderConfig(providerName, map[string]string{"api_key": "key"}); err != nil {
		t.Fatalf("SetProviderConfig failed: %v", err)
	}

	if err := svc.DeleteProviderConfig(providerName); err != nil {
		t.Fatalf("DeleteProviderConfig failed: %v", err)
	}

	// After delete, GetProviderConfig should return empty map
	cfg, err := svc.GetProviderConfig(providerName)
	if err != nil {
		t.Fatalf("GetProviderConfig after delete failed: %v", err)
	}
	if len(cfg) != 0 {
		t.Errorf("Expected empty map after delete, got: %v", cfg)
	}
}

func TestSecretsService_DeleteProviderConfig_Idempotent(t *testing.T) {
	keyring.MockInit()
	svc := NewSecretsService()

	// Deleting a provider that was never stored should not error
	if err := svc.DeleteProviderConfig("never_stored_provider"); err != nil {
		t.Errorf("Expected no error deleting nonexistent provider, got: %v", err)
	}
}
