package app_test

import (
	"testing"

	"github.com/LibreDental/libredental/internal/app"
)

func TestLoadServerConfig_Defaults(t *testing.T) {
	t.Setenv("LIBREDENTAL_HOST", "")
	t.Setenv("LIBREDENTAL_PORT", "")

	cfg := app.LoadServerConfig()
	if cfg.Host != "0.0.0.0" {
		t.Errorf("Expected default host 0.0.0.0, got %q", cfg.Host)
	}
	if cfg.Port != 4242 {
		t.Errorf("Expected default port 4242, got %d", cfg.Port)
	}
}

func TestLoadServerConfig_Custom(t *testing.T) {
	t.Setenv("LIBREDENTAL_HOST", "127.0.0.1")
	t.Setenv("LIBREDENTAL_PORT", "8080")

	cfg := app.LoadServerConfig()
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %q", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Port)
	}
}

func TestLoadServerConfig_InvalidPortFallback(t *testing.T) {
	t.Setenv("LIBREDENTAL_PORT", "invalid")

	cfg := app.LoadServerConfig()
	if cfg.Port != 4242 {
		t.Errorf("Expected fallback port 4242 for invalid input, got %d", cfg.Port)
	}
}
