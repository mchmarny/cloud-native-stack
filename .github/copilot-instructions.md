# Copilot Instructions for NVIDIA Cloud Native Stack

## Project Overview
NVIDIA Cloud Native Stack (CNS/Eidos) is a comprehensive toolkit for deploying, validating, and operating optimized AI workloads on GPU-accelerated Kubernetes clusters. It provides:
- **CLI Tool (eidos)** – System snapshot capture and configuration recipe generation
- **API Server** – REST API for recipe generation based on environment parameters
- **Agent** – Kubernetes Job for automated cluster configuration and optimization
- **Documentation** – Installation guides, Ansible playbooks, optimizations, and troubleshooting

The project is built in Go using cloud-native patterns and follows production-grade architectural principles.

## Architecture & Key Components

### Core Directories
- **cmd/** – Application entrypoints
  - `cmd/eidos/` – CLI application main entry
  - `cmd/eidos-api-server/` – HTTP API server main entry
- **pkg/** – Core Go packages organized by domain
  - `api/` – HTTP API server implementation
  - `cli/` – CLI command handlers (snapshot, recipe)
  - `collector/` – System data collectors (GPU, K8s, OS, SystemD) with factory pattern
  - `errors/` – Structured error types with error codes for observability
  - `logging/` – Structured logging with slog
  - `measurement/` – Data models for system measurements
  - `recipe/` – Recipe generation logic with overlay-based configuration
  - `serializer/` – JSON/YAML/Table output formatting
  - `server/` – HTTP server with middleware (rate limiting, metrics, logging)
  - `snapshotter/` – Orchestrates parallel data collection
- **docs/** – Comprehensive documentation
  - `install-guides/` – Platform-specific installation instructions
  - `playbooks/` – Ansible automation for deployment
  - `optimizations/` – Hardware-specific tuning guides
  - `troubleshooting/` – Common issues and solutions
- **deployments/** – Kubernetes manifests
  - `deployments/eidos-agent/` – Job manifests for agent deployment
- **examples/** – Configuration examples for hardware (GB200, H100, etc.)
- **api/** – OpenAPI/YAML specifications for API contracts
- **tools/** – Build and release utility scripts
- **infra/** – Terraform configuration for infrastructure

## Developer Workflows

### Local Development
```bash
# Setup and verify tooling
make info                    # Show versions (Go, linter, goreleaser)
make tidy                    # Format code and update dependencies
make upgrade                 # Upgrade all dependencies

# Development
make lint                    # Run golangci-lint and yamllint
make test                    # Run tests with race detector
make scan                    # Static analysis and vulnerability scan
make qualify                 # Run test + lint + scan
make build                   # Build with goreleaser (snapshot)
make server                  # Run API server locally with DEBUG logging

# Documentation
make docs                    # Serve Go docs on http://localhost:6060
```

### Testing Strategy
- **Unit Tests**: All packages have `*_test.go` files
- **Race Detector**: Always enabled via `make test` (runs `-race` flag)
- **Table-Driven Tests**: Use for comprehensive coverage
- **Integration Tests**: For K8s client, collectors
- **Test Commands**:
  ```bash
  go test ./...                    # All tests
  go test -race ./pkg/errors/      # Specific package with race detector
  go test -short ./...             # Skip long tests
  go test -v ./...                 # Verbose output
  ```

### Release Process
1. Tag version: `git tag v0.x.x && git push --tags`
2. GoReleaser builds binaries and container images
3. Ko builds multi-platform container images
4. GitHub Actions publishes to ghcr.io
5. Attestations generated with SLSA provenance

## Patterns & Conventions

### Go Architecture Patterns
1. **Functional Options Pattern** – Used throughout for configuration
   ```go
   builder := recipe.NewBuilder(
       recipe.WithVersion(version),
   )
   server := server.New(
       server.WithName("eidos-api-server"),
       server.WithVersion(version),
   )
   ```

2. **Factory Pattern** – For collectors
   ```go
   factory := collector.NewDefaultFactory(
       collector.WithSystemDServices([]string{"containerd.service"}),
   )
   gpuCollector := factory.CreateGPUCollector()
   ```

3. **Builder Pattern** – For measurements
   ```go
   measurement.NewMeasurement(measurement.TypeK8s).
       WithSubtype(subtype).
       Build()
   ```

4. **Singleton Pattern** – For K8s client (uses sync.Once)
   ```go
   client, config, err := k8s.GetKubeClient()
   ```

5. **Middleware Chain** – For HTTP server
   ```go
   // Order: metrics → version → requestID → panic → rateLimit → logging → handler
   ```

### Error Handling
**Always use structured errors from `pkg/errors`:**
```go
import "github.com/NVIDIA/cloud-native-stack/pkg/errors"

// Simple error
return errors.New(errors.ErrCodeNotFound, "GPU not found")

// Wrap existing error
return errors.Wrap(errors.ErrCodeInternal, "collection failed", err)

// With context for debugging
return errors.WrapWithContext(
    errors.ErrCodeTimeout,
    "operation timed out",
    ctx.Err(),
    map[string]interface{}{
        "component": "gpu-collector",
        "timeout": "10s",
    },
)
```

**Error Codes:**
- `ErrCodeNotFound` – Resource not found
- `ErrCodeUnauthorized` – Auth/authz failure
- `ErrCodeTimeout` – Operation exceeded time limit
- `ErrCodeInternal` – Internal system error
- `ErrCodeInvalidRequest` – Malformed input
- `ErrCodeUnavailable` – Service temporarily unavailable

### Context & Timeouts
**Always use context with timeouts to prevent resource leaks:**
```go
// In collectors
func (c *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    // Collection logic here
}

// In HTTP handlers
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()
    
    // Handler logic here
}
```

### Logging Standards
**Use structured logging with slog:**
```go
slog.Debug("request started",
    "requestID", requestID,
    "method", r.Method,
    "path", r.URL.Path,
)

slog.Error("operation failed",
    "error", err,
    "component", "gpu-collector",
    "node", nodeName,
)
```

**Log Levels:**
- `Debug` – Detailed diagnostic info (enabled with `--debug`)
- `Info` – General informational messages (default)
- `Warn` – Warning conditions
- `Error` – Error conditions requiring attention

### Concurrency Patterns
**Use errgroup for parallel operations:**
```go
g, ctx := errgroup.WithContext(ctx)

g.Go(func() error {
    return collector1.Collect(ctx)
})
g.Go(func() error {
    return collector2.Collect(ctx)
})

if err := g.Wait(); err != nil {
    return fmt.Errorf("collection failed: %w", err)
}
```

**Protect shared state with mutexes:**
```go
var mu sync.Mutex
g.Go(func() error {
    result := compute()
    mu.Lock()
    results = append(results, result)
    mu.Unlock()
    return nil
})
```

### HTTP Server Best Practices
1. **Response Writer Wrapper** – Use `newResponseWriter()` to track status codes
2. **Middleware Order** – Metrics → Version → RequestID → Panic Recovery → Rate Limit → Logging → Handler
3. **Timeouts** – All handlers use 30-second request timeout
4. **Rate Limiting** – 100 req/s with burst of 200
5. **Graceful Shutdown** – Signal handling with timeout
6. **Metrics** – RED metrics (Rate, Errors, Duration) exposed at `/metrics`

### API Versioning
**Support version negotiation via Accept header:**
```bash
# Default (v1)
curl https://api.example.com/v1/recipe

# Request specific version
curl -H "Accept: application/vnd.nvidia.cns.v2+json" \
     https://api.example.com/v1/recipe
```

Response includes `X-API-Version` header.

## Code Quality Standards

### Linting
- **golangci-lint** configuration in `.golangci.yaml`
- Enabled linters: staticcheck, errcheck, errorlint, gosec, misspell, etc.
- Run `make lint` before committing

### Testing Requirements
- All new code must have tests
- Use table-driven tests for multiple scenarios
- Test with `-race` flag to catch concurrency issues
- Aim for >80% code coverage
- Mock external dependencies (K8s API, nvidia-smi, etc.)

### Code Review Checklist
- [ ] Tests pass with race detector (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Vulnerability scan passes (`make scan`)
- [ ] Proper error handling with structured errors
- [ ] Context timeouts for long operations
- [ ] Structured logging with appropriate levels
- [ ] Documentation updated (godoc, README, etc.)
- [ ] Breaking changes noted in commit message

## Integration Points

### Kubernetes
- **Client**: Use singleton `k8s.GetKubeClient()` for connection reuse
- **Deployment**: Manifests in `deployments/eidos-agent/`
- **RBAC**: Defined in `1-deps.yaml` (ServiceAccount, Role, RoleBinding)
- **Security**: Pod security standards, minimal privileges

### NVIDIA Operators
- **GPU Operator** – GPU driver, device plugin, DCGM, MIG manager
- **Network Operator** – RDMA, SR-IOV, OFED drivers
- **NIM Operator** – NVIDIA Inference Microservices
- **Nsight Operator** – Profiling and debugging

### Container Images
- **Base Images**: NVIDIA CUDA runtime for GPU support
- **Registry**: ghcr.io/mchmarny (replace with nvidia org in production)
- **Multi-Arch**: linux/amd64, linux/arm64
- **Build Tool**: Ko for efficient container builds
- **Attestations**: SLSA provenance for supply chain security

### Observability
- **Metrics**: Prometheus format at `/metrics`
  - `eidos_http_requests_total{method,path,status}`
  - `eidos_http_request_duration_seconds{method,path}`
  - `eidos_http_requests_in_flight`
  - `eidos_rate_limit_rejects_total`
  - `eidos_panic_recoveries_total`
- **Logging**: Structured JSON logs to stderr
- **Tracing**: Ready for OpenTelemetry integration

## Common Development Tasks

### Adding a New Collector
1. Create package in `pkg/collector/<name>/`
2. Implement `Collector` interface:
   ```go
   type Collector interface {
       Collect(ctx context.Context) (*measurement.Measurement, error)
   }
   ```
3. Add factory method in `pkg/collector/factory.go`
4. Register in `DefaultFactory`
5. Add to `snapshotter.Measure()` parallel collection
6. Write tests with mocks

### Adding a New CLI Command
1. Create command in `pkg/cli/<command>.go`
2. Use urfave/cli/v3 framework
3. Add to `Commands` slice in `pkg/cli/root.go`
4. Implement with context timeout
5. Use structured errors for failures
6. Support output formats (JSON, YAML, table)

### Adding a New API Endpoint
1. Create handler in appropriate package
2. Register route in `pkg/api/server.go`
3. Add middleware for protection
4. Implement request timeout (30s)
5. Return structured errors on failure
6. Add metrics tracking
7. Update API specification in `api/`
8. Write integration tests

### Updating Recipe Data
1. Edit `pkg/recipe/data/data-v1.yaml`
2. Add base measurements or overlays
3. Use Query matching for overlays:
   ```yaml
   key:
     os: ubuntu
     gpu: h100
     intent: training
   ```
4. Test with `eidos recipe` CLI
5. Validate YAML structure

## Troubleshooting & Support

### Common Issues
- **K8s Connection**: Check kubeconfig at `~/.kube/config` or `KUBECONFIG` env
- **GPU Detection**: Requires nvidia-smi in PATH
- **Linter Errors**: Use `errors.Is()` for error comparison, add `return` after `t.Fatal()`
- **Race Conditions**: Run tests with `-race` flag to detect
- **Build Failures**: Run `make tidy` to fix Go module issues

### Debugging Tips
```bash
# Enable debug logging
eidos --debug snapshot

# Run server with debug logs
LOG_LEVEL=debug make server

# Check race conditions
go test -race -run TestSpecificTest ./pkg/...

# Profile performance
go test -cpuprofile cpu.prof -memprofile mem.prof
go tool pprof cpu.prof
```

### Getting Help
- **Documentation**: `docs/` directory
- **Contributing**: See [CONTRIBUTING.md](../CONTRIBUTING.md)
- **Issues**: Open on GitHub with error codes and logs
- **Discussions**: Use GitHub Discussions for questions

## Security Considerations

### Input Validation
- Validate all CLI flags and API parameters
- Use allowlists for enum values
- Sanitize file paths to prevent traversal
- Validate versions against regex patterns

### Container Security
- Run as non-root when possible
- Use read-only root filesystem
- Drop unnecessary capabilities
- Apply seccomp and AppArmor profiles
- Scan images for vulnerabilities (`make scan`)

### API Security
- Rate limiting (100 req/s)
- Request ID tracking for audit trails
- Timeout all requests (30s max)
- Validate Accept headers for version negotiation
- Future: Add authentication/authorization

## Performance Optimization

### Best Practices
1. **Connection Pooling**: K8s client uses singleton pattern
2. **Parallel Collection**: Collectors run concurrently with errgroup
3. **Context Timeouts**: Prevent resource leaks
4. **Response Writer**: Track status without buffering
5. **Metrics**: Low overhead with Prometheus client

### Profiling
```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Trace execution
go test -trace=trace.out
go tool trace trace.out
```

## Examples & References

### Example: New Collector
```go
package mycollector

import (
    "context"
    "github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

type Collector struct{}

func (c *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    data := make(map[string]measurement.Reading)
    data["key"] = measurement.Str("value")
    
    return &measurement.Measurement{
        Type: measurement.Type("MyType"),
        Subtypes: []measurement.Subtype{{
            Name: "subtype",
            Data: data,
        }},
    }, nil
}
```

### Example: CLI Command
```go
func newCommand() *cli.Command {
    return &cli.Command{
        Name:  "mycommand",
        Usage: "Description",
        Flags: []cli.Flag{
            outputFlag,
            formatFlag,
        },
        Action: func(ctx context.Context, cmd *cli.Command) error {
            ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
            defer cancel()
            
            // Implementation
            return nil
        },
    }
}
```

---

## Quick Reference Links
- **Main README**: [../README.md](../README.md)
- **Contributing Guide**: [../CONTRIBUTING.md](../CONTRIBUTING.md)
- **Playbooks**: [../docs/playbooks/readme.md](../docs/playbooks/readme.md)
- **API Specification**: [../api/eidos/v1/api-server-v1.yaml](../api/eidos/v1/api-server-v1.yaml)
- **GoDoc**: Run `make docs` and visit http://localhost:6060
