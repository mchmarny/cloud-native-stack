# Cloud Native Stack Architecture

This directory contains architecture documentation for the Cloud Native Stack (CNS) tooling.

## Components

- **[CLI Architecture](cli.md)** - Architecture of the `eidos` command-line tool for capturing system snapshots, generating configuration recipes, and creating deployment bundles
- **[API Server Architecture](api-server.md)** - Architecture of the HTTP API server for serving configuration recommendations
- **Bundler Framework** - Extensible system for generating deployment bundles (Helm charts, manifests, scripts) from recipes with parallel execution by default

## Overview

Cloud Native Stack provides two complementary interfaces for system configuration management:

### CLI Tool (`eidos`)
A command-line interface for direct interaction with system configuration:
- **Snapshot Command**: Captures comprehensive system configuration
- **Recipe Command**: Generates optimized configuration recipes from environment parameters or snapshots
  - **Query Mode**: Direct recipe generation from system parameters (OS, GPU, K8s, etc.)
  - **Snapshot Mode**: Analyzes captured snapshots and generates tailored recipes based on workload intent
- **Bundle Command**: Generates deployment-ready bundles (Helm values, manifests, scripts) from recipes
  - Parallel execution of multiple bundlers by default
  - Error collection or fail-fast modes

### API Server (`eidos-api-server`)
An HTTP REST API for programmatic access to configuration recipes:
- **Recipe Endpoint**: Serves configuration recipes via HTTP
- **Health/Metrics**: Kubernetes-ready observability endpoints

## Key Design Principles

### 1. Separation of Concerns
**Pattern**: Shared library with multiple entry points (CLI, API server)  
**Rationale**: Maximizes code reuse while maintaining deployment flexibility  
**Reference**: [Go Project Layout](https://go.dev/doc/modules/layout)

### 2. Concurrent Collection with Bounded Parallelism
**Pattern**: `errgroup.WithContext` for fail-fast concurrent operations  
**Rationale**: Parallel collection reduces latency; context propagation enables cancellation  
**Trade-offs**: Memory overhead vs latency gain; appropriate for I/O-bound operations  
**Reference**: [golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)

**Implementation**:
```go
g, ctx := errgroup.WithContext(parentCtx)
g.Go(func() error { return collectK8s(ctx) })
g.Go(func() error { return collectGPU(ctx) })
if err := g.Wait(); err != nil {
    // First error cancels all goroutines via context
    return fmt.Errorf("collection failed: %w", err)
}
```

### 3. Pluggable Collectors via Abstract Factory
**Pattern**: Factory interface with concrete implementations  
**Rationale**: Testability (mock collectors), extensibility (add new sources)  
**Trade-off**: Additional abstraction vs testing simplicity  
**Reference**: [Go Interfaces](https://go.dev/doc/effective_go#interfaces)

### 4. Format Flexibility through Strategy Pattern
**Pattern**: Serializer interface with format-specific implementations  
**Rationale**: Open/closed principle - add formats without modifying callers  
**Implementation**: JSON, YAML, Table writers behind common `Serialize` interface

### 5. Production-Ready HTTP Server
**Patterns Implemented**:
- **Rate Limiting**: Token bucket (`golang.org/x/time/rate`)  
- **Graceful Shutdown**: Signal handling with deadline-based cleanup  
- **Observability**: Prometheus metrics, structured logging, request tracing  
- **Resilience**: Panic recovery, timeout enforcement, circuit breaker patterns  

**References**:  
- [Graceful Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)  
- [Rate Limiting](https://pkg.go.dev/golang.org/x/time/rate)  
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)

### 6. Semantic Versioning with Precision Control and Vendor Extras
**Pattern**: Version struct with Major.Minor.Patch components and optional Extras field  
**Rationale**: Flexible matching (1.2 matches 1.2.x); reject negative components; preserve vendor-specific suffixes  
**Vendor Support**: Handles kernel versions like `6.8.0-1028-aws` and Kubernetes versions like `v1.33.5-eks-3025e55`  
**Trade-off**: Complexity vs matching flexibility and vendor compatibility  
**Implementation**: Extras field stores vendor suffixes (e.g., `-1028-aws`, `-eks-3025e55`) without affecting version comparison  
**Reference**: [Semantic Versioning 2.0.0](https://semver.org/)

### 7. Immutable Data Structures
**Pattern**: Read-only recipe store with deep cloning for modifications  
**Rationale**: Thread-safety without locks; functional programming style  
**Implementation**: `sync.Once` for initialization, cloning for per-request mutations

### 8. Context-Aware Request Handling
**Pattern**: Context propagation for cancellation and timeouts  
**Rationale**: Prevents resource leaks; enables graceful degradation  
**Reference**: [Go Context Package](https://pkg.go.dev/context)

## Deployment Topologies

### Topology 1: Standalone CLI
**Use Case**: Local development, CI/CD pipelines, troubleshooting  
**Architecture**: Single binary, no network dependencies  
**Scaling**: Run on each node/machine independently

```mermaid
flowchart TD
    A["Developer"] --> B["eidos CLI"]
    B --> C["Local Node<br/>(K8s/GPU)"]
```

### Topology 2: Centralized API with Load Balancer
**Use Case**: Production environments, multi-tenant platforms  
**Architecture**: Multiple stateless replicas behind L7 load balancer  
**Scaling**: Horizontal auto-scaling based on request rate/latency

```mermaid
flowchart TD
    C1["Client 1"] --> LB
    C2["Client 2"] --> LB
    CN["Client N"] --> LB
    
    LB["Load Balancer<br/>(L7/HTTPS)"] --> API1
    LB --> API2
    LB --> API3
    
    API1["API v1 Pod"] --> PROM
    API2["API v2 Pod"] --> PROM
    API3["API v3 Pod"] --> PROM
    
    PROM["Prometheus<br/>(Metrics)"]
```

### Topology 3: Kubernetes Job Agent
**Use Case**: Automated cluster auditing, scheduled configuration checks  
**Architecture**: Job running on GPU nodes with host access  
**Scaling**: One Job per node or node-group

```mermaid
flowchart TD
    subgraph K8S["Kubernetes Cluster"]
        subgraph NODE1["GPU Node 1"]
            JOB1["Eidos Job"] --> OUT1["stdout"]
        end
        
        subgraph NODE2["GPU Node 2"]
            JOB2["Eidos Job"] --> OUT2["stdout"]
        end
        
        OUT1 --> STORE
        OUT2 --> STORE
        
        STORE["ConfigMap / Secret<br/>(Snapshots stored here)"]
    end
```

### Topology 4: Service Mesh Integration
**Use Case**: Zero-trust environments, mTLS everywhere  
**Architecture**: API server with sidecar proxy (Istio, Linkerd)  
**Scaling**: Service mesh handles load balancing, circuit breaking, retries

```mermaid
flowchart LR
    subgraph MESH["Kubernetes Cluster (Service Mesh)"]
        subgraph POD["eidos-api-server Pod"]
            PROXY["Envoy Proxy<br/>(mTLS/L7)"] <--> API["eidos<br/>API Server"]
        end
        
        OBS["Observability:<br/>Grafana, Jaeger"]
        SEC["Security:<br/>OPA, Cert-Manager"]
    end
    
    CLIENT["Clients"] --> PROXY
```

## Shared Core Packages

Both components leverage shared functionality:

```mermaid
flowchart TD
    PKG["pkg/"] --> COLL
    PKG --> MEAS
    PKG --> REC
    PKG --> VER
    PKG --> SER
    PKG --> LOG
    PKG --> SVR
    PKG --> BUN
    
    COLL["collector/<br/>System data collection<br/>(OS, K8s, GPU, SystemD)"]
    MEAS["measurement/<br/>Data model for<br/>collected metrics"]
    REC["recipe/<br/>Recipe building,<br/>query matching, and<br/>snapshot analysis"]
    VER["version/<br/>Semantic version<br/>parsing & comparison<br/>(with vendor extras)"]
    SER["serializer/<br/>Output formatting<br/>(JSON, YAML, table)"]
    LOG["logging/<br/>Structured logging"]
    SVR["server/<br/>HTTP server<br/>infrastructure (API only)"]
    BUN["bundler/<br/>Parallel bundle generation<br/>(Helm, manifests, scripts)<br/>Registry-based extensibility"]
```

## Data Flow

### CLI Snapshot Flow
```mermaid
flowchart LR
    A[User Command] --> B[CLI Parser]
    B --> C[Factory]
    C --> D[Collectors]
    D --> E[Measurements]
    E --> F[Serializer]
    F --> G[Output]
```

### CLI Recipe Flow
```mermaid
flowchart LR
    A[User Flags] --> B[Query Builder]
    B --> C[Recipe Builder]
    C --> D[Store Lookup]
    D --> E[Overlay Merge]
    E --> F[Serializer]
    F --> G[Output]
```

### API Recipe Flow
```mermaid
flowchart LR
    A[HTTP Request] --> B[Server Middleware]
    B --> C[Handler]
    C --> D[Recipe Builder]
    D --> E[Store Lookup]
    E --> F[JSON Response]
```

## Failure Modes and Recovery Strategies

### Collector Failures
**Failure**: Individual collector (K8s, GPU, SystemD) fails  
**Detection**: `errgroup` propagates first error  
**Recovery**: 
- **Fail-fast** (current): Entire snapshot fails if any collector fails  
- **Best-effort** (alternative): Continue with partial data, mark incomplete

**Trade-off Analysis**:  
- Fail-fast ensures data consistency but may be too strict  
- Best-effort improves availability but complicates downstream logic  
- **Decision**: Fail-fast for now; add best-effort mode behind feature flag

### Kubernetes API Server Unavailable
**Failure**: K8s API server unreachable or rate-limiting  
**Detection**: HTTP errors, context deadline exceeded  
**Recovery**:  
- Exponential backoff with jitter (2^n * 100ms + rand(0, 100ms))  
- Max retries: 3 with circuit breaker after 5 consecutive failures  
- Fallback: Cached data (if stale data acceptable)  

**Implementation Guidance**:  
```go
import "k8s.io/client-go/util/retry"

retry.OnError(retry.DefaultBackoff, func(err error) bool {
    return errors.Is(err, context.DeadlineExceeded)
}, func() error {
    return client.Get(ctx, key, obj)
})
```

**Reference**: [client-go Retry](https://pkg.go.dev/k8s.io/client-go/util/retry)

### GPU Driver/SMI Unavailable
**Failure**: nvidia-smi not found, driver not loaded  
**Detection**: Exec error, exit code != 0  
**Recovery**:  
- Graceful degradation: Return empty GPU measurements  
- Log warning with actionable message  
- Continue with other collectors

### Rate Limit Exceeded (API Server)
**Failure**: HTTP 429 Too Many Requests  
**Detection**: Response status code  
**Recovery**:  
- Read `Retry-After` header  
- Adaptive rate limiting: Reduce request rate dynamically  
- Circuit breaker: Open after N consecutive 429s  

**Implementation Pattern**:  
```go
if resp.StatusCode == http.StatusTooManyRequests {
    retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
    select {
    case <-time.After(retryAfter):
        return retry()
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### Memory Exhaustion
**Failure**: Large cluster with 1000s of pods causing OOM  
**Detection**: Runtime memory stats, container limits  
**Prevention**:  
- Streaming JSON parsing for large responses  
- Pagination for list operations  
- Memory limits in Kubernetes Deployment  

**Monitoring**:  
```promql
process_resident_memory_bytes / container_spec_memory_limit_bytes > 0.9
```

### API Server Graceful Shutdown
**Scenario**: SIGTERM received during active requests  
**Behavior**:  
1. Stop accepting new connections  
2. Wait for in-flight requests (30s timeout)  
3. Force-close remaining connections  
4. Exit with code 0

**Implementation** (already in place):  
```go
ctx, stop := signal.NotifyContext(context.Background(), 
    os.Interrupt, syscall.SIGTERM)
defer stop()

g.Go(func() error {
    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(
        context.Background(), 30*time.Second)
    defer cancel()
    return server.Shutdown(shutdownCtx)
})
```

**Reference**: [Graceful Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)

## Performance Considerations

### Latency Budget Breakdown

**Target**: p99 < 100ms for snapshot operations

| Component | Latency | Mitigation |
|-----------|---------|------------|
| K8s API List Pods | 10-50ms | Pagination, field selectors |
| SystemD DBus Calls | 5-20ms | Parallel collection |
| GPU nvidia-smi | 10-30ms | Cache results (5s TTL) |
| GRUB/Sysctl Read | 1-5ms | Buffered I/O |
| JSON Serialization | 1-10ms | Streaming encoder |
| **Total (parallel)** | **50-100ms** | **errgroup parallelism** |

### Memory Profile

| Component | Memory | Optimization |
|-----------|--------|-------------|
| Recipe Store | 5-10MB | Embed compressed YAML |
| K8s Client | 10-20MB | Shared informers |
| Snapshot Data | 1-5MB | Streaming serialization |
| Go Runtime | 5-10MB | GOGC tuning |
| **Total** | **21-45MB** | **Minimal footprint** |

### Concurrency Patterns

**CLI**: Bounded parallelism with errgroup (1 goroutine per collector)
```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(5) // Max 5 concurrent collectors
```

**API Server**: Per-request goroutines (bounded by rate limiter)  
- Rate limiter prevents goroutine explosion  
- Each request handled in dedicated goroutine (Go's http.Server pattern)  
- No explicit goroutine pooling needed

**Reference**: [Go HTTP Server Concurrency](https://go.dev/blog/context)

## Security Architecture

### Threat Model

| Threat | Impact | Mitigation | Priority |
|--------|--------|------------|----------|
| **DoS via Rate Exhaustion** | High | Token bucket rate limiter | P0 |
| **Memory Exhaustion** | High | Request timeouts, memory limits | P0 |
| **Command Injection** | Critical | No shell exec; use syscall | P0 |
| **Path Traversal** | Medium | Validate file paths | P1 |
| **Information Disclosure** | Medium | Sanitize error messages | P1 |
| **MITM** | High | TLS enforcement (external proxy) | P1 |
| **Replay Attacks** | Low | Idempotent operations | P2 |

### Defense in Depth

**Layer 1: Network**  
- Kubernetes NetworkPolicy: Restrict ingress to API server  
- Service Mesh mTLS: Encrypt inter-service communication

**Layer 2: Application**  
- Input validation: Strict enum/version parsing  
- Rate limiting: Prevent resource exhaustion  
- Timeout enforcement: Kill long-running requests

**Layer 3: Runtime**  
- Least privilege: Run as non-root user (UID 1000)  
- Read-only root filesystem  
- Seccomp/AppArmor profiles

**Layer 4: Data**  
- No sensitive data in logs  
- Sanitize error messages (no stack traces to clients)

### Secure Defaults

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
```

## Observability Strategy

### Three Pillars

**1. Metrics** (Prometheus)  
- **RED Method**: Rate, Errors, Duration per endpoint  
- **USE Method**: Utilization, Saturation, Errors for resources  
- **Custom**: Recipe cache hit rate, collector success rate

**2. Logs** (Structured JSON via slog)  
- **Levels**: DEBUG, INFO, WARN, ERROR  
- **Context**: Request ID, user ID, trace ID  
- **Sampling**: 1/100 for DEBUG in production

**3. Traces** (OpenTelemetry - future)  
- **Spans**: HTTP request → Collectors → Serialization  
- **Baggage**: Request metadata propagation  
- **Sampling**: Probability-based (10% of requests)

### SLIs and SLOs

| SLI | SLO | Alert Threshold |
|-----|-----|----------------|
| Availability | 99.9% | < 99.5% over 5m |
| Latency (p99) | < 100ms | > 200ms over 5m |
| Error Rate | < 0.1% | > 1% over 5m |
| Rate Limit Rejects | < 5% | > 10% over 5m |

**Error Budget**: 43 minutes downtime per month (99.9% SLO)

## Bundler Framework Architecture

### Overview

The Bundler Framework provides an extensible system for generating deployment bundles from configuration recipes. It follows a registry-based pattern allowing multiple bundler implementations to coexist and be selected at runtime.

### Design Principles

**1. Registry Pattern**  
**Rationale**: Decoupled bundler registration; extensibility without modifying core code  
**Implementation**: Global registry with type-safe bundler lookup  
**Trade-off**: Runtime registration vs compile-time safety

**2. Functional Options**  
**Pattern**: Configuration via variadic option functions  
**Rationale**: Optional parameters without constructor bloat; backward compatibility  
**Reference**: [Self-referential functions and the design of options](https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html)

**3. Template-based Generation**  
**Pattern**: Embedded templates with go:embed, data-driven rendering  
**Rationale**: Separation of structure (templates) from logic (Go code)  
**Implementation**: text/template with custom functions, embedded at compile time

**4. Parallel Execution**  
**Pattern**: errgroup.WithContext for concurrent bundle generation  
**Rationale**: Fast bundle creation; configurable fail-fast behavior  
**Implementation**: All bundlers execute in parallel by default  
**Trade-off**: Memory usage vs latency; coordination overhead vs throughput

### Component Architecture

```mermaid
flowchart TD
    A[Recipe] --> B[Bundler.Make]
    
    B --> B1[Validate Recipe]
    B --> B2[Build Config Map]
    B --> B3[Create Output Dir]
    
    B3 --> C{Parallel Generation}
    
    C --> C1[Generate Helm Values]
    C --> C2[Generate Manifests]
    C --> C3[Generate Scripts]
    C --> C4[Generate README]
    C --> C5[Generate Checksums]
    
    C1 --> D1[Extract K8s/GPU Data]
    C2 --> D2[Extract Config Flags]
    C3 --> D3[Extract Metadata]
    C4 --> D4[Render Templates]
    C5 --> D5[Compute SHA256]
    
    D1 --> E[Bundle Result]
    D2 --> E
    D3 --> E
    D4 --> E
    D5 --> E
    
    E --> F[Write Files]
    F --> G[Record Metrics]
    G --> H[Return BundleResult]
```

### Bundler Interface

```go
// Bundler generates deployment bundles from recipes.
type Bundler interface {
    // Make generates a bundle in the specified directory.
    // Returns a Result containing generated files, sizes, and any errors.
    Make(ctx context.Context, recipe *recipe.Recipe, outputDir string) (*Result, error)
}

// Optional interfaces that bundlers may implement:

// Validator allows bundlers to validate recipes before generation.
type Validator interface {
    Validate(ctx context.Context, recipe *recipe.Recipe) error
}

// Configurable allows bundlers to accept custom configuration.
type Configurable interface {
    Configure(config *Config) error
}
```

### GPU Operator Bundler

**Purpose**: Generate complete GPU Operator deployment bundle from recipe  
**Output**: Helm values, ClusterPolicy manifest, scripts, README, checksums

**Data Extraction**:
- **K8s Measurements**:
  - `image` subtype → component versions (gpu-operator, driver, dcgm, etc.)
  - `config` subtype → boolean flags (cdi, mig, rdma, useOpenKernelModule)
- **GPU Measurements**:
  - `smi` subtype → driver version, CUDA version

**Key Features**:
1. **Recipe Structure Alignment**: Matches actual recipe.yaml structure (not legacy policy/cluster)
2. **Boolean Handling**: Correctly interprets boolean flags vs string values
3. **Version Extraction**: Extracts all component versions from image subtype
4. **CDI Support**: Reads CDI configuration from config subtype
5. **Template Rendering**: Uses embedded templates for all output files

**Bundle Contents**:
```
gpu-operator/
├── values.yaml              # Helm chart values
├── manifests/
│   └── clusterpolicy.yaml  # Kubernetes ClusterPolicy
├── scripts/
│   ├── install.sh          # Installation script
│   └── uninstall.sh        # Uninstallation script
├── README.md                # Deployment instructions
└── checksums.txt            # SHA256 checksums
```

### Example: Adding a New Bundler

```go
package networkoperator

import (
    "context"
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler"
    "github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const bundlerType = bundler.BundleType("network-operator")

type Bundler struct {
    config *bundler.BundlerConfig
}

func init() {
    // Register bundler on package import
    bundler.Register(bundlerType, func() bundler.Bundler {
        return NewBundler()
    })
}

func NewBundler(opts ...Option) *Bundler {
    b := &Bundler{
        config: bundler.DefaultBundlerConfig(),
    }
    for _, opt := range opts {
        opt(b)
    }
    return b
}

func (b *Bundler) Type() bundler.BundleType {
    return bundlerType
}

func (b *Bundler) Validate(ctx context.Context, recipe *recipe.Recipe) error {
    // Check for required K8s measurements
    return bundler.ValidateRecipeStructure(recipe)
}

func (b *Bundler) Make(ctx context.Context, recipe *recipe.Recipe, 
    outputDir string) (*bundler.BundleResult, error) {
    
    result := bundler.NewBundleResult(bundlerType)
    
    // Generate bundle files in parallel
    g, ctx := errgroup.WithContext(ctx)
    
    g.Go(func() error {
        return b.generateHelmValues(ctx, recipe, outputDir, result)
    })
    
    g.Go(func() error {
        return b.generateManifests(ctx, recipe, outputDir, result)
    })
    
    if err := g.Wait(); err != nil {
        return nil, err
    }
    
    return result, nil
}

func (b *Bundler) Configure(config *bundler.BundlerConfig) error {
    b.config = config
    return nil
}
```

### Usage Example

```go
package main

import (
    "context"
    "github.com/NVIDIA/cloud-native-stack/pkg/bundler"
    _ "github.com/NVIDIA/cloud-native-stack/pkg/bundler/gpuoperator"
)

func main() {
    ctx := context.Background()
    
    // Create recipe
    recipe := createRecipe()
    
    // Get registered bundlers
    bundlers := bundler.GetBundlers("gpu-operator")
    
    // Generate bundles
    output, err := bundler.Make(ctx, recipe, "./output",
        bundler.WithBundlers(bundlers),
        bundler.WithParallel(true),
    )
    
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Generated %d files (%d bytes)\n", 
        output.TotalFiles, output.TotalBytes)
}
```

### Metrics and Observability

**Bundler Metrics** (Prometheus):
```go
bundler_make_duration_seconds{bundler_type="gpu-operator"}
bundler_make_total{bundler_type="gpu-operator",result="success|error"}
bundler_files_generated_total{bundler_type="gpu-operator"}
bundler_bytes_generated_total{bundler_type="gpu-operator"}
bundler_validation_failures_total{bundler_type="gpu-operator"}
```

**Logging** (Structured with slog):
- Bundle generation start/complete
- Per-bundler execution time
- File creation events
- Validation errors with context

### Testing Strategy

**1. Unit Tests**  
- Template rendering with test data
- Version extraction from recipe measurements
- Configuration validation
- Error handling paths

**2. Integration Tests**  
- Full bundle generation from realistic recipes
- File system operations
- Parallel execution correctness
- Thread-safety verification

**3. Table-Driven Tests**  
- Multiple recipe structures
- Various configuration combinations
- Edge cases (empty data, missing subtypes)

**Example Test**:
```go
func TestBundler_Make(t *testing.T) {
    tests := []struct {
        name    string
        recipe  *recipe.Recipe
        wantErr bool
        wantFiles []string
    }{
        {
            name: "complete recipe",
            recipe: createTestRecipe(),
            wantFiles: []string{
                "values.yaml",
                "manifests/clusterpolicy.yaml",
                "scripts/install.sh",
                "README.md",
                "checksums.txt",
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            b := NewBundler()
            result, err := b.Make(ctx, tt.recipe, tmpDir)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Make() error = %v, wantErr %v", err, tt.wantErr)
            }
            
            // Verify expected files exist
            for _, file := range tt.wantFiles {
                path := filepath.Join(tmpDir, "gpu-operator", file)
                if _, err := os.Stat(path); os.IsNotExist(err) {
                    t.Errorf("expected file not found: %s", file)
                }
            }
        })
    }
}
```

### Future Enhancements

**1. Additional Bundlers**  
- Network Operator (RDMA, SR-IOV configuration)
- Storage Operator (CSI drivers, volume configuration)
- NIM Operator (Inference microservices deployment)
- Nsight Operator (Profiling and debugging tools)

**2. Template Management**  
- External template loading (not just embedded)
- Template versioning and compatibility checks
- Template validation and testing framework

**3. Bundle Composition**  
- Multi-bundler orchestration
- Dependency resolution between bundles
- Unified checksums across all bundles

**4. Distribution**  
- Bundle packaging (tar.gz, OCI images)
- Signature verification (cosign, GPG)
- Registry push/pull for bundle artifacts

## References and Further Reading

### Official Go Documentation
- [Effective Go](https://go.dev/doc/effective_go)  
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)  
- [Context Package](https://pkg.go.dev/context)  
- [Error Handling](https://go.dev/blog/error-handling-and-go)

### Distributed Systems
- [Designing Data-Intensive Applications](https://dataintensive.net/) by Martin Kleppmann  
- [Site Reliability Engineering](https://sre.google/books/) by Google  
- [Building Microservices](https://www.oreilly.com/library/view/building-microservices-2nd/9781492034018/) by Sam Newman

### Kubernetes
- [Kubernetes Patterns](https://www.oreilly.com/library/view/kubernetes-patterns/9781492050278/)  
- [Programming Kubernetes](https://www.oreilly.com/library/view/programming-kubernetes/9781492047094/)  
- [client-go Documentation](https://github.com/kubernetes/client-go/tree/master/examples)

### Observability
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)  
- [OpenTelemetry Go SDK](https://pkg.go.dev/go.opentelemetry.io/otel)  
- [Structured Logging](https://pkg.go.dev/log/slog)

See individual architecture documents for detailed diagrams and component interactions.
