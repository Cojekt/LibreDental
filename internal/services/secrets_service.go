package services

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
)

const keyringServiceName = "LibreDental"

// SecretsService manages secure storage of external credentials (like API keys)
// using the host OS's native keychain/credential vault.
type SecretsService struct{}

func NewSecretsService() *SecretsService {
	return &SecretsService{}
}

// GetProviderConfig retrieves and decrypts the configuration for a specific provider.
func (s *SecretsService) GetProviderConfig(providerName string) (map[string]string, error) {
	key := fmt.Sprintf("provider_config_%s", providerName)

	secretJSON, err := keyring.Get(keyringServiceName, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to get secret from keychain: %w", err)
	}

	var config map[string]string
	if err := json.Unmarshal([]byte(secretJSON), &config); err != nil {
		return nil, fmt.Errorf("failed to parse secret config: %w", err)
	}

	// Redact the API key for the frontend
	if apiKey, ok := config["api_key"]; ok && apiKey != "" {
		config["api_key"] = "********"
	}

	return config, nil
}

// SetProviderConfig encrypts and stores the configuration for a specific provider.
func (s *SecretsService) SetProviderConfig(providerName string, config map[string]string) error {
	key := fmt.Sprintf("provider_config_%s", providerName)

	// If the frontend sent back the redacted string, restore the real API key
	if config["api_key"] == "********" {
		oldConfig, err := s.getRawProviderConfig(providerName)
		if err != nil {
			return fmt.Errorf("failed to retrieve existing config to restore api_key: %w", err)
		}
		config["api_key"] = oldConfig["api_key"]
	}

	bytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal secret config: %w", err)
	}

	// Get existing secret first to restore if needed
	existingSecret, existingErr := keyring.Get(keyringServiceName, key)

	// Delete existing before set to avoid creating duplicate entries on Linux
	_ = keyring.Delete(keyringServiceName, key)

	if err := keyring.Set(keyringServiceName, key, string(bytes)); err != nil {
		if existingErr == nil {
			_ = keyring.Set(keyringServiceName, key, existingSecret)
		}
		return fmt.Errorf("failed to store secret in keychain: %w", err)
	}

	return nil
}

// getRawProviderConfig gets the unredacted config for internal use
func (s *SecretsService) getRawProviderConfig(providerName string) (map[string]string, error) {
	key := fmt.Sprintf("provider_config_%s", providerName)

	secretJSON, err := keyring.Get(keyringServiceName, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return make(map[string]string), nil
		}
		return nil, err
	}

	var config map[string]string
	if err := json.Unmarshal([]byte(secretJSON), &config); err != nil {
		return nil, err
	}
	return config, nil
}

// DeleteProviderConfig removes a provider's configuration from the keychain.
func (s *SecretsService) DeleteProviderConfig(providerName string) error {
	key := fmt.Sprintf("provider_config_%s", providerName)
	err := keyring.Delete(keyringServiceName, key)
	if err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("failed to delete secret from keychain: %w", err)
	}
	return nil
}
