# Copilot Instructions for NVIDIA Cloud Native Stack

## Critical Rules (Always Apply)

**Code Quality (Non-Negotiable):**
- Tests must pass with race detector (`make test`)
- Never disable tests to make CI green (including "temporary" skips)
- Use structured errors from `pkg/errors` with error codes
- Context with timeouts for all I/O operations (prevent resource leaks)
- Stop after 3 failed attempts at same fix → reassess approach

**Development Process:**
- Plan non-trivial work in stages (use `IMPLEMENTATION_PLAN.md` for complex tasks)
- Follow red → green → refactor cycle
- Commit incrementally with "why" explanations
- Learn existing patterns before inventing new ones
- Verify assumptions with code/tests, never assume

**Go Code Requirements:**
- Handle context cancellation explicitly
- Define timeouts at API boundaries (collectors: 10s, handlers: 30s)
- Wrap errors with actionable context: `fmt.Errorf("operation: %w", err)`
- Use table-driven tests for multiple scenarios
- Run with `-race` flag enabled

**Decision Framework:**
Choose solutions based on: testability, readability, consistency, simplicity, reversibility

---

## Project Context

NVIDIA Cloud Native Stack (CNS/Eidos) provides validated GPU-accelerated Kubernetes configurations through a three-stage workflow:

1. **Snapshot** → Capture system state (OS, kernel, K8s, GPU)
2. **Recipe** → Generate optimized config from captured data or query parameters
3. **Bundle** → Create deployment artifacts (Helm values, manifests, scripts)

**Core Components:**
- **CLI (`eidos`)**: All three stages (snapshot/recipe/bundle)
- **API Server**: Recipe generation via REST API (https://cns.dgxc.io)
- **Agent**: Kubernetes Job for automated cluster snapshots → ConfigMaps
- **Bundlers**: Plugin-based artifact generators (GPU Operator, Network Operator, Cert-Manager, NVSentinel, Skyhook)

**Tech Stack:** Go 1.25, Kubernetes 1.33+, golangci-lint v2.6, Container images via Ko

**Quick Start:**
```bash
make qualify  # Run tests + lint + scan (full check)
make build    # Build binaries
make server   # Start API server locally
```

---

## Common Tasks (Start Here)

### I Need To: Add GPU Support for New Hardware

1. **Add collector** in `pkg/collector/gpu/`:
   - Implement `Collector` interface
   - Add factory method in `factory.go`
   - Write table-driven tests with mocks

2. **Update recipe data** in `pkg/recipe/data/data-v1.yaml`:
   - Add base measurements for new GPU type
   - Create overlays with Query matching

3. **Test workflow**:
   ```bash
   eidos snapshot --output snapshot.yaml
   eidos recipe --snapshot snapshot.yaml --intent training
   ```

→ See Extended Reference: Adding a New Collector for full example

### I Need To: Generate Bundles for New Operator

1. **Create bundler** in `pkg/bundler/<name>/`:
   - Embed `BaseBundler` (reduces boilerplate by 75%)
   - Implement `Make()` method
   - Use `internal` helpers for recipe extraction
   - Self-register with `MustRegister()` in `init()`

2. **Add templates** in `templates/` directory:
   - Use `go:embed` for portability
   - Pass Go structs directly (no map conversion)

3. **Write tests** with `TestHarness`:
   ```go
   harness := internal.NewTestHarness(t, NewBundler())
   result := harness.RunTest(recipe, wantErr)
   harness.AssertFileContains(dir, "values.yaml", "version:")
   ```

→ See Extended Reference: Adding a New Bundler for full guide

### I Need To: Add New API Endpoint

1. Create handler in `pkg/api/`:
   ```go
   func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
       ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
       defer cancel()
       // Handler logic
   }
   ```

2. Register route in `pkg/api/server.go`
3. Add middleware (metrics → version → requestID → panic → rateLimit → logging)
4. Update API spec in `api/eidos/v1/api-server-v1.yaml`
5. Write integration tests

→ See Extended Reference: Adding a New API Endpoint for detailed steps

### I Need To: Fix Failing Tests

1. **Check error messages** → Use proper assertions:
   ```go
   if err != nil {
       t.Fatalf("unexpected error: %v", err)  // Stop test immediately
   }
   ```

2. **Race conditions** → Run `go test -race ./...`
3. **Linting issues** → `make lint` then fix reported problems
4. **Context** → Ensure collectors/handlers respect `ctx.Done()`

→ See Troubleshooting & Support for common issues and debugging tips

---

## Development Patterns

### Go Architecture Patterns

**1. Functional Options (Configuration)**
```go
builder := recipe.NewBuilder(
    recipe.WithVersion(version),
)
server := server.New(
    server.WithName("eidos-api-server"),
    server.WithVersion(version),
)
```

**2. Factory Pattern (Collectors)**
```go
factory := collector.NewDefaultFactory(
    collector.WithSystemDServices([]string{"containerd.service"}),
)
gpuCollector := factory.CreateGPUCollector()
```

**3. Builder Pattern (Measurements)**
```go
measurement.NewMeasurement(measurement.TypeK8s).
    WithSubtype(subtype).
    Build()
```

**4. Singleton Pattern (K8s Client)**
```go
import "github.com/NVIDIA/cloud-native-stack/pkg/k8s/client"

clientset, config, err := client.GetKubeClient()  // Uses sync.Once
```

### Error Handling (Required Pattern)

Always use structured errors from `pkg/errors`:

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

**Error Codes:** `ErrCodeNotFound`, `ErrCodeUnauthorized`, `ErrCodeTimeout`, `ErrCodeInternal`, `ErrCodeInvalidRequest`, `ErrCodeUnavailable`

### Context & Timeouts (Required Pattern)

Always use context with timeouts:

```go
// In collectors
func (c *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    // Collection logic
}

// In HTTP handlers
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()
    // Handler logic
}
```

### Concurrency (errgroup Pattern)

Use errgroup for parallel operations:

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

### Structured Logging (slog)

Use structured logging with appropriate levels:

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
- `Debug` – Detailed diagnostics (enabled with `--debug`)
- `Info` – General informational messages (default)
- `Warn` – Warning conditions
- `Error` – Error conditions requiring attention

---

## Testing & Quality

### Essential Commands

```bash
# Full qualification (run before PR)
make qualify       # test + lint + scan

# Individual checks
make test          # Unit tests with race detector
make lint          # golangci-lint + yamllint
make scan          # Trivy vulnerability scan

# Build and run
make build         # Build for current platform
make server        # Start API server (debug mode)
make tidy          # Format code + update deps
```

### Testing Requirements

- **Coverage**: Aim for >70% meaningful coverage (current: ~60%)
- **Race Detector**: Always enabled (`make test` runs with `-race`)
- **Table-Driven Tests**: Required for multiple test cases
- **Mocks**: Use fake clients for external dependencies (K8s client-go fakes)
- **Error Cases**: Test error conditions and edge cases explicitly

Example test structure:
```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid", "test", "test", false},
        {"empty", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

---

## Troubleshooting & Support

### Common Issues

- **K8s Connection**: Check kubeconfig at `~/.kube/config` or `KUBECONFIG` env
- **GPU Detection**: Requires nvidia-smi in PATH
- **Linter Errors**: Use `errors.Is()` for error comparison, add `return` after `t.Fatal()`
- **Race Conditions**: Run tests with `-race` flag to detect
- **Build Failures**: Run `make tidy` to fix Go module issues

### Debugging Commands

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

---

## Go & Distributed Systems Principles

### Architectural Philosophy

**Role:** Act as a Principal Distributed Systems Architect. Default to correctness, resiliency, and operational simplicity. All Go code should be production-grade.

**Core Tenets:**
- **Partial failure is the steady state** → Design for timeouts, retries, circuit breakers, backpressure
- **Boring first** → Use proven, simple technologies; introduce complexity only for concrete limitations
- **Observability is mandatory** → Structured logging, metrics, tracing are part of the contract
- **Precision over generalities** → "enforce mTLS using SPIFFE identities" not "ensure security"

**Code Quality Requirements (Go):**
- Handle context cancellation explicitly
- Define timeouts at API boundaries
- Wrap errors with actionable context
- Use table-driven tests

**Evidence-Based:** Ground recommendations in verifiable sources (Go spec, k8s.io docs, CNCF projects, industry papers)

---

## Documentation Development

When writing documentation, act as a senior open-source documentation editor with CNCF/Linux Foundation experience.

**Goals:**
- Improve technical clarity without changing intent
- Ensure suitability for diverse, global open-source audience
- Align with CNCF / Linux Foundation conventions

**Standards:**

1. **Accuracy & Scope**
   - Don't invent features, guarantees, timelines, or roadmap commitments
   - Clearly distinguish current behavior from future intent
   - Remove speculative or marketing language

2. **Tone & Style**
   - Use neutral, factual, engineering-oriented language
   - Avoid hype ("best", "powerful", "game-changing")
   - Prefer short, declarative sentences

3. **Structure & Readability**
   - Organize with clear sections and logical flow
   - Use headings that answer user questions
   - Convert dense paragraphs into lists or tables
   - Ensure examples are minimal, relevant, clearly labeled

4. **Audience Awareness**
   - Assume engineers but not necessarily project experts
   - Define acronyms on first use
   - Clearly state prerequisites and assumptions

5. **Operational Clarity**
   - Document configuration boundaries
   - Document failure modes or limitations
   - Document upgrade/compatibility considerations
   - Prefer "what happens" over "what should happen"

---

## Quick Reference

### Commands

```bash
# Development
make tidy         # Format code + update deps
make build        # Build binaries
make server       # Start API server locally
make test         # Run tests with coverage
make lint         # Lint Go and YAML
make scan         # Security scanning
make qualify      # Full check (test + lint + scan)

# Workflow
eidos snapshot --output snapshot.yaml
eidos recipe --snapshot snapshot.yaml --intent training
eidos bundle --recipe recipe.yaml --output ./bundles

# Override bundle values at generation time
eidos bundle -f recipe.yaml -b gpu-operator \
  --set gpuoperator:gds.enabled=true \
  --set gpuoperator:driver.version=570.86.16 \
  -o ./bundles
```

### Integration Points

- **Kubernetes**: Singleton client via `pkg/k8s/client.GetKubeClient()`
- **NVIDIA Operators**: GPU Operator, Network Operator, NIM Operator, Nsight Operator
- **Container Images**: ghcr.io/nvidia/eidos, ghcr.io/nvidia/eidos-api-server
- **Observability**: Prometheus metrics at `/metrics`, structured JSON logs to stderr

### Key Links

- **[Contributing Guide](../CONTRIBUTING.md)** – Development setup, PR process
- **[Architecture Overview](../docs/architecture/README.md)** – System design
- **[Bundler Development](../docs/architecture/bundler-development.md)** – Create new bundlers
- **[API Reference](../docs/integration/api-reference.md)** – REST API endpoints
- **[GitHub Actions README](actions/README.md)** – CI/CD architecture
- **[API Specification](../api/eidos/v1/api-server-v1.yaml)** – OpenAPI spec

### Version Information

Check current versions dynamically:
```bash
make info         # Show all tool versions
cat go.mod        # Go module versions
```

---

## Extended Reference

### Adding a New Collector

If adding a new system collector (like the OS release collector added in v0.7.0):

**1. Create the collector in `pkg/collector/os/`:**
```go
// pkg/collector/os/release.go
package os

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"
)

// collectRelease reads and parses /etc/os-release
func (c *Collector) collectRelease(ctx context.Context) (*measurement.Subtype, error) {
    data := make(map[string]measurement.Reading)
    
    file, err := os.Open("/etc/os-release")
    if err != nil {
        return nil, fmt.Errorf("failed to open /etc/os-release: %w", err)
    }
    defer file.Close()
    
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        
        key := parts[0]
        value := strings.Trim(parts[1], `"`)
        data[key] = measurement.Reading{Value: value}
    }
    
    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("error reading /etc/os-release: %w", err)
    }
    
    return &measurement.Subtype{
        Name: "release",
        Data: data,
    }, nil
}
```

**2. Update the main collector:**
```go
// pkg/collector/os/os.go
func (c *Collector) Collect(ctx context.Context) ([]*measurement.Measurement, error) {
    // Collect all OS subtypes in parallel
    grubSubtype, _ := c.collectGrub(ctx)
    sysctlSubtype, _ := c.collectSysctl(ctx)
    kmodSubtype, _ := c.collectKmod(ctx)
    releaseSubtype, _ := c.collectRelease(ctx) // New subtype
    
    return []*measurement.Measurement{{
        Type: measurement.TypeOS,
        Subtypes: []*measurement.Subtype{
            grubSubtype,
            sysctlSubtype,
            kmodSubtype,
            releaseSubtype, // Add to list
        },
    }}, nil
}
```

**3. Add tests:**
```go
// pkg/collector/os/release_test.go
func TestCollectRelease(t *testing.T) {
    c := NewCollector()
    ctx := context.Background()
    
    subtype, err := c.collectRelease(ctx)
    if err != nil {
        t.Fatalf("collectRelease() error = %v", err)
    }
    
    // Verify expected fields exist
    expectedFields := []string{"ID", "VERSION_ID", "PRETTY_NAME"}
    for _, field := range expectedFields {
        if _, exists := subtype.Data[field]; !exists {
            t.Errorf("expected field %q not found", field)
        }
    }
    
    // Verify subtype name
    if subtype.Name != "release" {
        t.Errorf("expected subtype name 'release', got %q", subtype.Name)
    }
}
```

**4. Update integration tests:**
```go
// pkg/collector/os/os_test.go
func TestOSCollector(t *testing.T) {
    measurements, err := c.Collect(ctx)
    if err != nil {
        t.Fatalf("Collect() error = %v", err)
    }
    
    // Should return 4 subtypes: grub, sysctl, kmod, release
    if len(measurements[0].Subtypes) != 4 {
        t.Errorf("expected 4 subtypes, got %d", len(measurements[0].Subtypes))
    }
}
```

### Adding a New Bundler

The bundler framework uses **BaseBundler** - a helper that reduces boilerplate by ~75% (from ~400 lines to ~100 lines). Instead of implementing the full `Bundler` interface from scratch, embed `BaseBundler` and override only what you need.

**1. Create bundler package in `pkg/bundler/<bundler-name>/`:**
```go
package networkoperator

import (
    "context"
    "embed"
    
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler"
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
    "github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

const bundlerType = bundler.BundleType("network-operator")

func init() {
    // Self-register using MustRegister (panics on duplicates)
    bundler.MustRegister(bundlerType, NewBundler())
}

// Bundler generates Network Operator deployment bundles.
type Bundler struct {
    *bundler.BaseBundler  // Embed helper for common functionality
}

// NewBundler creates a new Network Operator bundler instance.
func NewBundler() *Bundler {
    return &Bundler{
        BaseBundler: bundler.NewBaseBundler(bundlerType, templatesFS),
    }
}

// Make generates the bundle (override BaseBundler.Make).
func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, 
    outputDir string) (*bundler.BundleResult, error) {
    
    // 1. Create bundle directory structure
    dirs := []string{"manifests", "scripts"}
    if err := b.CreateBundleDir(outputDir, dirs...); err != nil {
        return nil, err
    }
    
    // 2. Build configuration map from recipe
    configMap := b.buildConfigMap(r)
    
    // 3. Generate typed data structures from recipe
    helmValues := GenerateHelmValues(r, configMap)
    scriptData := GenerateScriptData(r, configMap)
    
    // 4. Generate files from templates with typed structs
    filePath := filepath.Join(outputDir, "values.yaml")
    if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml",
        filePath, helmValues, 0644); err != nil {
        return nil, err
    }
    
    filePath = filepath.Join(outputDir, "scripts/install.sh")
    if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "install.sh",
        filePath, scriptData, 0755); err != nil {
        return nil, err
    }
    
    var generatedFiles []string
    // ... collect file paths
    
    // 5. Generate checksums and return result
    return b.GenerateResult(outputDir, generatedFiles)
}
```

**2. Create templates directory:**
```
pkg/bundler/networkoperator/templates/
├── values.yaml.tmpl               # Helm chart values
├── nicclusterpolicy.yaml.tmpl     # NICClusterPolicy CR
├── install.sh.tmpl                # Installation script
├── uninstall.sh.tmpl              # Cleanup script
└── README.md.tmpl                 # Documentation
```

**Example template (`values.yaml.tmpl`):**
```yaml
# Network Operator Helm Values
# Generated by CNS Eidos
# Timestamp: {{ .Timestamp }}
# Version: {{ .Version }}

networkOperator:
  # Direct struct field access - no map conversion
  version: {{ .NetworkOperatorVersion.Value }}
  
driver:
  image: nvcr.io/nvidia/mellanox/mofed
  version: {{ .OFEDVersion.Value }}
  
config:
  rdma:
    enabled: {{ .EnableRDMA.Value }}
  sriov:
    enabled: {{ .EnableSRIOV.Value }}
```

**3. Write tests with TestHarness:**
```go
package networkoperator

import (
    "testing"
    
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
    "github.com/NVIDIA/cloud-native-stack/pkg/measurement"
    "github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

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
                    "networkOperator:", "version:")
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

**Key Components:**
- **BaseBundler** provides: `CreateBundleDir`, `WriteFile`, `GenerateFileFromTemplate`, `GenerateResult`, `Validate`
- **Internal helpers**: `BuildBaseConfigMap`, `ExtractK8sImageSubtype`, `ExtractGPUDeviceSubtype`, etc.
- **Data generation functions**: `GenerateHelmValues`, `GenerateManifestData`, `GenerateScriptData`
- **TestHarness**: `NewTestHarness`, `RunTest`, `AssertFileContains`

**Best Practices:**
- Embed `BaseBundler` instead of implementing from scratch
- Use `internal` package helpers for recipe extraction
- Pass Go structs directly to templates (no map conversion)
- Self-register with `MustRegister()` for fail-fast behavior
- Keep bundlers stateless for thread-safe operation
- Use `TestHarness` for consistent test structure

### Adding a New API Endpoint

**1. Create handler in `pkg/api/`:**
```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()
    
    // Parse query parameters
    query, err := recipe.NewQuery(
        recipe.WithOS(r.URL.Query().Get("os")),
        recipe.WithGPU(r.URL.Query().Get("gpu")),
    )
    if err != nil {
        serializer.WriteError(w, r, err, http.StatusBadRequest)
        return
    }
    
    // Generate recipe
    recipe, err := h.builder.Build(ctx, query)
    if err != nil {
        serializer.WriteError(w, r, err, http.StatusInternalServerError)
        return
    }
    
    // Serialize response
    serializer.WriteJSON(w, recipe, http.StatusOK)
}
```

**2. Register route in `pkg/api/server.go`:**
```go
mux.Handle("/v1/recipe", handler)
```

**3. Add middleware (order matters):**
```go
// Order: metrics → version → requestID → panic → rateLimit → logging → handler
handler = metricsMiddleware(handler)
handler = versionMiddleware(handler, version)
handler = requestIDMiddleware(handler)
handler = panicMiddleware(handler)
handler = rateLimitMiddleware(handler, limiter)
handler = loggingMiddleware(handler)
```

**4. Update API spec in `api/eidos/v1/api-server-v1.yaml`:**
```yaml
paths:
  /v1/recipe:
    get:
      summary: Get optimized system configuration recipe
      parameters:
        - name: os
          in: query
          required: false
          schema:
            type: string
            enum: [ubuntu, cos, any]
```

**5. Write integration tests:**
```go
func TestRecipeHandler(t *testing.T) {
    server := httptest.NewServer(handler)
    defer server.Close()
    
    resp, err := http.Get(server.URL + "/v1/recipe?os=ubuntu&gpu=h100")
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected 200, got %d", resp.StatusCode)
    }
}
```

### GitHub Actions & CI/CD Architecture

Cloud Native Stack uses a **three-layer composite actions architecture** for reusability:

**Layer 1: Primitives** (Single-Purpose Building Blocks)
- `ghcr-login` – GHCR authentication
- `setup-build-tools` – Modular tool installer (ko, syft, crane, goreleaser)
- `security-scan` – Trivy vulnerability scanning

**Layer 2: Composed Actions** (Combine Primitives)
- `go-ci` – Complete Go CI pipeline (setup → test → lint)
- `go-build-release` – Full build/release pipeline
- `attest-image-from-tag` – Resolve digest + generate attestations
- `cloud-run-deploy` – GCP deployment with Workload Identity

**Layer 3: Workflows** (Orchestrate Actions)
- `on-push.yaml` – CI validation for PRs and main branch
- `on-tag.yaml` – Release, attestation, and deployment

**Key Workflows:**

**on-push.yaml** (CI validation):
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
```

**on-tag.yaml** (Release pipeline):
```yaml
jobs:
  release:
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/go-ci
      - id: release
        uses: ./.github/actions/go-build-release
      - uses: ./.github/actions/attest-image-from-tag
        with:
          image_name: 'ghcr.io/nvidia/eidos'
          image_tag: ${{ github.ref_name }}
      - if: steps.release.outputs.release_outcome == 'success'
        uses: ./.github/actions/cloud-run-deploy
```

**Supply Chain Security:**
- **SLSA Build Level 3**: GitHub OIDC attestations
- **SBOMs**: SPDX format via Syft (containers) and GoReleaser (binaries)
- **Signing**: Cosign keyless signing (Fulcio + Rekor)
- **Verification**: `gh attestation verify oci://ghcr.io/nvidia/eidos:${TAG}`

For detailed GitHub Actions architecture, see [.github/actions/README.md](actions/README.md)

### Workflow Patterns

**Complete End-to-End: Snapshot → Recipe → Bundle**
```bash
# 1. Capture system configuration
eidos snapshot --output snapshot.yaml

# 2. Generate optimized recipe for training workloads
eidos recipe \
  --snapshot snapshot.yaml \
  --intent training \
  --format yaml \
  --output recipe.yaml

# 3. Create deployment bundle
eidos bundle \
  --recipe recipe.yaml \
  --bundlers gpu-operator \
  --output ./bundles

# 4. Deploy to cluster
cd bundles/gpu-operator
sha256sum -c checksums.txt  # Verify integrity
chmod +x scripts/install.sh
./scripts/install.sh
```

**ConfigMap-based Workflow (for Kubernetes Jobs):**
```bash
# 1. Capture snapshot directly to ConfigMap
eidos snapshot -o cm://gpu-operator/eidos-snapshot

# 2. Generate recipe from ConfigMap snapshot
eidos recipe -f cm://gpu-operator/eidos-snapshot \
  --intent training \
  -o cm://gpu-operator/eidos-recipe

# 3. Create bundle from ConfigMap recipe
eidos bundle -f cm://gpu-operator/eidos-recipe \
  -b gpu-operator \
  -o ./bundles

# 4. Verify ConfigMap data
kubectl get configmap eidos-snapshot -n gpu-operator -o yaml
kubectl get configmap eidos-recipe -n gpu-operator -o yaml
```

**E2E Testing with Agent:**
```bash
# Run full E2E test (snapshot → recipe → bundle)
./tools/e2e -s examples/snapshots/h100.yaml \
           -r examples/recipes/h100-eks-ubuntu-training.yaml \
           -b examples/bundles/h100-eks-ubuntu-training

# The script:
# 1. Deploys agent Job to cluster
# 2. Waits for snapshot to be written to ConfigMap
# 3. Optionally saves snapshot to file
# 4. Optionally generates recipe using cm://gpu-operator/eidos-snapshot
# 5. Optionally generates bundle from recipe
# 6. Validates each step completes successfully
```

**Agent Deployment Pattern:**
```bash
# Deploy agent for automated snapshots
kubectl apply -f deployments/eidos-agent/1-deps.yaml
kubectl apply -f deployments/eidos-agent/2-job.yaml

# Check logs
kubectl logs -n gpu-operator job/eidos

# Get snapshot from ConfigMap
kubectl get configmap eidos-snapshot -n gpu-operator \
  -o jsonpath='{.data.snapshot\.yaml}' > snapshot.yaml
```

