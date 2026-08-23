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
			return make(map[string]string), nil // Return empty config if not found
		}
		return nil, fmt.Errorf("failed to get secret from keychain: %w", err)
	}

	var config map[string]string
	if err := json.Unmarshal([]byte(secretJSON), &config); err != nil {
		return nil, fmt.Errorf("failed to parse secret config: %w", err)
	}

	return config, nil
}

// SetProviderConfig encrypts and stores the configuration for a specific provider.
func (s *SecretsService) SetProviderConfig(providerName string, config map[string]string) error {
	key := fmt.Sprintf("provider_config_%s", providerName)

	bytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal secret config: %w", err)
	}

	if err := keyring.Set(keyringServiceName, key, string(bytes)); err != nil {
		return fmt.Errorf("failed to store secret in keychain: %w", err)
	}

	return nil
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
