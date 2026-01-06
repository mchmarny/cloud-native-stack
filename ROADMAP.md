# Cloud Native Stack Roadmap

> Note: roadmap is under development and does not represent committed deliverables. Items may change based on community feedback and project priorities.

**Project Scope**: Generate validated configurations for GPU-accelerated Kubernetes deployments. Configurations are tested against NVIDIA hardware (H100, GB200, A100, ...) and support managed Kubernetes offerings (e.g. Amazon EKS, Google GKE, Azure AKS, Oracle OKE, ...) as well as self-managed clusters. 

## Table of Contents

- [Next Release - Blocking adoption, major user pain points, security issues](#next-release)
- [High Priority - Important features that must land before launch](#launch-blockers)
- [Backlog](#backlog)

See [Opens](#opens) section for list of still to be decided architectural decisions, implementation questions, and discussion topics. These items represent areas where community input, design trade-offs, or further exploration would be valuable.

---

## Next Release

### Validate Current Bundlers

**Scope**: Review the GPU Operator, Network Operator bundlers against known good deployments

**Acceptance Criteria**:
- [ ] GPU Operator and Network Operator bundlers generate valid deployments from recipe measurements
- [ ] README documents deployment steps and configuration options

### Additional Bundlers

#### Skyhook

**Scope**: Generate Helm values and manifests for Skyhook Operator deployment.

**Acceptance Criteria**:
- [ ] Bundler generates Skyhook Operator Helm values from recipe measurements
- [ ] README documents deployment steps and configuration options

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

### Additional Bundlers

**Current bundlers**: GPU Operator, Network Operator, Skyhook

**Proposed bundlers** (not yet implemented):

#### NIM Operator Bundler

**Scope**: Generate Helm values and manifests for NVIDIA Inference Microservices (NIM) Operator deployment.

**Acceptance Criteria**:
- [ ] Bundler generates NIM Operator Helm values from recipe measurements
- [ ] Supports model configuration for common inference workloads
- [ ] Includes resource requests/limits derived from GPU type in recipe
- [ ] Installation script validates prerequisites (kubectl, helm, NIM license)
- [ ] README documents deployment steps and configuration options

**Reference**: [NIM Operator Documentation](https://docs.nvidia.com/nim-operator/)

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

### PVC-Based Agent Output

**Current behavior**: Agent Job writes snapshot to a ConfigMap (`eidos-snapshot` in `gpu-operator` namespace). ConfigMap is overwritten on each Job run. Historical snapshots are not retained.

**Proposed change**: Add optional PersistentVolumeClaim (PVC) output mode. When enabled, agent writes timestamped files to a persistent volume instead of ConfigMap.

**Motivation**: Persistent storage enables historical snapshot retention and drift detection across multiple Job runs.

**Acceptance Criteria**:
- [ ] Create optional PVC manifest (in deployments/eidos-agent/1-deps.yaml) with RWO/RWX support
- [ ] Update Job to use PVC when available (volume mount at `/data`)
- [ ] Write files with timestamps: `snapshot-YYYYMMDD-HHMMSS.yaml`, `recipe-YYYYMMDD-HHMMSS.yaml`
- [ ] Document access pattern: helper pod or kubectl cp alternative
- [ ] Support both stdout (current) and PVC (new) modes
- [ ] Add example multi-cluster auditing script using PVC
- [ ] Storage class configurable (default: `standard`)

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

## Opens

This section outlines key architectural decisions, implementation questions, and discussion topics for Cloud Native Stack. These items represent areas where community input, design trade-offs, or further exploration would be valuable.

### Architecture & Design

**Should snapshot collection support best-effort mode for partial data scenarios?**  
Current implementation uses fail-fast: if any collector (K8s, GPU, OS, SystemD) fails, the entire snapshot fails. Alternative is best-effort mode where partial data is returned with incomplete markers. Trade-offs: fail-fast ensures data consistency but may be too strict for degraded environments; best-effort improves availability but complicates downstream validation and could mask real issues requiring attention.

**How should we handle collector execution order and dependencies?**  
Currently all collectors run in parallel via `errgroup` for speed. Some collectors might benefit from sequential execution or conditional execution based on previous results. Trade-offs: parallel execution maximizes speed but prevents dependency resolution; sequential execution enables smarter collection but increases latency. Should we add collector metadata for optional dependency declarations?

**What is the right caching strategy for recipe data and collected snapshots?**  
Recipe store is currently loaded once at initialization. Snapshots are collected fresh each time. Adding caching (in-memory with TTL) could speed up repeated calls in CI/CD pipelines (5-10x improvement). Trade-offs: cached data may be stale; cache invalidation complexity; memory overhead for large snapshots. Need clear cache key strategy (hash of collection parameters) and explicit `--no-cache` flag for critical operations.

**Should we support incremental snapshots that only capture changed data?**  
Current snapshots are always complete, capturing all measurements. Incremental snapshots could significantly reduce size and collection time by only including changed data since last snapshot. This requires baseline storage and diff calculation. Trade-offs: smaller payloads and faster collection vs added complexity in diff calculation, baseline management, and potential for missed changes if baseline is corrupted or unavailable.

### Storage & Distribution

**How should bundles be packaged and distributed in disconnected environments?**  
Current bundles are directory-based with files and checksums. Disconnected/air-gapped environments need portable bundle formats. Options: (1) tar.gz archives with signature verification, (2) OCI image format for registry storage with cosign signatures, (3) custom package format with metadata. Trade-offs: OCI provides existing infrastructure but adds complexity; tar.gz is simple but requires separate signature mechanism; custom format offers control but requires tooling ecosystem.

**What storage backends should agent snapshots support beyond ConfigMaps?**  
Agent currently writes to ConfigMap (`cm://namespace/name`). PVC-based storage enables historical retention and drift detection. Other options: S3/GCS for centralized multi-cluster storage, custom CRDs for structured data with validation. Trade-offs: ConfigMaps are simple and native but size-limited (1MB); PVCs enable large snapshots but require storage provisioning; cloud storage centralizes data but adds external dependencies; CRDs provide validation but require custom controllers. Any time we push captured data out of the cluster there is also the perception issue. 

**Should recipe data remain embedded or support external data sources?**  
Recipe data is currently embedded in binary (via `go:embed`) from `pkg/recipe/data/data-v1.yaml`, making the system self-contained with zero external dependencies. External sources (HTTP URL, git repository, ConfigMap) would enable runtime updates without recompilation. Trade-offs: embedded data ensures consistency and zero runtime dependencies but requires recompilation for updates; external sources enable dynamic updates but introduce network dependencies, version skew risks, and cache invalidation challenges.

### Bundle Generation & Orchestration

**How should bundlers declare and resolve dependencies on each other?**  
Current bundlers execute independently in parallel. Some bundlers have implicit dependencies (e.g., KServe needs storage provisioner, load balancer, monitoring). Options: (1) declarative dependency metadata in bundler registration, (2) topological sort for installation order, (3) unified installation orchestrator. Trade-offs: independent bundlers are simple but require manual coordination; dependency resolution automates ordering but adds complexity; orchestrator provides full control but becomes single point of failure.

**Should bundle generation support template customization or extension points?**  
Current bundlers use embedded templates (`go:embed`) that cannot be modified without code changes. Users might want to customize templates for organizational standards (labels, annotations, registry URLs). Options: (1) external template overlays, (2) template variables in config files, (3) post-generation hooks. Trade-offs: embedded templates ensure consistency but limit customization; external overlays enable flexibility but introduce version compatibility issues; hooks provide power but can break bundles if misused.

**What is the right level of abstraction for cross-bundle configurations?**  
Bundles currently operate independently with separate configuration. Shared settings (registry URLs, namespace names, image pull secrets) are duplicated across bundles. Options: (1) shared configuration file with common values, (2) bundle composition layer that applies common transforms, (3) base bundle that others extend. Trade-offs: shared config reduces duplication but creates implicit dependencies; composition layer is flexible but complex; base bundle inheritance provides structure but limits independent evolution.

### Validation & Policy

**How should configuration validation be integrated into the workflow?**  
Currently no built-in validation for snapshots or recipes beyond basic schema checks. Users need policy enforcement (e.g., minimum K8s version, required GPU driver versions, security settings). Options: (1) JSON Schema validation embedded in CLI, (2) OPA integration with embedded policy engine, (3) admission webhooks for Kubernetes resources. Trade-offs: JSON Schema is simple but limited to structure; OPA provides full policy language but adds binary size; admission webhooks enforce at deploy time but require cluster infrastructure.

**Should bundles include validation and smoke tests for deployment verification?**  
Generated bundles contain installation scripts but no verification. Post-deployment validation would confirm operator readiness and basic functionality. Options: (1) include validation scripts that check pod status and CRD creation, (2) generate smoke test workloads (simple GPU job), (3) integration with test frameworks (Ginkgo). Trade-offs: validation scripts are simple but limited; smoke tests provide confidence but increase bundle size; test framework integration is powerful but requires dependencies.

### Data Collection & Observability

**What level of context and provenance should measurements include?**  
Measurements currently capture values but minimal context about collection method or source. Richer context (collection timestamp, source file/API, validation status, confidence level) would improve debugging and trustworthiness. Trade-offs: detailed context increases snapshot size significantly (potentially 2-3x); collection overhead for metadata gathering; but provides crucial debugging information and audit trail for compliance scenarios.

**How should the system handle collection failures and partial data?**  
Some environments have intermittent failures (Kubernetes API timeouts, GPU SMI hangs, missing systemd services). Current fail-fast approach might be too strict. Options: (1) continue-on-error mode with failure annotations, (2) retry with exponential backoff, (3) degraded mode indicators. Trade-offs: continue-on-error improves success rate but downstream tools must handle partial data; retries increase latency; degraded mode is complex to implement and reason about.

**Should collectors support filtering to reduce snapshot size and collection time?**  
Current snapshot captures all measurement types (OS, SystemD, K8s, GPU). Large clusters generate large snapshots. Filtering (`--include gpu,os` or `--exclude k8s`) would reduce size and collection time for targeted use cases. Trade-offs: filtering reduces overhead but requires collectors to support conditional execution; downstream tools must handle incomplete snapshots; risk of excluding needed data if filter is too aggressive.

### Performance & Scalability

**How should the system handle very large Kubernetes clusters (1000+ nodes)?**  
K8s collector lists all pods, nodes, and images which can be expensive in large clusters. Options: (1) pagination for API requests, (2) field selectors to filter data at API level, (3) sampling strategies for large datasets, (4) distributed collection with multiple agents. Trade-offs: pagination reduces memory but increases latency; field selectors are efficient but might miss needed data; sampling loses completeness; distributed collection is complex but scales.

### Developer Experience & Extensibility

**Should the project support plugin-based extensibility for custom collectors?**  
Current collectors are compiled into binary. Organizations with proprietary metrics or configurations cannot extend without forking. Options: (1) Go plugins (unstable due to version requirements), (2) WASM-based sandboxed plugins, (3) external process execution with structured output, (4) gRPC-based plugin protocol. Trade-offs: Go plugins have version compatibility issues; WASM is secure but limited syscall access; external processes are flexible but slow; gRPC protocol is robust but adds complexity.

**What is the right level of abstraction for the bundler framework?**  
Current BaseBundler helper reduces boilerplate significantly (75% reduction). Could go further with even more opinionated conventions or pull back for more flexibility. Options: (1) full framework with convention-over-configuration, (2) current helper approach, (3) minimal interface with examples. Trade-offs: full framework is fastest for common cases but constraining; helper approach balances speed and flexibility; minimal interface provides freedom but requires more code.

### Migration & Compatibility

**How should the project handle backward compatibility with v1 documentation-based approach?**  
Legacy documentation in `docs/v1/` represents previous manual installation approach. Some users still use Ansible playbooks and install guides. Support strategy needed: (1) maintain both approaches indefinitely, (2) deprecation timeline with migration tools, (3) immediate removal after transition period. Trade-offs: dual maintenance is costly; deprecation provides transition time but delays simplification; immediate removal forces migration but may lose users.

**What is the strategy for recipe data versioning and API evolution?**  
Recipe format is currently `v1` with embedded data. Future versions might need schema changes, new measurement types, or different overlay rules. Options: (1) API version negotiation with multiple format support, (2) in-place evolution with backward compatibility, (3) explicit version bumps with migration tools. Trade-offs: multiple version support is complex but safe; in-place evolution is simpler but risks breaking changes; version bumps are clear but require migration tooling. 

---

## Revision History

- **2026-01-01**: Initial comprehensive roadmap based on project objectives and gap analysis 
- **2026-01-05**: Added [Opens section](#opens) based on architectural decisions, implementation questions
