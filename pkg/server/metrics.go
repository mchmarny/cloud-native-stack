package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eidos_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "eidos_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "eidos_http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	// Rate limiting metrics
	rateLimitRejects = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "eidos_rate_limit_rejects_total",
			Help: "Total number of requests rejected due to rate limiting",
		},
	)

	// Panic recovery metrics
	panicRecoveries = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "eidos_panic_recoveries_total",
			Help: "Total number of panics recovered in HTTP handlers",
		},
	)
)

// metricsMiddleware instruments HTTP requests with Prometheus metrics.
// It tracks request rate, errors, and duration (RED metrics) for observability.
func (s *Server) metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		// Wrap response writer to capture status code
		wrapped := newResponseWriter(w)

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		path := r.URL.Path
		method := r.Method
		status := strconv.Itoa(wrapped.Status())

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}
