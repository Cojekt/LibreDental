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

func TestSecretsService_SetProviderConfig_OmittedSecretKeepsKey(t *testing.T) {
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

	// The frontend never receives the key, so it leaves it out when it isn't changed.
	if err := svc.SetProviderConfig(providerName, map[string]string{"mode": "staging"}); err != nil {
		t.Fatalf("SetProviderConfig without the key failed: %v", err)
	}

	raw, err := svc.getRawProviderConfig(providerName)
	if err != nil {
		t.Fatalf("getRawProviderConfig failed: %v", err)
	}
	if raw["api_key"] != "original-secret-key" {
		t.Errorf("Expected original api_key to be kept, got %q", raw["api_key"])
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

func TestSecretsService_RedactsEverySecretField(t *testing.T) {
	keyring.MockInit()
	svc := NewSecretsService()
	providerName := "test_integration_secrets"

	if err := svc.SetProviderConfig(providerName, map[string]string{
		"api_key":           "key-1",
		"password":          "smtp-password",
		"secret_access_key": "aws-secret",
		"host":              "smtp.example.com",
	}); err != nil {
		t.Fatalf("SetProviderConfig failed: %v", err)
	}

	cfg, err := svc.GetProviderConfig(providerName)
	if err != nil {
		t.Fatalf("GetProviderConfig failed: %v", err)
	}
	for _, k := range []string{"api_key", "password", "secret_access_key"} {
		if cfg[k] != redactedSecret {
			t.Errorf("Expected %s to be redacted, got %q", k, cfg[k])
		}
	}
	if cfg["host"] != "smtp.example.com" {
		t.Errorf("Expected non-secret host to pass through, got %q", cfg["host"])
	}

	// Secrets left out keep their stored value; secrets sent are stored as sent, even when the
	// value looks like the redaction placeholder.
	if err := svc.SetProviderConfig(providerName, map[string]string{
		"host":     "smtp2.example.com",
		"password": "new-password",
	}); err != nil {
		t.Fatalf("SetProviderConfig failed: %v", err)
	}
	raw, err := svc.getRawProviderConfig(providerName)
	if err != nil {
		t.Fatalf("getRawProviderConfig failed: %v", err)
	}
	want := map[string]string{"api_key": "key-1", "password": "new-password", "secret_access_key": "aws-secret", "host": "smtp2.example.com"}
	for k, v := range want {
		if raw[k] != v {
			t.Errorf("Stored %s = %q; want %q", k, raw[k], v)
		}
	}

	if err := svc.SetProviderConfig(providerName, map[string]string{"password": redactedSecret, "api_key": ""}); err != nil {
		t.Fatalf("SetProviderConfig failed: %v", err)
	}
	raw, _ = svc.getRawProviderConfig(providerName)
	if raw["password"] != redactedSecret || raw["api_key"] != "" || raw["secret_access_key"] != "aws-secret" {
		t.Errorf("Expected a literal \"********\" stored, api_key cleared, and the AWS secret kept; got %v", raw)
	}

	// A provider with nothing stored yet, saved without secrets, stores none.
	if err := svc.SetProviderConfig("test_integration_secrets_empty", map[string]string{"host": "h"}); err != nil {
		t.Fatalf("SetProviderConfig failed: %v", err)
	}
	raw, _ = svc.getRawProviderConfig("test_integration_secrets_empty")
	for _, k := range secretConfigKeys {
		if v, ok := raw[k]; ok {
			t.Errorf("Expected no %s stored, got %q", k, v)
		}
	}
	if err := svc.SetProviderConfig("test_integration_secrets_nil", nil); err != nil {
		t.Errorf("Expected a nil config to be accepted, got %v", err)
	}
}
