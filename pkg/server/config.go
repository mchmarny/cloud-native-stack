package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/time/rate"
)

// Config holds server configuration
type Config struct {
	// Server identity
	Name    string
	Version string

	// Additional Handlers to be added to the server
	Handlers map[string]http.HandlerFunc

	// Server configuration
	Address string
	Port    int

	// Rate limiting configuration
	RateLimit      rate.Limit // requests per second
	RateLimitBurst int        // burst size

	// Request limits
	MaxBulkRequests int

	// Timeouts
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// NewConfig returns a new Config with sensible defaults.
// Use this when you want to customize config programmatically.
func NewConfig() *Config {
	return parseConfig()
}

// parseConfig returns sensible defaults
func parseConfig() *Config {
	cfg := &Config{
		Name:            "server",
		Version:         "undefined",
		Address:         "",
		Port:            8080,
		RateLimit:       100, // 100 req/s
		RateLimitBurst:  200, // burst of 200
		MaxBulkRequests: 100,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}

	// Override with environment variables if set
	if portStr := os.Getenv("PORT"); portStr != "" {
		var port int
		if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
			cfg.Port = port
		}
	}

	// Allow customization of shutdown timeout to match K8s eviction grace period
	if shutdownStr := os.Getenv("SHUTDOWN_TIMEOUT_SECONDS"); shutdownStr != "" {
		var seconds int
		if _, err := fmt.Sscanf(shutdownStr, "%d", &seconds); err == nil && seconds > 0 {
			cfg.ShutdownTimeout = time.Duration(seconds) * time.Second
		}
	}

	return cfg
}
