package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/recommendation"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

const (
	name           = "eidos-server"
	versionDefault = "dev"
)

var (
	// overridden during build with ldflags
	version = versionDefault
	commit  = "unknown"
	date    = "unknown"
)

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	cfg := &Config{
		Address:         "",
		Port:            8080,
		RateLimit:       100, // 100 req/s
		RateLimitBurst:  200, // burst of 200
		CacheMaxAge:     300, // 5 minutes
		MaxBulkRequests: 100,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		LogLevel:        slog.LevelInfo,
	}

	// Override with environment variables if set
	if portStr := os.Getenv("PORT"); portStr != "" {
		var port int
		if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
			cfg.Port = port
		}
	}

	if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
		var level slog.Level
		if err := level.UnmarshalText([]byte(logLevelStr)); err == nil {
			cfg.LogLevel = level
		}
	}

	return cfg
}

// Server represents the HTTP server
type Server struct {
	config                *Config
	httpServer            *http.Server
	rateLimiter           *rate.Limiter
	mu                    sync.RWMutex
	recommendationHandler http.HandlerFunc
	ready                 bool
	logger                Logger
}

// NewServer creates a new server instance
func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	s := &Server{
		config:      config,
		rateLimiter: rate.NewLimiter(config.RateLimit, config.RateLimitBurst),
		logger:      NewLogger(slog.LevelInfo),
	}

	// Initialize recommendation handler
	rb := &recommendation.Builder{
		CacheTTL: time.Duration(config.CacheMaxAge) * time.Second,
	}
	s.recommendationHandler = rb.Handle

	// Setup HTTP server
	mux := s.setupRoutes()
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Address, config.Port),
		Handler:      mux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return s
}

// setupRoutes configures all HTTP routes and middleware
func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// System endpoints (no rate limiting)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)

	// API endpoints with middleware
	mux.HandleFunc("/v1/recommendations", s.withMiddleware(s.recommendationHandler))

	return mux
}

// writeError writes error response
func (s *Server) writeError(w http.ResponseWriter, r *http.Request, statusCode int,
	code, message string, retryable bool, details map[string]interface{}) {

	requestID, _ := r.Context().Value(contextKeyRequestID).(string)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	errResp := ErrorResponse{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
		Retryable: retryable,
	}

	serializers.RespondJSON(w, statusCode, errResp)
}

// SetReady marks the server as ready to serve traffic
func (s *Server) SetReady(ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = ready
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	s.SetReady(true)

	fmt.Printf("Starting server on %s\n", s.httpServer.Addr)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.SetReady(false)

	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	fmt.Println("Shutting down server...")
	return s.httpServer.Shutdown(shutdownCtx)
}

// Run starts the server with graceful shutdown handling
func Run() error {
	if err := RunWithConfig(DefaultConfig()); err != nil {
		slog.Error("error running server", slog.String("error", err.Error()))
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// RunWithConfig starts the server with custom configuration
func RunWithConfig(config *Config) error {
	slog.Info("starting server",
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("date", date))

	server := NewServer(config)

	slog.Info("server config",
		slog.String("address", server.httpServer.Addr),
		slog.Int("port", config.Port),
		slog.Any("rateLimit", config.RateLimit),
		slog.Int("rateLimitBurst", config.RateLimitBurst),
		slog.Int("maxBulkRequests", config.MaxBulkRequests),
		slog.Duration("readTimeout", config.ReadTimeout),
		slog.Duration("writeTimeout", config.WriteTimeout),
		slog.Duration("idleTimeout", config.IdleTimeout),
		slog.Duration("shutdownTimeout", config.ShutdownTimeout),
		slog.String("logLevel", config.LogLevel.String()),
	)

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Use errgroup for concurrent operations
	g, gctx := errgroup.WithContext(ctx)

	// Start HTTP server
	g.Go(func() error {
		return server.Start(gctx)
	})

	// Wait for completion or error
	if err := g.Wait(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	slog.Info("server stopped gracefully")
	return nil
}
