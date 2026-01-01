# Cloud Native Stack Roadmap

> **Project Mission**: Deliver production-grade, validated GPU-accelerated Kubernetes configurations through opinionated, battle-tested blueprints that ensure reproducible deployments across diverse platforms.

> This is only a draft document, does not represent the official roadmap of this project until fully reviewed. 

## Table of Contents

- [Next Release - Blocking adoption, major user pain points, security issues](#next-release)
- [High Priority - Important features that must land before launch](#launch-blockers)
- [Backlog](#backlog)

---

## Next Release

### PVC-Based Agent Output

**User Story**: As a cluster operator, I want agent snapshots persisted to a PVC that survives Job lifecycle, so I can access historical snapshots and compare configuration drift over time.

**Problem**: Current Job uses stdout/emptyDir, files lost when Job completes or is deleted. User must parse the Snapshot and Recipe from the Job logs.

**Solution**: Optional PVC mode for agent Job with timestamped file storage.

**Acceptance Criteria**:
- [ ] Create optional PVC manifest (in deployments/eidos-agent/1-deps.yaml) with RWO/RWX support
- [ ] Update Job to use PVC when available (volume mount at `/data`)
- [ ] Write files with timestamps: `snapshot-YYYYMMDD-HHMMSS.yaml`, `recipe-YYYYMMDD-HHMMSS.yaml`
- [ ] Document access pattern: helper pod or kubectl cp alternative
- [ ] Support both stdout (current) and PVC (new) modes
- [ ] Add example multi-cluster auditing script using PVC
- [ ] Storage class configurable (default: `standard`)

---

### Remote Snapshot from CLI

**User Story**: As a platform engineer, I want to capture cluster snapshots from my workstation without deploying a Job, so I can quickly audit remote GPU clusters without manual Job management.

**Problem**: Current workflow requires:
1. Deploy RBAC (1-deps.yaml)
2. Deploy Job (2-job.yaml)
3. Wait for completion
4. Retrieve logs
5. Clean up Job

**Solution**: Add `--remote` flag to `eidos snapshot` that orchestrates Job lifecycle programmatically.

**Acceptance Criteria**:
- [ ] `eidos snapshot --remote` deploys Job from embedded manifest
- [ ] Auto-checks RBAC exists, applies if missing (with confirmation)
- [ ] Waits for Job completion with progress indicator
- [ ] Streams logs to stdout or saves to file
- [ ] Auto-cleanup Job on success (configurable with `--keep`)
- [ ] Supports `--intent` to generate recipe automatically
- [ ] Works with existing KUBECONFIG
- [ ] Error handling for missing permissions, network issues

---

## Launch Blockers

### Differential Snapshots & Drift Detection

**User Story**: As a compliance officer, I want to detect configuration drift between clusters, so I can ensure consistency across environments and catch unauthorized changes.

**Acceptance Criteria**:
- [ ] `eidos diff baseline.yaml current.yaml` command
- [ ] Output formats: human-readable, JSON Patch (RFC 6902), table
- [ ] Highlight critical differences (GPU driver, K8s version, security settings)
- [ ] Exit code: 0=identical, 1=drifted
- [ ] Integration examples for CI/CD drift detection
- [ ] Ignore rules for expected differences (timestamps, unique IDs)

---

### Additional Bundlers

#### NIM Operator Bundler

**User Story**: As an ML engineer, I want to deploy NVIDIA Inference Microservices using CNS bundles, so I can quickly set up optimized inference infrastructure.

**Acceptance Criteria**:
- [ ] Bundler generates NIM Operator Helm values
- [ ] Supports model configuration (Llama, Mistral, etc.)
- [ ] Includes resource requests/limits based on GPU type
- [ ] Installation scripts with prerequisites check
- [ ] README with deployment examples

**References**: [NIM Operator Docs](https://docs.nvidia.com/nim-operator/)

#### Nsight Operator Bundler

**User Story**: As a performance engineer, I want to deploy Nsight profiling tools via CNS bundles, so I can debug GPU workload performance.

**Acceptance Criteria**:
- [ ] Bundler generates Nsight Operator manifests
- [ ] Includes RBAC for profiling permissions
- [ ] Installation scripts with validation
- [ ] Examples for common profiling scenarios
 
**References**: [Nsight Operator NGC](https://catalog.ngc.nvidia.com/orgs/nvidia/teams/devtools/helm-charts/nsight-operator)

#### KServe Bundler

**User Story**: As a platform team, I want to deploy KServe inference platform using CNS bundles, so I can provide MLOps teams with standard inference infrastructure.

**Acceptance Criteria**:
- [ ] Bundler generates KServe + Istio + Cert-Manager manifests
- [ ] Includes MetalLB configuration for LoadBalancer
- [ ] Storage provisioner setup (NFS/Local Path)
- [ ] Installation script with dependency checks
- [ ] Examples for common inference patterns

**References**: [KServe Docs](https://kserve.github.io/)

#### Storage Bundler

**User Story**: As a cluster admin, I want to deploy storage provisioners via CNS bundles, so I can provide persistent storage for AI workloads.

**Acceptance Criteria**:
- [ ] Local Path Provisioner configuration
- [ ] NFS Provisioner with server setup
- [ ] Storage class definitions
- [ ] PVC examples for common patterns

#### Monitoring Bundler

**User Story**: As an SRE, I want to deploy Prometheus/Grafana/Elastic stack via CNS bundles, so I can monitor GPU infrastructure health.

**Acceptance Criteria**:
- [ ] Prometheus stack with GPU metrics
- [ ] Grafana dashboards for GPU utilization
- [ ] Elastic stack for log aggregation
- [ ] Default passwords and access instructions

---

### Schema Validation

**User Story**: As a CI/CD pipeline developer, I want to validate snapshots against API version schemas, so I can catch malformed data before downstream processing.

**Acceptance Criteria**:
- [ ] `eidos validate --schema v1 snapshot.yaml` command
- [ ] Embed JSON Schema with go:embed
- [ ] Library: `github.com/santhosh-tekuri/jsonschema/v5`
- [ ] Validation errors with line numbers and paths
- [ ] Exit code: 0=valid, 1=invalid
- [ ] CI/CD integration examples

## Backlog

---

### Caching Layer

**User Story**: As a CI/CD pipeline operator, I want snapshot caching to avoid redundant collection, so I can speed up repeated eidos calls in scripts.

**Acceptance Criteria**:
- [ ] In-memory cache with TTL (default: 5 minutes)
- [ ] `--cache-ttl` flag to configure TTL
- [ ] `--no-cache` flag to force fresh collection
- [ ] Cache key: hash of collection parameters
- [ ] Cache invalidation on TTL expiry
- [ ] Performance: 5-10x improvement for cached calls

---

### Measurement Filtering

**User Story**: As a developer, I want to capture only specific measurement types, so I can reduce snapshot size and collection time for targeted use cases.

**Acceptance Criteria**:
- [ ] `eidos snapshot --filter gpu,os` includes only specified types
- [ ] `eidos snapshot --exclude k8s` excludes specific types
- [ ] Validation: error on unknown type names
- [ ] Performance: skip excluded collectors entirely (not post-filter)
- [ ] Documentation: common filter patterns (GPU-only, K8s-only, etc.)

---

### Configuration Files

**User Story**: As a power user, I want persistent configuration for common flags, so I don't repeat --os, --gpu, --format in every command.

**Acceptance Criteria**:
- [ ] Config file location: `~/.config/eidos/config.yaml` (XDG spec)
- [ ] Defaults section for common flags
- [ ] Server URL configuration for API mode
- [ ] Command-line flags override config file
- [ ] `eidos config init` to generate template
- [ ] Validation on config load

---

### Watch Mode

**User Story**: As a monitoring system, I want continuous snapshot monitoring with change detection, so I can alert on configuration drift in real-time.

**Acceptance Criteria**:
- [ ] `eidos snapshot --watch --interval 30s --on-change ./alert.sh`
- [ ] Stream of JSON diffs to stdout
- [ ] Configurable polling interval
- [ ] Hook script execution on change detection
- [ ] Signal handling for graceful shutdown

---

### Platform-Specific Optimizations in Recipes

**User Story**: As a GB200 operator, I want CNS recipes to include platform-specific optimizations, so I automatically get best practices for my hardware.

**Problem**: Current recipes are generic. GB200 has specific boot parameters, NUMA settings, IOMMU configs not captured in base recipes.

**Solution**: Enhance recipe overlay system with hardware-specific optimizations.

**Acceptance Criteria**:
- [ ] GB200 overlay with boot parameters (`init_on_alloc=0`, `numa_balancing=disable`)
- [ ] H100 overlay with optimizations
- [ ] Recipe includes OS-level tuning (grub, sysctl)
- [ ] Bundle install scripts apply optimizations automatically
- [ ] Documentation: optimization rationale and trade-offs

**References**: `docs/v1/optimizations/GB200-NVL72.md`

---

### Multi-Bundler Orchestration

**User Story**: As a platform engineer, I want CNS to generate all required bundles in one command with dependency resolution, so I don't manually track component dependencies.

**Problem**: Current bundlers are independent. KServe needs storage + loadbalancer + monitoring but no dependency tracking.

**Acceptance Criteria**:
- [ ] Bundler dependency declaration in metadata
- [ ] `eidos bundle --recipe recipe.yaml --with-deps` resolves dependencies
- [ ] Topological sort for installation order
- [ ] Unified install script across bundles
- [ ] Conflict detection (incompatible bundler combinations)

---

### Bundle Distribution & Packaging

**User Story**: As a disconnected environment operator, I want to package bundles as OCI images, so I can distribute them in air-gapped clusters.

**Acceptance Criteria**:
- [ ] `eidos bundle --package --output bundle.tar.gz` creates tarball
- [ ] OCI image format for registry storage
- [ ] Signature verification with cosign
- [ ] `eidos bundle --push registry/bundle:v1.0.0` pushes to registry
- [ ] `eidos bundle --pull registry/bundle:v1.0.0` downloads bundle

---

### Cloud Provider Integration

**User Story**: As a cloud architect, I want CNS to query cloud provider APIs for metadata, so I get accurate instance types, regions, and cloud-specific configurations.

**Acceptance Criteria**:
- [ ] AWS: Query EC2 instance metadata, EKS cluster details
- [ ] GCP: Query GCE metadata, GKE cluster configuration
- [ ] Azure: Query AKS metadata, VM instance details
- [ ] Automatic cloud detection (IMDS endpoints)
- [ ] Fallback to manual collection if IMDS unavailable

---

### Policy Enforcement

**User Story**: As a security engineer, I want to enforce configuration policies in CI/CD, so non-compliant configs are blocked before deployment.

**Acceptance Criteria**:
- [ ] `eidos validate --policy policy.rego snapshot.yaml` command
- [ ] OPA integration with embedded engine
- [ ] Policy examples: GPU driver version constraints, K8s version requirements
- [ ] Exit code: 0=compliant, 1=violations
- [ ] Detailed violation reports with remediation hints

**References**: [OPA Go Integration](https://www.openpolicyagent.org/docs/latest/integration/)

---

### Cloud Storage Integration

**User Story**: As a fleet manager, I want snapshots automatically uploaded to S3/GCS, so I can centralize configuration data across hundreds of clusters.

**Acceptance Criteria**:
- [ ] `eidos snapshot --upload s3://bucket/snapshots/$(hostname).yaml`
- [ ] Support: S3, GCS, Azure Blob
- [ ] Resumable uploads for large snapshots
- [ ] Authentication: IAM roles, service accounts, credential chain
- [ ] Retry logic with exponential backoff

**References**: [AWS SDK for Go V2](https://aws.github.io/aws-sdk-go-v2/)

### Plugin System

**User Story**: As an enterprise, I want to add custom collectors without forking CNS, so I can capture proprietary metrics and configurations.

**Acceptance Criteria**:
- [ ] Collector interface for plugins
- [ ] WASM-based sandboxed execution (not Go plugins - version instability)
- [ ] Restricted syscalls for security
- [ ] Plugin discovery and loading
- [ ] Examples for common custom collectors

**References**: [WASI](https://wasi.dev/)

---

### gRPC API Mode

**User Story**: As a high-throughput system, I want gRPC API with streaming for better performance, so I can handle 10x more requests with smaller payloads.

**Acceptance Criteria**:
- [ ] Protobuf definitions for Recipe service
- [ ] Bi-directional streaming support
- [ ] HTTP/2 multiplexing
- [ ] 3-5x smaller payloads vs JSON
- [ ] Backward compatible (run gRPC + HTTP on same port with cmux)

**References**: [gRPC Go](https://grpc.io/docs/languages/go/quickstart/)

---

### GraphQL API

**User Story**: As a frontend developer, I want GraphQL API to request only needed fields, so I reduce bandwidth and improve mobile app performance.

**Acceptance Criteria**:
- [ ] GraphQL schema for Recipe and Measurement types
- [ ] Field-level filtering
- [ ] Query depth limiting (prevent abuse)
- [ ] Backward compatible with REST API
 
**References**: [GraphQL Go](https://graphql.org/code/#go)

---

### Distributed Tracing

**User Story**: As an SRE, I want OpenTelemetry tracing for performance debugging, so I can identify slow collectors and optimize collection pipelines.

**Acceptance Criteria**:
- [ ] OpenTelemetry SDK integration
- [ ] Span per collector with attributes
- [ ] OTLP exporter to Jaeger/Tempo
- [ ] `--trace` flag to enable tracing
- [ ] Sampling configuration

**References**: [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)

---

### Multi-Tenancy for API Server

**User Story**: As a SaaS provider, I want per-customer isolation with separate rate limits, so I can safely serve multiple tenants from one deployment.

**Acceptance Criteria**:
- [ ] Tenant ID from API key header
- [ ] Per-tenant rate limiting
- [ ] Per-tenant metrics
- [ ] Tenant-specific recipe customization

---

## Revision History

- **2025-01-01**: Initial comprehensive roadmap based on project objectives and gap analysis 