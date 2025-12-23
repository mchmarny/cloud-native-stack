package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"
)

// setupRoutes configures all HTTP routes and middleware
func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Default handler
	mux.HandleFunc("/", s.handleDefault)

	// System endpoints (no rate limiting)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)

	// API endpoints with middleware
	mux.HandleFunc("/v1/recommendations", s.withMiddleware(s.handleRecommendations))

	return mux
}

func (s *Server) handleDefault(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handling default route",
		"path", r.URL.Path,
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
	)

	resp := struct {
		Name      string   `json:"name"`
		Version   string   `json:"version"`
		Ready     bool     `json:"ready"`
		Timestamp string   `json:"timestamp"`
		Routes    []string `json:"routes"`
	}{
		Name:      name,
		Version:   version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Routes: []string{
			"GET /v1/recommendations",
			"GET /health",
			"GET /ready",
		},
	}

	s.mu.RLock()
	resp.Ready = s.ready
	s.mu.RUnlock()

	serializers.RespondJSON(w, http.StatusOK, resp)
}
