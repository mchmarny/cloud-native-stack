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

	"github.com/NVIDIA/cloud-native-stack/pkg/logging"
	"github.com/NVIDIA/cloud-native-stack/pkg/recommendation"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

const (
	name           = "eidos-server"
	versionDefault = "dev"
)

var (
	// overridden during build with ldflags to reflect actual version info
	// e.g., -X "github.com/NVIDIA/cloud-native-stack/pkg/server.version=1.0.0"
	version = versionDefault
	commit  = "unknown"
	date    = "unknown"
)

// Server represents the HTTP server for handling requests.
type Server struct {
	config                *Config
	httpServer            *http.Server
	rateLimiter           *rate.Limiter
	mu                    sync.RWMutex
	recommendationBuilder *recommendation.Builder
	ready                 bool
}

// NewServer creates a new server instance with the given configuration.
func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	s := &Server{
		config:      config,
		rateLimiter: rate.NewLimiter(config.RateLimit, config.RateLimitBurst),
	}

	// Initialize recommendation handler
	rb := &recommendation.Builder{
		CacheTTL: time.Duration(config.CacheMaxAge) * time.Second,
	}
	s.recommendationBuilder = rb

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

// SetReady marks the server as ready to serve traffic or not.
func (s *Server) SetReady(ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = ready
}

// Start starts the HTTP server and listens for incoming requests.
func (s *Server) Start(ctx context.Context) error {
	s.SetReady(true)

	fmt.Printf("starting server on %s\n", s.httpServer.Addr)

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

// Shutdown gracefully shuts down the server within the given context.
func (s *Server) Shutdown(ctx context.Context) error {
	s.SetReady(false)

	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	fmt.Println("shutting down server...")
	return s.httpServer.Shutdown(shutdownCtx)
}

// Run starts the server with graceful shutdown handling using default configuration
// and logs any errors encountered during execution.
func Run() error {
	if err := RunWithConfig(DefaultConfig()); err != nil {
		slog.Error("error running server", slog.String("error", err.Error()))
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// RunWithConfig starts the server with custom configuration and graceful shutdown handling.
func RunWithConfig(config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	// Initialize logger
	logging.SetDefaultStructuredLoggerWithLevel(name, version, config.LogLevel)
	slog.Debug("starting",
		"name", name,
		"version", version,
		"commit", commit,
		"date", date)

	server := NewServer(config)

	slog.Debug("server config",
		slog.String("address", server.httpServer.Addr),
		slog.Int("port", config.Port),
		slog.Any("rateLimit", config.RateLimit),
		slog.Int("rateLimitBurst", config.RateLimitBurst),
		slog.Int("maxBulkRequests", config.MaxBulkRequests),
		slog.Duration("readTimeout", config.ReadTimeout),
		slog.Duration("writeTimeout", config.WriteTimeout),
		slog.Duration("idleTimeout", config.IdleTimeout),
		slog.Duration("shutdownTimeout", config.ShutdownTimeout),
		slog.String("logLevel", config.LogLevel),
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

	slog.Debug("server stopped gracefully")
	return nil
}
