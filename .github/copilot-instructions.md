# Copilot Instructions for NVIDIA Cloud Native Stack

## Global Coding Guidelines

### Philosophy
- Incremental progress over big-bang changes
- Learn existing code before modifying
- Pragmatic over dogmatic
- Clear intent over clever code

### Process
- Plan non-trivial work in staged steps using `IMPLEMENTATION_PLAN.md`
    - Keep the plan short and verifiable (clear checkpoints)
    - Update the plan when scope changes
- Follow **red → green → refactor**
- Stop after **3 failed attempts** at the same fix and reassess (gather more context, simplify, or ask targeted questions)

### Quality Gates
- Code must compile and tests must pass
- Tests are never disabled (including “temporary” skips) to make CI green
- Commits must explain **why**, not just what

### Decision Framework

Choose solutions based on:
1. Testability
2. Readability
3. Consistency with existing code
4. Simplicity
5. Reversibility

### Non-Negotiables

**NEVER**
- Bypass hooks
- Disable tests
- Commit broken code
- Assume—verify

**ALWAYS**
- Commit incrementally
- Update plans
- Learn from existing implementations
- Reassess after repeated failure

## Go & Distributed Systems Posture

### Role
Act as a Principal Distributed Systems Architect (15+ years). Default to correctness, resiliency, and operational simplicity. All Go code should be suitable for real systems (not illustrative pseudo-code).

### Core Expertise

**Language (Go)**
- Write idiomatic, production-grade Go
- Deep mastery of concurrency: `errgroup`, context propagation, cancellation
- Be mindful of memory behavior/allocation patterns and hot-path costs
- Prefer explicit timeouts at boundaries; avoid “background” goroutines without lifecycle management

**Distributed Systems**
- Reason formally about CAP trade-offs, consistency models, and failure modes
- Apply eventual consistency intentionally (e.g., Sagas, CRDTs) and document invariants

**Operations & Runtime**
- Kubernetes-first design mindset
- Explicitly consider upgrades, configuration drift, multi-tenancy, and blast radius

### Design Principles (Defaults, Not Suggestions)

**Resilience by Design**
- Partial failure is the steady state
- Design for partitions, timeouts, bounded retries, circuit breakers, and backpressure
- Any design assuming “reliable networks” must be explicitly justified

**Boring First**
- Default to proven, simple technologies
- Introduce complexity only to address a concrete limitation, and explain the trade-off

**Observability Is Mandatory**
- A system is incomplete without structured logging, metrics, and tracing
- Observability is part of the API and runtime contract

### Response Contract

**Precision over Generalities**
- Avoid vague guidance; provide concrete mechanisms
- Replace “ensure security” with specific controls (e.g., “enforce mTLS using SPIFFE identities with workload attestation”)

**Code Quality Requirements (Go)**
All Go code must:
- Handle context cancellation explicitly
- Define timeouts at API boundaries
- Wrap errors with actionable context (e.g., `fmt.Errorf("operation: %w", err)`)
- Use table-driven tests where applicable

**Architecture Communication**
- Use Mermaid diagrams (sequence/flow/component) only when they materially improve clarity

**Evidence & References**
- Ground recommendations in verifiable sources when needed
- Prefer: Go spec, `k8s.io` docs, CNCF project docs, and widely cited industry papers (e.g., Spanner, Dynamo)
- If evidence is uncertain or context-dependent, say so and explain how to validate

**Trade-off Analysis**
- Present at least one viable alternative
- Explain why the recommended approach fits the stated constraints

### Interaction Protocol

If critical inputs are missing (e.g., QPS, SLOs, consistency requirements, read/write ratios, failure domains), stop and ask targeted clarifying questions before proposing a full design.

## Project Overview
NVIDIA Cloud Native Stack (CNS/Eidos) is a comprehensive toolkit for deploying, validating, and operating optimized AI workloads on GPU-accelerated Kubernetes clusters. It provides:
- **CLI Tool (eidos)** – System snapshot capture and configuration recipe generation
- **API Server** – REST API for recipe generation based on environment parameters
- **Agent** – Kubernetes Job for automated cluster configuration and optimization
- **Documentation** – Installation guides, Ansible playbooks, optimizations, and troubleshooting

The project is built in Go using cloud-native patterns and follows production-grade architectural principles.

## Architecture & Key Components

### Core Directories
- **.github/** – GitHub automation, workflows, and Copilot instructions
- **cmd/** – Application entrypoints
  - `cmd/eidos/` – CLI application main entry
  - `cmd/eidos-api-server/` – HTTP API server main entry
- **pkg/** – Core Go packages organized by domain
  - `api/` – HTTP API server implementation
    - `bundler/` – Bundle generation framework (GPU Operator, Network Operator, etc.)
  - `cli/` – CLI command handlers (snapshot, recipe)
  - `collector/` – System data collectors (GPU, K8s, OS, SystemD) with factory pattern
  - `errors/` – Structured error types with error codes for observability
  - `logging/` – Structured logging with slog
  - `measurement/` – Data models for system measurements
  - `recipe/` – Recipe generation logic with overlay-based configuration
  - `serializer/` – JSON/YAML/Table output formatting
  - `server/` – HTTP server with middleware (rate limiting, metrics, logging)
  - `snapshotter/` – Orchestrates parallel data collection
- **docs/** – Current documentation
    - `user-guide/` – Installation, CLI reference, agent deployment
    - `architecture/` – System design notes (API server, bundlers, data model)
    - `integration/` – API and automation guidance
    - `v1/` – Legacy docs (manual install guides, playbooks, optimizations, troubleshooting)
- **deployments/** – Kubernetes manifests
  - `deployments/eidos-agent/` – Job manifests for agent deployment
- **examples/** – Example snapshots, recipes, and generated bundles
    - `examples/snapshots/` – Sample snapshot inputs
    - `examples/recipes/` – Sample recipe outputs
    - `examples/bundles/` – Sample generated bundles
- **api/** – OpenAPI/YAML specifications for API contracts
- **install/** – Installation assets/scripts for the CLI
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

## GitHub Actions & CI/CD

### Architecture Overview
The project uses a **three-layer composite actions architecture** for maximum reusability and separation of concerns:

1. **Primitives** – Single-purpose, reusable building blocks
   - `ghcr-login` – GHCR authentication
   - `setup-build-tools` – Modular tool installer
   - `security-scan` – Trivy vulnerability scanning

2. **Composed Actions** – Combine primitives for specific workflows
   - `go-ci` – Complete Go CI pipeline (setup → test → lint)
   - `attest-image-from-tag` – Resolve digest + generate attestations
   - `go-build-release` – Full build/release pipeline
   - `cloud-run-deploy` – GCP deployment with Workload Identity

3. **Workflows** – Orchestrate actions for complete CI/CD pipelines
   - `on-push.yaml` – CI validation for PRs and main branch
   - `on-tag.yaml` – Release, attestation, and deployment

### Composite Actions Best Practices

**Creating New Actions:**
```yaml
name: 'Action Name'
description: 'Clear, concise description of what it does'

inputs:
  required_input:
    description: 'What this input controls'
    required: true
  optional_input:
    description: 'Optional parameter with sensible default'
    required: false
    default: 'default-value'

outputs:
  result:
    description: 'What this output contains'
    value: ${{ steps.step-id.outputs.value }}

runs:
  using: 'composite'
  steps:
    # Use primitives where possible
    - uses: ./.github/actions/primitive-action
      with:
        param: ${{ inputs.required_input }}
```

**Key Principles:**
- **Single Responsibility** – Each action does one thing well
- **Input Validation** – Validate all inputs, fail fast with clear errors
- **Idempotent** – Safe to run multiple times
- **Composable** – Can be used standalone or as part of larger workflows
- **No Checkout** – Composite actions assume repo is already checked out (workflows handle checkout)
- **Explicit Outputs** – Document all outputs clearly

### Workflow Patterns

**Push/PR Validation (on-push.yaml):**
```yaml
jobs:
  validate:
    steps:
      - uses: actions/checkout@v4
      
      - uses: ./.github/actions/go-ci
        with:
          go_version: '1.25'
          golangci_lint_version: 'v2.6'
          upload_codecov: 'true'
      
      - uses: ./.github/actions/security-scan
        with:
          severity: 'MEDIUM,HIGH,CRITICAL'
```

**Release Pipeline (on-tag.yaml):**
```yaml
jobs:
  release:
    steps:
      - uses: actions/checkout@v4
      
      # Validate before release
      - uses: ./.github/actions/go-ci
        with:
          go_version: '1.25'
          golangci_lint_version: 'v2.6'
      
      # Build and publish
      - id: release
        uses: ./.github/actions/go-build-release
      
      # Generate attestations
      - uses: ./.github/actions/attest-image-from-tag
        with:
          image_name: 'ghcr.io/nvidia/cloud-native-stack/eidos'
          image_tag: ${{ github.ref_name }}
      
      # Deploy (only if build succeeded)
      - if: steps.release.outputs.release_outcome == 'success'
        uses: ./.github/actions/cloud-run-deploy
        with:
          service_name: 'eidos-api-server'
```

### Authentication Patterns

**Container Registries:**
- Use `ghcr-login` action (centralizes GHCR auth logic)
- Pass `github.token` for authentication (not PATs)
- Actions inherit authentication from workflows

**Cloud Providers:**
- Use Workload Identity Federation (keyless)
- No stored credentials
- `cloud-run-deploy` handles GCP WIF authentication

**Attestation/SBOM:**
- Cosign uses keyless signing (Sigstore)
- GitHub's attestation API for provenance
- SBOM generated with Syft (SPDX format)

### Security Best Practices

**Workflow Permissions:**
```yaml
permissions:
  contents: read          # Minimal permissions by default
  id-token: write        # For attestations and OIDC
  security-events: write # For SARIF uploads
```

**Action Pinning:**
```yaml
# Pin external actions by SHA (not tag)
- uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # v4.2.2

# Local actions use relative path (always current version)
- uses: ./.github/actions/go-ci
```

**Secrets Management:**
- Use `github.token` (automatically provided)
- Use OIDC for cloud authentication (no stored credentials)
- Never log or expose secrets in outputs

### Metrics & Observability

**CI/CD Metrics:**
- Workflow duration (tracked automatically by GitHub)
- Test coverage (uploaded to Codecov)
- Vulnerability scan results (SARIF to Security tab)
- Rate limit usage (monitor in workflow logs)

**Cost Optimization:**
- Use dependency caching (`actions/setup-go` built-in cache)
- Parallel job execution where possible
- Fail fast on validation errors
- Use `if` conditions to skip unnecessary steps

### Common Pitfalls

❌ **Don't do this:**
```yaml
# Redundant authentication (already logged in via ghcr-login)
- uses: docker/login-action@v3
  with:
    registry: ghcr.io

# Inline tool installation (use setup-build-tools)
- run: |
    curl -sSfL https://raw.githubusercontent.com/ko-build/ko/main/install.sh | sh
```

✅ **Do this:**
```yaml
# Use centralized authentication
- uses: ./.github/actions/ghcr-login

# Use modular tool installer
- uses: ./.github/actions/setup-build-tools
  with:
    install_ko: 'true'
    install_crane: 'true'
```

### Debugging Workflows

**Enable Debug Logging:**
1. Go to repository Settings → Secrets → Actions
2. Add `ACTIONS_STEP_DEBUG` = `true`
3. Re-run workflow to see detailed logs

**Local Testing:**
```bash
# Test composite actions locally with act
act -j validate --secret GITHUB_TOKEN="$GITHUB_TOKEN"

# Test individual make targets
make qualify  # Runs test + lint + scan (same as CI)
```

**Common Issues:**
- **"Can't find 'action.yml'"** – Add `actions/checkout@v4` before local actions
- **Rate limit exceeded** – Authenticate with `github.token` (higher limits)
- **SARIF upload fails** – Ensure `security-events: write` permission
- **Action output not available** – Check step ID matches and action has `outputs` defined

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

### Adding a New Bundler

Bundlers generate deployment artifacts (Helm values, manifests, scripts) from recipes. The framework uses **BaseBundler** to reduce boilerplate by ~75%.

**Quick Start:**

1. **Create bundler package** in `pkg/bundler/<bundler-name>/`:
```go
package mybundler

import (
    "context"
    "embed"
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler"
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

const bundlerType = bundler.BundleType("my-bundler")

func init() {
    // Self-register (panics on duplicates for fail-fast)
    bundler.MustRegister(bundlerType, NewBundler())
}

type Bundler struct {
    *bundler.BaseBundler  // Embed helper
}

func NewBundler() *Bundler {
    return &Bundler{
        BaseBundler: bundler.NewBaseBundler(bundlerType, templatesFS),
    }
}

func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, 
    outputDir string) (*bundler.BundleResult, error) {
    
    // 1. Create directory structure
    dirs := []string{"manifests", "scripts"}
    if err := b.CreateBundleDir(outputDir, dirs...); err != nil {
        return nil, err
    }
    
    // 2. Build config map from recipe
    configMap := b.buildConfigMap(r)
    
    // 3. Generate typed data structures
    helmValues := GenerateHelmValues(r, configMap)
    
    // 4. Generate files from templates
    filePath := filepath.Join(outputDir, "values.yaml")
    if err := b.GenerateFileFromTemplate(ctx, GetTemplate, 
        "values.yaml", filePath, helmValues, 0644); err != nil {
        return nil, err
    }
    
    var generatedFiles []string
    // ... collect file paths
    
    // 5. Generate checksums and return result
    return b.GenerateResult(outputDir, generatedFiles)
}
```

2. **Create templates** in `templates/` directory:
```yaml
# values.yaml.tmpl
operator:
  version: {{ .OperatorVersion.Value }}
  enabled: {{ .Enabled.Value }}
```

3. **Test with TestHarness** (reduces test code by 34%):
```go
func TestBundler_Make(t *testing.T) {
    harness := internal.NewTestHarness(t, NewBundler())
    
    tests := []struct {
        name    string
        recipe  *recipe.Recipe
        wantErr bool
        verify  func(t *testing.T, outputDir string)
    }{
        {
            name:    "valid recipe",
            recipe:  createTestRecipe(),
            wantErr: false,
            verify: func(t *testing.T, outputDir string) {
                harness.AssertFileContains(outputDir, "values.yaml",
                    "operator:", "version: v1.0.0")
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := harness.RunTest(tt.recipe, tt.wantErr)
            if !tt.wantErr && tt.verify != nil {
                tt.verify(t, result.OutputDir)
            }
        })
    }
}
```

**BaseBundler provides:**
- `CreateBundleDir(path, subdirs...)` – Directory structure
- `WriteFile(path, content)` – File writing with error handling
- `GenerateFileFromTemplate(ctx, getter, name, path, data, perm)` – Template rendering
- `GenerateResult(dir, files)` – BundleResult with checksums
- `Validate(ctx, recipe)` – Default validation (override if needed)

**Internal helpers** in `pkg/bundler/internal`:
- `BuildBaseConfigMap(recipe, additional)` – Extract config strings
- `ExtractK8sImageSubtype(recipe)` – Get K8s image measurements
- `ExtractGPUDeviceSubtype(recipe)` – Get GPU measurements
- Plus 12 more measurement extraction helpers

**Best Practices:**
- ✅ Embed `BaseBundler` instead of implementing from scratch
- ✅ Use `internal` package helpers for recipe data extraction
- ✅ Pass Go structs directly to templates (no map conversion)
- ✅ Self-register in `init()` with `MustRegister()` (fail-fast)
- ✅ Keep bundlers stateless (thread-safe by default)
- ✅ Use `TestHarness` for consistent test structure
- ✅ Document template variables and expected data types

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
- **GitHub Actions README**: [.github/actions/README.md](actions/README.md)
- **Playbooks**: [../docs/v1/playbooks/readme.md](../docs/v1/playbooks/readme.md)
- **API Specification**: [../api/eidos/v1/api-server-v1.yaml](../api/eidos/v1/api-server-v1.yaml)
- **GoDoc**: Run `make docs` and visit http://localhost:6060

## Version Information

**Current Stack:**
- Go: 1.25
- Kubernetes: 1.33+
- golangci-lint: v2.6
- Trivy: v0.33.1
- Ko: latest
- Syft: latest
- Crane: v0.20.6
- GoReleaser: v6.4.0
- Cosign: v4.0.0

**Container Registries:**
- GHCR: `ghcr.io/nvidia/cloud-native-stack`
- Google Artifact Registry: `us-docker.pkg.dev/PROJECT/REPO`

**Deployment Targets:**
- Google Cloud Run: `eidos-api-server` service
- Kubernetes: `eidos-agent` Job in `gpu-operator` namespace
