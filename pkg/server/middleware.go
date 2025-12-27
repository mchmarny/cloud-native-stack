package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// withMiddleware wraps handlers with common middleware
func (s *Server) withMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return s.metricsMiddleware(
		s.versionMiddleware(
			s.requestIDMiddleware(
				s.panicRecoveryMiddleware( // Recover first to prevent token waste on panics
					s.rateLimitMiddleware(
						s.loggingMiddleware(handler),
					),
				),
			),
		),
	)
}

// Middleware implementations

// versionMiddleware handles API version negotiation and sets version header
func (s *Server) versionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := negotiateAPIVersion(r)
		SetAPIVersionHeader(w, version)

		// Store version in context for handlers to access if needed
		ctx := context.WithValue(r.Context(), contextKeyAPIVersion, version)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// requestIDMiddleware extracts or generates request IDs
func (s *Server) requestIDMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Validate UUID format if provided
		if _, err := uuid.Parse(requestID); err != nil {
			requestID = uuid.New().String()
		}

		// Store in context and response header
		ctx := context.WithValue(r.Context(), contextKeyRequestID, requestID)
		w.Header().Set("X-Request-Id", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// rateLimitMiddleware implements rate limiting
func (s *Server) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.rateLimiter.Allow() {
			rateLimitRejects.Inc()
			retryAfterSeconds := "1"
			w.Header().Set("Retry-After", retryAfterSeconds)
			WriteError(w, r, http.StatusTooManyRequests, ErrCodeRateLimitExceeded,
				"Rate limit exceeded", true, map[string]interface{}{
					"limit": s.config.RateLimit,
					"burst": s.config.RateLimitBurst,
				})
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", int(s.config.RateLimit)))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", int(s.rateLimiter.Tokens())))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))

		next.ServeHTTP(w, r)
	}
}

// panicRecoveryMiddleware recovers from panics
func (s *Server) panicRecoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				panicRecoveries.Inc()
				var errMsg string
				switch v := err.(type) {
				case error:
					errMsg = v.Error()
				default:
					errMsg = fmt.Sprintf("%v", v)
				}
				slog.Error("panic recovered",
					"error", errMsg,
					"requestID", r.Context().Value(contextKeyRequestID),
					"path", r.URL.Path,
					"method", r.Method,
				)
				WriteError(w, r, http.StatusInternalServerError, ErrCodeInternalError,
					"Internal server error", true, nil)
			}
		}()
		next.ServeHTTP(w, r)
	}
}

// loggingMiddleware logs requests
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Context().Value(contextKeyRequestID)

		// Wrap response writer to track status code
		rw := newResponseWriter(w)

		slog.Debug("request started",
			"requestID", requestID,
			"method", r.Method,
			"path", r.URL.Path,
		)

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		slog.Debug("request completed",
			"requestID", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.Status(),
			"duration", duration.String(),
		)
	}
}
