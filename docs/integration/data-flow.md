# Data Flow Architecture

Understanding how data flows through the Cloud Native Stack system.

## Overview

Cloud Native Stack follows a three-stage data transformation pipeline:

```
System Config → Snapshot → Recipe → Bundle → Deployment
     (Raw)      (Capture)  (Optimize) (Package)  (Deploy)
```

Each stage transforms data into increasingly deployment-ready formats.

## Stage 1: Snapshot (Data Capture)

### Input Sources

**SystemD Services:**
- Source: `systemctl show containerd.service`
- Data: Service configuration, resource limits, cgroup delegates
- Format: Key-value pairs from SystemD properties

**OS Configuration:**
- **grub**: `/proc/cmdline` - Boot parameters
- **kmod**: `/proc/modules` - Loaded kernel modules
- **sysctl**: `/proc/sys/**/*` - Kernel runtime parameters
- **release**: `/etc/os-release` - OS identification

**Kubernetes Cluster:**
- Source: Kubernetes API via `client-go`
- **server**: Version info from `/version` endpoint
- **image**: Container images from all pods across namespaces
- **policy**: GPU Operator ClusterPolicy custom resource

**GPU Hardware:**
- Source: `nvidia-smi` command-line tool
- Data: Driver version, CUDA version, MIG settings, device info
- Format: Parsed XML/text output

### Snapshot Data Structure

```
┌─────────────────────────────────────────────────────────┐
│ Snapshot (snapshot.dgxc.io/v1)                          │
├─────────────────────────────────────────────────────────┤
│ metadata:                                               │
│   created: timestamp                                    │
│   hostname: string                                      │
│                                                         │
│ measurements: []Measurement                             │
│   ├─ SystemD                                            │
│   │   └─ subtypes: [containerd.service, ...]            │
│   │       └─ data: map[string]Reading                   │
│   │                                                     │
│   ├─ OS                                                 │
│   │   └─ subtypes: [grub, kmod, sysctl, release]        │
│   │       └─ data: map[string]Reading                   │
│   │                                                     │
│   ├─ K8s                                                │
│   │   └─ subtypes: [server, image, policy]              │
│   │       └─ data: map[string]Reading                   │
│   │                                                     │
│   └─ GPU                                                │
│       └─ subtypes: [smi, driver, device]                │
│           └─ data: map[string]Reading                   │
└─────────────────────────────────────────────────────────┘
```

**Output Destinations:**
- **File**: `eidos snapshot --output system.yaml`
- **Stdout**: `eidos snapshot` (default, pipe to other commands)
- **ConfigMap**: `eidos snapshot --output cm://namespace/name` (Kubernetes-native)

**ConfigMap Storage Pattern:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: eidos-snapshot
  namespace: gpu-operator
data:
  snapshot.yaml: |
    # Complete snapshot YAML stored as ConfigMap data
    apiVersion: snapshot.dgxc.io/v1
    kind: Snapshot
    measurements: [...]
```

**Agent Deployment:**  
Kubernetes Job writes snapshots directly to ConfigMap without volumes:
```bash
eidos snapshot --output cm://gpu-operator/eidos-snapshot
```

**Reading Interface:**
```go
type Reading interface {
    Any() interface{}      // Type-safe value extraction
    String() string        // String representation
    // Supports: int, string, bool, float64
}
```

### Collection Process

**Parallel Collection:**
```
┌──────────────┐
│ Snapshotter  │
└──────┬───────┘
       │ errgroup.WithContext()
       ├────────────┬─────────────┬─────────────┐
       │            │             │             │
  ┌────▼────┐   ┌───▼───┐     ┌───▼───┐     ┌───▼───┐
  │ SystemD │   │  OS   │     │  K8s  │     │  GPU  │
  │Collector│   │Collect│     │Collect│     │Collect│
  └────┬────┘   └───┬───┘     └───┬───┘     └───┬───┘
       │            │             │             │
       └────────────┴─────────────┴─────────────┘
                    │
              ┌─────▼──────┐
              │  Snapshot  │
              │   (YAML)   │
              └────────────┘
```

**Context Propagation:**
- All collectors respect context cancellation
- First error cancels remaining operations
- Timeout: 30 seconds per collector

## Stage 2: Recipe (Data Optimization)

### Recipe Input Options

**Query Mode** - Direct generation from parameters:
```bash
eidos recipe --os ubuntu --gpu h100 --service eks --intent training
```

**Snapshot Mode (File)** - Analyze captured snapshot:
```bash
eidos snapshot --output system.yaml
eidos recipe --snapshot system.yaml --intent training
```

**Snapshot Mode (ConfigMap)** - Read from Kubernetes:
```bash
# Agent or CLI writes snapshot to ConfigMap
eidos snapshot --output cm://gpu-operator/eidos-snapshot

# CLI reads from ConfigMap to generate recipe
eidos recipe --snapshot cm://gpu-operator/eidos-snapshot --intent training

# Recipe can also be written to ConfigMap
eidos recipe --snapshot cm://gpu-operator/eidos-snapshot \
            --intent training \
            --output cm://gpu-operator/eidos-recipe
```

### Query Extraction (Snapshot Mode)

When a snapshot is provided, the recipe builder extracts query parameters:

```
Snapshot → Query Extractor → Recipe Query
```

**Extraction mapping:**
```
K8s/server/version          → k8s (version)
K8s/image/gpu-operator      → service (eks/gke/aks detection)
K8s/config/*                → intent hints
OS/release/ID               → os (family)
OS/release/VERSION_ID       → osv (version)
OS/grub/BOOT_IMAGE          → kernel (version)
GPU/smi/model               → gpu (type)
```

### Recipe Generation

**Base + Overlay Merging:**
```
┌────────────────────────────────────────────────────────┐
│ Recipe Builder                                         │
├────────────────────────────────────────────────────────┤
│                                                        │
│  1. Load base measurements (universal config)          │
│     └─ From embedded data-v1.yaml                      │
│                                                        │
│  2. Match query to overlays                            │
│     ├─ service=eks, os=ubuntu                          │
│     ├─ service=eks, gpu=gb200                          │
│     └─ service=eks, gpu=h100                           │
│                                                        │
│  3. Merge measurements                                 │
│     ├─ Base values                                     │
│     ├─ + Overlay 1 (augment/override)                  │
│     ├─ + Overlay 2 (augment/override)                  │
│     └─ + Overlay N (augment/override)                  │
│                                                        │
│  4. Strip context (if !context)                        │
│     └─ Remove context maps from all subtypes           │
│                                                        │
│  5. Return recipe                                      │
│                                                        │
└────────────────────────────────────────────────────────┘
```

**Overlay Matching Algorithm:**
```go
// Overlay matches if all specified fields match query
// Omitted fields act as wildcards

overlay.key {
    service: "eks"   // Must match
    gpu: "gb200"     // Must match
    os: <omitted>    // Wildcard (any OS)
}

query {
    service: "eks"
    gpu: "gb200"
    os: "ubuntu"
}

Result: MATCH (os wildcarded)
```

### Recipe Data Structure

```
┌─────────────────────────────────────────────────────────┐
│ Recipe (recipe.dgxc.io/v1)                              │
├─────────────────────────────────────────────────────────┤
│ metadata:                                               │
│   created: timestamp                                    │
│   recipe-version: version                               │
│                                                         │
│ request: Query (original parameters)                    │
│                                                         │
│ matchedRules: []string (matched overlay keys)           │
│                                                         │
│ measurements: []Measurement                             │
│   ├─ SystemD                                            │
│   │   └─ subtypes + merged data                         │
│   ├─ OS                                                 │
│   │   └─ subtypes + merged data                         │
│   ├─ K8s                                                │
│   │   └─ subtypes + merged data (with recommendations)  │
│   └─ GPU                                                │
│       └─ subtypes + merged data                         │
└─────────────────────────────────────────────────────────┘
```

**Context Metadata (Optional):**
```yaml
data:
  iommu.passthrough: "1"
context:
  iommu.passthrough: "Bypass IOMMU translation for GB200"
```

When `context=false`, context maps are stripped before response.

## Stage 3: Bundle (Data Packaging)

### Bundler Framework

```
┌────────────────────────────────────────────────────────┐
│ Bundle Generator                                       │
├────────────────────────────────────────────────────────┤
│                                                        │
│  Recipe → Bundler Registry → Parallel Execution        │
│                                                        │
│  ┌─────────────────┐                                   │
│  │ Recipe Input    │                                   │
│  └────────┬────────┘                                   │
│           │                                            │
│  ┌────────▼────────┐                                   │
│  │ Extract Config  │ (buildConfigMap)                  │
│  │ ├─ K8s images   │                                   │
│  │ ├─ GPU settings │                                   │
│  │ ├─ OS params    │                                   │
│  │ └─ Network cfg  │                                   │
│  └────────┬────────┘                                   │
│           │                                            │
│    ┌──────┴──────┐                                     │
│    │   Parallel  │                                     │
│    ├─────────────┤                                     │
│    ├─ GPU Operator                                     │
│    │  ├─ GenerateHelmValues()                          │
│    │  ├─ GenerateManifestData()                        │
│    │  └─ GenerateScriptData()                          │
│    │                                                   │
│    └─ Network Operator                                 │
│       ├─ GenerateHelmValues()                          │
│       ├─ GenerateManifestData()                        │
│       └─ GenerateScriptData()                          │
│                                                        │
│  ┌────────▼────────┐                                   │
│  │ Template Engine │ (go:embed templates)              │
│  │ ├─ values.yaml  │                                   │
│  │ ├─ manifests/   │                                   │
│  │ ├─ scripts/     │                                   │
│  │ └─ README.md    │                                   │
│  └────────┬────────┘                                   │
│           │                                            │
│  ┌────────▼────────┐                                   │
│  │ Generate Files  │                                   │
│  │ └─ checksums    │                                   │
│  └─────────────────┘                                   │
│                                                        │
└────────────────────────────────────────────────────────┘
```

### Configuration Extraction

**ValueWithContext Pattern:**
```go
type ValueWithContext struct {
    Value   interface{}  // Actual configuration value
    Context string       // Human-readable explanation
}

// Bundlers extract values with context
DriverVersion := ValueWithContext{
    Value:   "580.82.07",
    Context: "NVIDIA GPU driver for H100",
}
```

**Template Usage:**
```yaml
# Helm values.yaml
driver:
  version: {{ .DriverVersion.Value }}
  
# README.md
Driver Version: {{ .DriverVersion.Value }}
Reason: {{ .DriverVersion.Context }}
```

### Bundle Structure

```
bundle-output/
├── gpu-operator/
│   ├── values.yaml           # Helm values with extracted config
│   ├── manifests/
│   │   └── clusterpolicy.yaml  # Kubernetes manifest
│   ├── scripts/
│   │   ├── install.sh        # Deployment automation
│   │   └── uninstall.sh      # Cleanup automation
│   ├── README.md              # Deployment guide
│   └── checksums.txt          # SHA256 verification
│
└── network-operator/
    ├── values.yaml
    ├── manifests/
    │   └── nicclusterpolicy.yaml
    ├── scripts/
    │   ├── install.sh
    │   └── uninstall.sh
    ├── README.md
    └── checksums.txt
```

## Data Serialization

### Formats Supported

**JSON:**
```json
{
  "apiVersion": "v1",
  "kind": "Recipe",
  "measurements": [...]
}
```

**YAML:**
```yaml
apiVersion: v1
kind: Recipe
measurements:
  - type: K8s
    subtypes: [...]
```

**Table (Human-readable):**
```
TYPE    SUBTYPE      KEY                    VALUE
K8s     image        gpu-operator           v25.3.3
K8s     image        driver                 580.82.07
GPU     driver       version                580.82.07
```

### Serialization Pipeline

```
Go Struct → Interface → Marshaler → Output Format

Measurement{
  Type: "K8s"
  Subtypes: []Subtype{...}
}
    │
    ▼
json.Marshal() / yaml.Marshal()
    │
    ▼
{"type":"K8s","subtypes":[...]}
```

## API Server Data Flow

### Request Processing

```
HTTP Request → Middleware Chain → Handler → Response

1. Metrics Middleware (record request)
2. Version Middleware (check API version)
3. RequestID Middleware (add/echo request ID)
4. Panic Recovery (catch panics)
5. Rate Limit (100 req/s)
6. Logging (structured logs)
7. Handler:
   ├─ Parse query parameters
   ├─ Build Query
   ├─ recipe.Builder.Build(ctx, query)
   ├─ Serialize response
   └─ Return JSON
```

### Response Headers

```
HTTP/1.1 200 OK
Content-Type: application/json
X-Request-Id: 550e8400-e29b-41d4-a716-446655440000
Cache-Control: public, max-age=300
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1735650000

{recipe JSON}
```

## Data Storage

### Embedded Data

**Recipe Data:**
- Location: `pkg/recipe/data/data-v1.yaml`
- Embedded at compile time: `//go:embed data/data-v1.yaml`
- Loaded once per process, cached in memory
- TTL: 5 minutes (in-memory cache)

**Bundle Templates:**
- Location: `pkg/bundler/*/templates/*.tmpl`
- Embedded at compile time: `//go:embed templates/*.tmpl`
- Parsed once per bundler initialization

**No External Dependencies:**
- No database
- No configuration files
- No network calls (except Kubernetes API for snapshots)
- Fully self-contained binaries

## Performance Characteristics

### Snapshot Collection

- **Parallel**: All collectors run concurrently
- **Timeout**: 30 seconds per collector
- **Memory**: ~10-50MB depending on cluster size
- **Duration**: 1-5 seconds typical

### Recipe Generation

- **Cached**: Recipe data cached in memory (5min TTL)
- **Overlay Matching**: O(n) where n = number of overlays
- **Memory**: <1MB per request
- **Duration**: <100ms typical (in-memory only)

### Bundle Generation

- **Parallel**: All bundlers run concurrently
- **Template Rendering**: Minimal overhead (<10ms per template)
- **File I/O**: ~10-20 files per bundler
- **Duration**: <1 second typical

### API Server

- **Concurrency**: 100 req/s sustained, 200 burst
- **Latency**: p50: 50ms, p95: 150ms, p99: 300ms
- **Memory**: ~100MB baseline + 1MB per concurrent request
- **CPU**: Minimal (<5% single core at 100 req/s)

## Data Validation

### Input Validation

**Query Parameters:**
- Type checking (string, int, bool)
- Enum validation (eks, gke, aks, etc.)
- Version format validation (regex)
- Range validation (if applicable)

**Snapshot Files:**
- YAML/JSON schema validation
- Required fields presence
- Type consistency
- Measurement structure validation

### Output Validation

**Recipes:**
- At least one measurement present
- All measurements have valid type
- All subtypes have data
- Context keys match data keys (if context=true)

**Bundles:**
- All required files generated
- Templates rendered successfully
- Checksums computed
- File permissions correct (scripts executable)

## See Also

- [Architecture: Data](../architecture/data.md) - Recipe data architecture
- [API Reference](api-reference.md) - API endpoint details
- [Automation](automation.md) - CI/CD integration patterns
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Developer guide
