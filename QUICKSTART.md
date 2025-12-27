# Quick Reference - New Features

## Structured Errors

Use the new `pkg/errors` package for better error handling:

```go
import "github.com/NVIDIA/cloud-native-stack/pkg/errors"

// Simple error
err := errors.New(errors.ErrCodeNotFound, "GPU not found")

// Wrap existing error
err := errors.Wrap(errors.ErrCodeInternal, "collection failed", originalErr)

// With context for debugging
err := errors.WrapWithContext(
    errors.ErrCodeTimeout,
    "GPU collection timed out",
    ctx.Err(),
    map[string]interface{}{
        "command": "nvidia-smi",
        "node":    nodeName,
        "timeout": "10s",
    },
)

// Error codes available:
// - ErrCodeNotFound
// - ErrCodeUnauthorized
// - ErrCodeTimeout
// - ErrCodeInternal
// - ErrCodeInvalidRequest
// - ErrCodeUnavailable
```

## Context Timeouts

All collectors and handlers now use proper timeouts:

```go
// In collectors
func (c *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    // Your collection logic here
}

// In HTTP handlers
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()
    
    // Your handler logic here
}
```

## Kubernetes Client

Always use the singleton client:

```go
import "github.com/NVIDIA/cloud-native-stack/pkg/collector/k8s"

// Get the shared client (created once, reused forever)
client, config, err := k8s.GetKubeClient()
if err != nil {
    return err
}

// Use the client
nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
```

## API Versioning

Clients can request specific API versions:

```bash
# Request v1 (default)
curl https://api.example.com/v1/recipe

# Request v2 when available
curl -H "Accept: application/vnd.nvidia.cns.v2+json" \
     https://api.example.com/v1/recipe
```

Response includes version header:
```
X-API-Version: v1
```

## Metrics

New Prometheus metrics are available at `/metrics`:

```promql
# Request rate by endpoint
rate(eidos_http_requests_total[5m])

# Error rate (4xx + 5xx)
rate(eidos_http_requests_total{status=~"[45].."}[5m])

# Request duration P95
histogram_quantile(0.95, rate(eidos_http_request_duration_seconds_bucket[5m]))

# In-flight requests
eidos_http_requests_in_flight

# Rate limit rejections
rate(eidos_rate_limit_rejects_total[5m])
```

## Testing with Race Detector

All tests now run with race detector:

```bash
# Run all tests
make test

# Run specific package
go test -race ./pkg/errors/

# Run with verbose output
go test -race -v ./...
```

## Response Writer

Middleware now uses a wrapper to prevent header-after-body bugs:

```go
// Internal use only - automatically handled by middleware
rw := newResponseWriter(w)
rw.WriteHeader(http.StatusOK)  // Can only be called once
rw.Write(body)                  // Headers must come before this
status := rw.Status()           // Get status code for logging/metrics
```

## Migration Checklist

- [ ] Update error handling to use `pkg/errors`
- [ ] Add timeouts to all new collectors
- [ ] Use `k8s.GetKubeClient()` instead of creating new clients
- [ ] Test code with `-race` flag
- [ ] Monitor new Prometheus metrics
- [ ] Update API documentation with versioning support

## Support

- **Questions**: Open a GitHub issue
- **Bugs**: Include error codes and context from logs
- **Metrics**: Check `/metrics` endpoint for observability data
