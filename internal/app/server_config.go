package app

import (
	"log"
	"os"
	"strconv"
)

// ServerConfig holds environment and network configuration for server mode.
type ServerConfig struct {
	Host string
	Port int
}

// LoadServerConfig parses server configuration from environment variables (LIBREDENTAL_HOST, LIBREDENTAL_PORT).
func LoadServerConfig() ServerConfig {
	serveHost := os.Getenv("LIBREDENTAL_HOST")
	if serveHost == "" {
		serveHost = "0.0.0.0"
	}
	servePort := 4242
	if portStr := os.Getenv("LIBREDENTAL_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
			servePort = p
		} else {
			log.Printf("Warning: invalid LIBREDENTAL_PORT %q, using default %d", portStr, servePort)
		}
	}
	return ServerConfig{
		Host: serveHost,
		Port: servePort,
	}
}
