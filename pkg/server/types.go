package server

import (
	"log/slog"
	"time"

	"golang.org/x/time/rate"
)

// Domain types matching OpenAPI spec schemas

// ErrorResponse represents error responses as per OpenAPI spec
type ErrorResponse struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RequestID string                 `json:"requestId"`
	Timestamp time.Time              `json:"timestamp"`
	Retryable bool                   `json:"retryable"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Reason    string    `json:"reason,omitempty"`
}

// Config holds server configuration
type Config struct {
	// Server configuration
	Address string
	Port    int

	// Rate limiting configuration
	RateLimit      rate.Limit // requests per second
	RateLimitBurst int        // burst size

	// Cache configuration
	CacheMaxAge int // seconds

	// Request limits
	MaxBulkRequests int

	// Timeouts
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	// Logging
	LogLevel slog.Level
}
