package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name:   "custom config",
			config: DefaultConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.config)
			assert.NotNil(t, server)
			assert.NotNil(t, server.rateLimiter)
			assert.NotNil(t, server.logger)
			assert.NotNil(t, server.validator)
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	server := NewServer(DefaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp HealthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp.Status)
	assert.False(t, resp.Timestamp.IsZero())
}

func TestReadyEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		ready      bool
		wantStatus int
	}{
		{
			name:       "ready",
			ready:      true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "not ready",
			ready:      false,
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(DefaultConfig())
			server.SetReady(tt.ready)

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			w := httptest.NewRecorder()

			server.handleReady(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestGetRecommendations(t *testing.T) {
	server := NewServer(DefaultConfig())

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantError  bool
	}{
		{
			name:       "valid request with defaults",
			query:      "",
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "valid request with all parameters",
			query:      "?osFamily=Ubuntu&osVersion=24.04&kernel=6.8&environment=EKS&kubernetes=1.33&gpu=H100&intent=training",
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "invalid osFamily",
			query:      "?osFamily=Windows",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid osVersion format",
			query:      "?osVersion=24",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid kernel format",
			query:      "?kernel=6",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid environment",
			query:      "?environment=AKS",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid kubernetes format",
			query:      "?kubernetes=2.0",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid gpu",
			query:      "?gpu=V100",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid intent",
			query:      "?intent=gaming",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/recommendations"+tt.query, nil)
			w := httptest.NewRecorder()

			handler := server.withMiddleware(server.handleGetRecommendations)
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.NotEmpty(t, w.Header().Get("X-Request-Id"))

			if tt.wantError {
				var errResp ErrorResponse
				err := json.NewDecoder(w.Body).Decode(&errResp)
				require.NoError(t, err)
				assert.NotEmpty(t, errResp.Code)
				assert.NotEmpty(t, errResp.Message)
				assert.NotEmpty(t, errResp.RequestID)
			} else {
				var resp RecommendationResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)
				assert.NotEmpty(t, resp.PayloadVersion)
				assert.NotEmpty(t, resp.Measurements)
				assert.False(t, resp.GeneratedAt.IsZero())
				assert.NotEmpty(t, w.Header().Get("Cache-Control"))
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
			}
		})
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	server := NewServer(DefaultConfig())

	tests := []struct {
		name        string
		headerValue string
		expectNewID bool
	}{
		{
			name:        "no header provided",
			headerValue: "",
			expectNewID: true,
		},
		{
			name:        "valid UUID provided",
			headerValue: "550e8400-e29b-41d4-a716-446655440000",
			expectNewID: false,
		},
		{
			name:        "invalid UUID provided",
			headerValue: "not-a-uuid",
			expectNewID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.headerValue != "" {
				req.Header.Set("X-Request-Id", tt.headerValue)
			}
			w := httptest.NewRecorder()

			handler := server.requestIDMiddleware(func(w http.ResponseWriter, r *http.Request) {
				requestID := r.Context().Value(contextKeyRequestID).(string)
				assert.NotEmpty(t, requestID)
				w.WriteHeader(http.StatusOK)
			})

			handler.ServeHTTP(w, req)

			responseID := w.Header().Get("X-Request-Id")
			assert.NotEmpty(t, responseID)

			if tt.expectNewID {
				assert.NotEqual(t, tt.headerValue, responseID)
			} else {
				assert.Equal(t, tt.headerValue, responseID)
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	config := DefaultConfig()
	config.RateLimit = 2
	config.RateLimitBurst = 2

	server := NewServer(config)

	handler := server.rateLimitMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestPanicRecoveryMiddleware(t *testing.T) {
	server := NewServer(DefaultConfig())

	handler := server.panicRecoveryMiddleware(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), contextKeyRequestID, "test-id"))
	w := httptest.NewRecorder()

	require.NotPanics(t, func() {
		handler.ServeHTTP(w, req)
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestValidateRecommendationRequest(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		req     RecommendationRequest
		wantErr bool
	}{
		{
			name: "valid request with ALL",
			req: RecommendationRequest{
				OSFamily:    defaultQueryValue,
				OSVersion:   defaultQueryValue,
				Kernel:      defaultQueryValue,
				Environment: defaultQueryValue,
				Kubernetes:  defaultQueryValue,
				GPU:         defaultQueryValue,
				Intent:      defaultQueryValue,
			},
			wantErr: false,
		},
		{
			name: "valid request with specific values",
			req: RecommendationRequest{
				OSFamily:    "Ubuntu",
				OSVersion:   "24.04",
				Kernel:      "6.8.0",
				Environment: "EKS",
				Kubernetes:  "1.33",
				GPU:         "H100",
				Intent:      "training",
			},
			wantErr: false,
		},
		{
			name: "invalid osFamily",
			req: RecommendationRequest{
				OSFamily: "Windows",
			},
			wantErr: true,
		},
		{
			name: "invalid osVersion pattern",
			req: RecommendationRequest{
				OSFamily:  "Ubuntu",
				OSVersion: "24",
			},
			wantErr: true,
		},
		{
			name: "invalid kernel pattern",
			req: RecommendationRequest{
				OSFamily: "Ubuntu",
				Kernel:   "6",
			},
			wantErr: true,
		},
		{
			name: "invalid environment",
			req: RecommendationRequest{
				OSFamily:    "Ubuntu",
				Environment: "AKS",
			},
			wantErr: true,
		},
		{
			name: "invalid kubernetes pattern",
			req: RecommendationRequest{
				OSFamily:   "Ubuntu",
				Kubernetes: "2.0",
			},
			wantErr: true,
		},
		{
			name: "invalid gpu",
			req: RecommendationRequest{
				OSFamily: "Ubuntu",
				GPU:      "V100",
			},
			wantErr: true,
		},
		{
			name: "invalid intent",
			req: RecommendationRequest{
				OSFamily: "Ubuntu",
				Intent:   "gaming",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set defaults for unset fields
			if tt.req.OSFamily == "" {
				tt.req.OSFamily = defaultQueryValue
			}
			if tt.req.OSVersion == "" {
				tt.req.OSVersion = defaultQueryValue
			}
			if tt.req.Kernel == "" {
				tt.req.Kernel = defaultQueryValue
			}
			if tt.req.Environment == "" {
				tt.req.Environment = defaultQueryValue
			}
			if tt.req.Kubernetes == "" {
				tt.req.Kubernetes = defaultQueryValue
			}
			if tt.req.GPU == "" {
				tt.req.GPU = defaultQueryValue
			}
			if tt.req.Intent == "" {
				tt.req.Intent = defaultQueryValue
			}

			err := validator.ValidateRecommendationRequest(&tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServerLifecycle(t *testing.T) {
	config := DefaultConfig()
	config.Port = 0

	server := NewServer(config)

	server.SetReady(true)
	assert.True(t, server.ready)

	server.SetReady(false)
	assert.False(t, server.ready)
}

func TestGenerateRecommendation(t *testing.T) {
	server := NewServer(DefaultConfig())

	req := &RecommendationRequest{
		OSFamily:   "Ubuntu",
		OSVersion:  "24.04",
		Kubernetes: "1.33",
		GPU:        "H100",
		Intent:     "training",
	}

	resp := server.generateRecommendation(req)

	assert.Equal(t, *req, resp.Request)
	assert.NotEmpty(t, resp.PayloadVersion)
	assert.NotEmpty(t, resp.MatchedRuleID)
	assert.False(t, resp.GeneratedAt.IsZero())
	assert.NotEmpty(t, resp.Measurements)
	assert.Greater(t, len(resp.Measurements), 0)
}

func TestPayloadVersionValidation(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{
			name:    "valid format",
			version: "2025.12.0",
			wantErr: false,
		},
		{
			name:    "valid single digit month",
			version: "2024.1.15",
			wantErr: false,
		},
		{
			name:    "invalid format - no patch",
			version: "2025.12",
			wantErr: true,
		},
		{
			name:    "invalid format - short year",
			version: "25.12.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &RecommendationRequest{
				OSFamily:                defaultQueryValue,
				OSVersion:               defaultQueryValue,
				Kernel:                  defaultQueryValue,
				Environment:             defaultQueryValue,
				Kubernetes:              defaultQueryValue,
				GPU:                     defaultQueryValue,
				Intent:                  defaultQueryValue,
				PayloadVersionRequested: &tt.version,
			}

			err := validator.ValidateRecommendationRequest(req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func BenchmarkGetRecommendations(b *testing.B) {
	server := NewServer(DefaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/v1/recommendations?osFamily=Ubuntu&osVersion=24.04&kubernetes=1.33", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler := server.withMiddleware(server.handleGetRecommendations)
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkValidation(b *testing.B) {
	validator := NewValidator()

	req := &RecommendationRequest{
		OSFamily:    "Ubuntu",
		OSVersion:   "24.04",
		Kernel:      "6.8.0",
		Environment: "EKS",
		Kubernetes:  "1.33",
		GPU:         "H100",
		Intent:      "training",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateRecommendationRequest(req)
	}
}
