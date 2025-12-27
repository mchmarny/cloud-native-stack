// Package logging provides structured logging utilities for Cloud Native Stack components.
//
// # Overview
//
// This package wraps the standard library slog package with CNS-specific defaults
// and conventions for consistent logging across all components. It supports
// environment-based log level configuration, module/version context injection,
// and automatic source location tracking for debug logs.
//
// # Features
//
//   - Structured JSON logging to stderr
//   - Environment-based log level configuration (LOG_LEVEL)
//   - Automatic module and version context
//   - Source location tracking for debug logs
//   - Flexible log level parsing
//   - Integration with standard library log package
//
// # Log Levels
//
// Supported log levels (case-insensitive):
//   - DEBUG: Detailed diagnostic information with source location
//   - INFO: General informational messages (default)
//   - WARN/WARNING: Warning messages for potentially problematic situations
//   - ERROR: Error messages for failures requiring attention
//
// # Usage
//
// Setting the default logger (recommended):
//
//	func main() {
//	    logging.SetDefaultStructuredLogger("eidos", "v1.0.0")
//	    defer slog.Info("application started")
//
//	    // Use slog as normal
//	    slog.Info("processing request", "id", "req-123")
//	    slog.Debug("detailed state", "data", complexObject)
//	    slog.Error("operation failed", "error", err)
//	}
//
// Creating a custom logger:
//
//	logger := logging.NewStructuredLogger("api-server", "v2.0.0", "debug")
//	logger.Info("server starting", "port", 8080)
//
// Setting explicit log level:
//
//	logging.SetDefaultStructuredLoggerWithLevel("cli", "v1.0.0", "warn")
//
// Converting standard library logger:
//
//	stdLogger := logging.NewLogLogger(slog.LevelInfo, false)
//	stdLogger.Println("legacy log message")
//
// # Environment Configuration
//
// The LOG_LEVEL environment variable controls logging verbosity:
//
//	LOG_LEVEL=debug eidos snapshot
//	LOG_LEVEL=error eidos-api-server
//
// If LOG_LEVEL is not set, defaults to INFO level.
//
// # Output Format
//
// All logs are written to stderr in JSON format:
//
//	{
//	    "time": "2025-01-15T10:30:00.123Z",
//	    "level": "INFO",
//	    "msg": "server started",
//	    "module": "api-server",
//	    "version": "v1.0.0",
//	    "port": 8080
//	}
//
// Debug logs include source location:
//
//	{
//	    "time": "2025-01-15T10:30:00.123Z",
//	    "level": "DEBUG",
//	    "source": {
//	        "function": "main.processRequest",
//	        "file": "server.go",
//	        "line": 45
//	    },
//	    "msg": "processing request",
//	    "module": "api-server",
//	    "version": "v1.0.0"
//	}
//
// # Best Practices
//
// 1. Set default logger early in main():
//
//	func main() {
//	    logging.SetDefaultStructuredLogger("myapp", version)
//	    defer slog.Info("application started")
//	    // ...
//	}
//
// 2. Include context in log messages:
//
//	slog.Info("request processed",
//	    "method", "GET",
//	    "path", "/api/v1/resources",
//	    "duration_ms", 125,
//	)
//
// 3. Use appropriate log levels:
//
//	slog.Debug("cache hit", "key", key)  // Development/troubleshooting
//	slog.Info("server started")          // Normal operations
//	slog.Warn("retry attempt 3")         // Potential issues
//	slog.Error("db connection failed")   // Errors requiring action
//
// 4. Log errors with context:
//
//	slog.Error("failed to process request",
//	    "error", err,
//	    "request_id", requestID,
//	    "retry_count", retries,
//	)
//
// # Integration
//
// This package is used by:
//   - pkg/api - API server logging
//   - pkg/cli - CLI command logging
//   - pkg/collector - Data collection logging
//   - pkg/snapshotter - Snapshot operation logging
//   - pkg/recommender - Recommendation generation logging
//
// All components share consistent logging format and configuration.
package logging
