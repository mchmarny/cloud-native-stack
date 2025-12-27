# CLI Architecture

The `eidos` CLI provides command-line access to Cloud Native Stack configuration management capabilities.

## Overview

The CLI provides three main commands:
- `snapshot` - Capture system configuration
- `recipe` - Generate configuration recommendations based on environment parameters
- `recommend` - Analyze captured snapshots and provide tailored recommendations

## Architecture Diagram

```mermaid
flowchart TD
    A["eidos CLI<br/>cmd/eidos/main.go"] --> B["Root Command<br/>pkg/cli/root.go"]
    
    B --> B1["Version info (ldflags)<br/>Debug flag → Logging<br/>Shell completion"]
    
    B --> C["snapshot CMD<br/>pkg/cli/snapshot.go"]
    B --> D["recipe CMD<br/>pkg/cli/recipe.go"]
    B --> E["recommend CMD<br/>pkg/cli/recommned.go"]
    
    C --> F[Shared Packages]
    D --> F
    E --> F
    
    F --> F1["Collector Factory"]
    F --> F2["Recipe Builder"]
    F --> F3["Recommender Service"]
    F --> F4["Serializer<br/>(JSON/YAML/Table)"]
```

## Component Details

### Entry Point: `cmd/eidos/main.go`

Minimal entry point that delegates to the CLI package:

```go
package main

import "github.com/NVIDIA/cloud-native-stack/pkg/cli"

func main() {
    cli.Execute()
}
```

### Root Command: `pkg/cli/root.go`

**Responsibilities:**
- Command registration and routing
- Version information injection (via ldflags)
- Global flag handling (debug mode)
- Structured logging initialization

**Key Features:**
- Version info: `version`, `commit`, `date` (overridden at build time)
- Debug flag: `--debug` → Sets log level to debug
- Shell completion support
- Command listing for auto-completion

### Snapshot Command: `pkg/cli/snapshot.go`

Captures comprehensive system configuration snapshots.

#### Command Flow

```mermaid
flowchart TD
    A[User Invocation] --> B[Parse Flags<br/>format, output]
    B --> C[Create Collector Factory]
    C --> D[Initialize NodeSnapshotter]
    D --> E[Parallel Collection<br/>errgroup]
    E --> F[Aggregate Measurements]
    F --> G[Serialize Output]
    G --> H[Write to stdout/file]
    
    style E fill:#ffeb3b
```

#### Detailed Data Flow

```mermaid
flowchart TD
    A[Snapshot Command] --> B[collector.NewDefaultFactory]
    
    B --> B1["OSCollector<br/>(grub, kmod, sysctl)"]
    B --> B2["SystemDCollector<br/>(containerd, docker, kubelet)"]
    B --> B3["KubernetesCollector<br/>(server, images, policies)"]
    B --> B4["GPUCollector<br/>(nvidia-smi data)"]
    
    B1 & B2 & B3 & B4 --> C[NodeSnapshotter.Measure]
    
    C --> D["Parallel Collection<br/>(errgroup)"]
    
    D --> D1["Go Routine 1: Metadata<br/>• snapshot-version<br/>• source-node<br/>• timestamp"]
    D --> D2["Go Routine 2: Kubernetes<br/>• Server Version<br/>• Container Images<br/>• ClusterPolicies"]
    D --> D3["Go Routine 3: SystemD<br/>• containerd.service<br/>• docker.service<br/>• kubelet.service"]
    D --> D4["Go Routine 4: OS Config<br/>• GRUB parameters<br/>• Kernel modules<br/>• Sysctl parameters"]
    D --> D5["Go Routine 5: GPU<br/>• nvidia-smi properties<br/>• driver, CUDA, etc."]
    
    D1 & D2 & D3 & D4 & D5 --> E["All goroutines complete<br/>or first error returns"]
    
    E --> F["Snapshot Structure<br/>kind: Snapshot<br/>apiVersion: snapshot.dgxc.io/v1<br/>measurements: [k8s, systemd, os, gpu]"]
    
    F --> G[serializer.NewFileWriterOrStdout]
    
    G --> G1["Format: JSON/YAML/Table"]
    G --> G2["Output: stdout or file"]
    
    style D fill:#ffeb3b
    style D1 fill:#c8e6c9
    style D2 fill:#c8e6c9
    style D3 fill:#c8e6c9
    style D4 fill:#c8e6c9
    style D5 fill:#c8e6c9
```

#### Usage Examples

```bash
# Output to stdout in JSON format
eidos snapshot

# Save to file in YAML format
eidos snapshot --output system.yaml --format yaml

# Human-readable table format
eidos snapshot --format table
```

### Recipe Command: `pkg/cli/recipe.go`

Generates optimized configuration recipes based on environment parameters.

#### Command Flow

```mermaid
flowchart TD
    A[User Flags] --> B[Build Query from Flags]
    B --> C[Parse & Validate Versions]
    C --> D[recipe.BuildRecipe]
    D --> E["Load Recipe Store<br/>(embedded YAML)"]
    E --> F[Match Overlays]
    F --> G[Merge Measurements]
    G --> H[Serialize Output]
    H --> I[Write to stdout/file]
    
    style E fill:#fff9c4
    style F fill:#ffccbc
```

#### Detailed Data Flow

```mermaid
flowchart TD
    A[Recipe Command] --> B[buildQueryFromCmd]
    
    B --> B1["Parse CLI Flags:<br/>--os, --osv, --kernel<br/>--service, --k8s<br/>--gpu, --intent, --context"]
    B1 --> B2["Version Parsing:<br/>• ParseVersion for osv, kernel, k8s<br/>• Reject negative components<br/>• Support precision (1.2.3, 1.2, 1)"]
    
    B2 --> C[recipe.BuildRecipe]
    
    C --> C1["Step 1: Load Recipe Store<br/>(embedded YAML, cached)"]
    C1 --> C2["Step 2: Clone Base Measurements<br/>(deep copy: os, systemd, k8s, gpu)"]
    C2 --> C3["Step 3: Match Overlays<br/>• For each overlay: IsMatch(query)<br/>• Matching: empty=any, else equal<br/>• Version matching with precision"]
    C3 --> C4["Step 4: Merge Overlay Measurements<br/>• Index by measurement.Type<br/>• Merge subtypes by name<br/>• Overlay data takes precedence"]
    C4 --> C5["Step 5: Strip Context<br/>(if not requested)"]
    C5 --> C6["Recipe Structure:<br/>request,<br/>measurements"]
    
    C6 --> D["serializer.NewFileWriterOrStdout<br/>(JSON/YAML/Table)"]
    
    style C1 fill:#fff9c4
    style C3 fill:#ffccbc
    style C4 fill:#c5cae9
```

#### Recipe Matching Algorithm

The recipe matching uses a **rule-based query system** where overlays specify keys that must match the user's query:

```yaml
overlays:
  - key:
      service: eks          # Rule: must have service=eks
      os: ubuntu           # Rule: must have os=ubuntu
    types:
      - type: os
        subtypes:
          - subtype: grub
            data:
              BOOT_IMAGE: /boot/vmlinuz-6.8.0-1028-aws
```

**Matching Rules:**
1. **All** fields in the overlay key must be satisfied
2. Empty overlay field → matches anything (wildcard)
3. Empty query field → matches nothing (no match)
4. Version fields use semantic version equality with precision awareness

#### Usage Examples

```bash
# Basic recipe for Ubuntu with H100 GPU
eidos recipe --os ubuntu --gpu h100

# Full specification with all parameters
eidos recipe \
  --os ubuntu \
  --osv 24.04 \
  --kernel 6.8.0 \
  --service eks \
  --k8s v1.33.0 \
  --gpu gb200 \
  --intent training \
  --context \
  --format yaml \
  --output recipe.yaml

# Inference workload on GKE
eidos recipe --service gke --gpu a100 --intent inference
```

### Recommend Command: `pkg/cli/recommned.go`

Analyzes captured system snapshots and generates tailored configuration recommendations based on workload intent.

#### Command Flow

```mermaid
flowchart TD
    A[User Invocation] --> B[Parse Flags<br/>snapshot, intent, format, output]
    B --> C[Load Snapshot from File]
    C --> D[Create Recommender Service]
    D --> E[recommender.Recommend]
    E --> F[Analyze Snapshot]
    F --> G[Generate Intent-Based Recommendations]
    G --> H[Serialize Output]
    H --> I[Write to stdout/file]
    
    style E fill:#ffeb3b
```

#### Detailed Data Flow

```mermaid
flowchart TD
    A[Recommend Command] --> B[snapshotter.SnapshotFromFile]
    
    B --> B1["Load YAML Snapshot:<br/>• v0.7.0 format<br/>• 4 measurement types<br/>• Parse all subtypes"]
    
    B1 --> C[recommender.New]
    
    C --> C1["Initialize Recommender:<br/>• Version info<br/>• Recipe store access<br/>• Intent validation"]
    
    C1 --> D[recommender.Recommend]
    
    D --> D1["Analyze Snapshot:<br/>• OS configuration (4 subtypes)<br/>• K8s cluster state<br/>• GPU setup<br/>• SystemD services"]
    
    D1 --> D2["Intent Matching:<br/>• training: optimize for throughput<br/>• inference: optimize for latency<br/>• any: balanced recommendations"]
    
    D2 --> D3["Generate Recommendations:<br/>• Configuration changes<br/>• Optimization suggestions<br/>• Best practices"]
    
    D3 --> E["Recommendation Structure:<br/>snapshot, intent<br/>recommendations: []<br/>metadata"]
    
    E --> F["serializer.NewFileWriterOrStdout<br/>(JSON/YAML/Table)"]
    
    style D1 fill:#c8e6c9
    style D2 fill:#ffccbc
    style D3 fill:#c5cae9
```

#### Key Features

**1. Intent-Based Analysis**
- **training** – Optimizes for high throughput, batch processing, multi-GPU orchestration
- **inference** – Optimizes for low latency, single-request performance, efficient batching
- **any** – Provides general-purpose recommendations applicable to both workloads

**2. Snapshot Compatibility**
- Supports v0.7.0 snapshot format with 4 measurement types:
  - **OS** (4 subtypes: grub, sysctl, kmod, release)
  - **K8s** (server, image, policy subtypes)
  - **SystemD** (service configurations)
  - **GPU** (hardware and driver details)

**3. Vendor Version Support**
- Handles kernel versions with extras: `6.8.0-1028-aws`
- Handles K8s versions with vendor suffixes: `v1.33.5-eks-3025e55`
- Leverages version parser's Extras field for accurate version matching

#### Usage Examples

```bash
# Generate recommendations for training workloads
eidos recommend --snapshot system.yaml --intent training

# Analyze for inference optimization
eidos recommend \
  --snapshot cluster-snapshot.yaml \
  --intent inference \
  --format yaml \
  --output recommendations.yaml

# General recommendations with table format
eidos recommend -f system.yaml -i any --format table

# Abbreviated flags
eidos recommend -f system.yaml -i training -o recs.json
```

#### Recommendation Output Structure

```yaml
snapshot:
  version: v0.7.0
  node: ip-10-0-160-248
  timestamp: "2025-01-15T10:30:00Z"
intent: training
recommendations:
  - category: grub
    parameter: hugepages
    currentValue: "5128"
    recommendedValue: "8192"
    reason: "Increase hugepages for training workload memory efficiency"
    priority: high
  - category: sysctl
    parameter: net.core.somaxconn
    currentValue: "128"
    recommendedValue: "4096"
    reason: "Higher connection queue for distributed training"
    priority: medium
  - category: kubernetes
    parameter: gpu-operator.mig.strategy
    currentValue: "single"
    recommendedValue: "mixed"
    reason: "Enable MIG for multi-tenant training workloads"
    priority: low
metadata:
  recommendationVersion: v0.7.0
```

#### Error Handling

- **Invalid Intent**: Returns error with supported intent types
- **Missing Snapshot File**: File not found error with path
- **Invalid Snapshot Format**: Parse error with details
- **Unknown Output Format**: Error with supported formats list

## Shared Infrastructure

### Collector Factory Pattern

The CLI uses the **Factory Pattern** for collector instantiation, enabling:
- **Testability**: Inject mock collectors for unit tests
- **Flexibility**: Easy to add new collector types
- **Encapsulation**: Hide collector creation complexity

```go
type Factory interface {
    CreateSystemDCollector() Collector
    CreateOSCollector() Collector
    CreateKubernetesCollector() Collector
    CreateGPUCollector() Collector
}
```

### Serializer Abstraction

Output formatting is abstracted through the `serializer.Serializer` interface:

```go
type Serializer interface {
    Serialize(data interface{}) error
}
```

Implementations:
- **JSON**: `encoding/json` with 2-space indent
- **YAML**: `gopkg.in/yaml.v3` 
- **Table**: `text/tabwriter` for columnar display

### Measurement Data Model

All collected data uses a unified `measurement.Measurement` structure:

```go
type Measurement struct {
    Type     Type      // os, k8s, systemd, gpu
    Subtypes []Subtype // Named collections of readings
}

type Subtype struct {
    Name    string                // grub, kmod, sysctl, server, image, etc.
    Data    map[string]Reading    // Key-value readings
    Context map[string]string     // Human-readable descriptions
}

type Reading struct {
    Value interface{}  // Actual value (int, string, bool, float64)
}
```

## Error Handling

### CLI Error Strategy

1. **Flag Validation**: User-friendly error messages for invalid flags
2. **Version Parsing**: Specific error types (ErrNegativeComponent, etc.)
3. **Collector Failures**: Log errors, continue with partial data where possible
4. **Serialization Errors**: Fatal - abort and report
5. **Exit Codes**: Non-zero exit code on any failure

### Example Error Messages

```bash
# Invalid version format
$ eidos recipe --osv -1.0
Error: error parsing recipe input parameter: os version cannot contain negative numbers: -1.0

# Unknown output format
$ eidos snapshot --format xml
Error: unknown output format: "xml"

# Missing required parameters
$ eidos recipe
# Still succeeds - generates base recipe with no overlays
```

## Performance Characteristics

### Snapshot Command

- **Parallel Collection**: All collectors run concurrently via `errgroup`
- **Typical Duration**: 100-500ms depending on cluster size
- **Memory Usage**: ~10-50MB for typical workloads
- **Scalability**: O(n) with number of pods/nodes for K8s collector

### Recipe Command

- **Store Loading**: Once per process (cached via `sync.Once`)
- **Typical Duration**: <10ms after initial load
- **Memory Usage**: ~5-10MB (embedded YAML + parsed structure)
- **Scalability**: O(m) with number of overlays (typically <100)

## Build Configuration

### Version Injection via ldflags

Build-time version information injection:

```makefile
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X github.com/NVIDIA/cloud-native-stack/pkg/cli.version=$(VERSION)
LDFLAGS += -X github.com/NVIDIA/cloud-native-stack/pkg/cli.commit=$(COMMIT)
LDFLAGS += -X github.com/NVIDIA/cloud-native-stack/pkg/cli.date=$(DATE)

go build -ldflags="$(LDFLAGS)" -o bin/eidos ./cmd/eidos
```

## Testing Strategy

### Unit Tests
- Flag parsing and validation
- Version parsing and error handling
- Query building from command flags
- Serializer format selection

### Integration Tests
- Mock collectors for deterministic output
- Full command execution with fake factory
- Output format validation

### Example Test Structure

```go
func TestSnapshotCommand(t *testing.T) {
    // Create mock factory
    mockFactory := &MockFactory{
        k8s:     mockK8sCollector,
        systemd: mockSystemDCollector,
        os:      mockOSCollector,
        gpu:     mockGPUCollector,
    }
    
    // Execute snapshot with mock
    snapshotter := NodeSnapshotter{
        Factory: mockFactory,
        Serializer: &bytes.Buffer{},
    }
    
    err := snapshotter.Measure(ctx)
    assert.NoError(t, err)
}
```

## Dependencies

### External Libraries
- `github.com/urfave/cli/v3` - CLI framework
- `golang.org/x/sync/errgroup` - Concurrent error handling
- `gopkg.in/yaml.v3` - YAML parsing
- `log/slog` - Structured logging

### Internal Packages
- `pkg/collector` - System data collection
- `pkg/measurement` - Data model
- `pkg/recipe` - Recipe building
- `pkg/recipe/version` - Semantic versioning
- `pkg/serializer` - Output formatting
- `pkg/logging` - Logging configuration
- `pkg/snapshotter` - Snapshot orchestration

## Future Enhancements

### Short-Term (< 3 months)

1. **Caching Layer**  
   **Rationale**: Reduce latency for repeated `eidos snapshot` calls in scripts  
   **Implementation**: `sync.Map` with TTL-based eviction using `time.AfterFunc`  
   **Trade-off**: Stale data risk vs 5-10x performance improvement  
   **Reference**: [sync.Map](https://pkg.go.dev/sync#Map)

2. **Differential Snapshots**  
   **Use Case**: CI/CD pipelines detecting configuration drift  
   **Implementation**: `github.com/google/go-cmp/cmp` for deep comparison  
   **Output**: JSON Patch (RFC 6902) format for machine consumption  
   **CLI**: `eidos diff baseline.yaml current.yaml --format patch`

3. **Measurement Filtering**  
   **Use Case**: Extract only GPU data without K8s overhead  
   **CLI**: `eidos snapshot --filter gpu,os --exclude k8s`  
   **Implementation**: Post-collection filtering before serialization  
   **Performance**: Saves 60-70% execution time when K8s excluded

4. **Batch Mode**  
   **Use Case**: Fleet-wide configuration auditing (100s of nodes)  
   **Implementation**: Worker pool with `errgroup.SetLimit()`  
   **CLI**: `eidos snapshot --nodes nodes.txt --workers 10 --output results/`  
   **Reference**: [errgroup Limits](https://pkg.go.dev/golang.org/x/sync/errgroup#Group.SetLimit)

### Mid-Term (3-6 months)

5. **Plugin System**  
   **Rationale**: Custom collectors without forking codebase  
   **Interface**: `type Collector interface { Collect(context.Context) (Measurement, error) }`  
   **Options**: Go plugins (unstable across versions) or WASM (safe, portable)  
   **Security**: Sandboxed execution with restricted syscalls  
   **Reference**: [WebAssembly System Interface](https://wasi.dev/)

6. **Configuration Files**  
   **Use Case**: Avoid repeating --os, --gpu flags  
   **Format**: YAML following XDG Base Directory spec  
   **Location**: `~/.config/eidos/config.yaml` (Linux/macOS), `%APPDATA%\eidos\config.yaml` (Windows)  
   **Example**:
   ```yaml
   defaults:
     os: ubuntu
     gpu: h100
     format: yaml
   server:
     url: https://recipe-api.example.com
   ```

7. **Watch Mode**  
   **Implementation**: Hybrid of `fsnotify` + periodic polling  
   **CLI**: `eidos snapshot --watch --interval 30s --on-change ./alert.sh`  
   **Output**: Stream of JSON diffs to stdout  
   **Use Case**: Real-time monitoring with alerting

8. **Schema Validation**  
   **Use Case**: Ensure snapshots conform to API version spec  
   **Implementation**: Embed JSON Schema in binary with `go:embed`  
   **Library**: `github.com/santhosh-tekuri/jsonschema/v5` (fastest Go validator)  
   **CLI**: `eidos validate --schema v1 snapshot.json`

### Long-Term (6-12 months)

9. **gRPC Mode**  
   **Rationale**: Better streaming, 3-5x smaller payloads than JSON  
   **Implementation**: Bi-directional streaming with protobuf  
   **Trade-off**: Added complexity (proto definitions) vs performance gains  
   **Reference**: [gRPC Go](https://grpc.io/docs/languages/go/)

10. **Distributed Tracing**  
    **Use Case**: Debug performance issues across collectors  
    **Implementation**: OpenTelemetry SDK with span per collector  
    **Exporter**: OTLP to Jaeger/Tempo  
    **CLI**: `eidos snapshot --trace --trace-endpoint localhost:4317`  
    **Reference**: [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)

11. **Policy Enforcement**  
    **Use Case**: Block non-compliant configs in CI/CD  
    **Implementation**: Embed OPA (`github.com/open-policy-agent/opa`)  
    **CLI**: `eidos validate --policy policy.rego snapshot.yaml`  
    **Exit Code**: 0 = pass, 1 = policy violations  
    **Reference**: [OPA Go Integration](https://www.openpolicyagent.org/docs/latest/integration/)

12. **Cloud Storage Integration**  
    **Use Case**: Centralized storage for fleet management  
    **CLI**: `eidos snapshot --upload s3://bucket/snapshots/$(hostname).yaml`  
    **Implementation**: AWS SDK v2 with resumable uploads  
    **Authentication**: IAM roles, service accounts, credential chain  
    **Reference**: [AWS SDK for Go V2](https://aws.github.io/aws-sdk-go-v2/)

## Production Deployment Patterns

### Pattern 1: CI/CD Integration

**Use Case**: Automated configuration validation in build pipelines

**GitLab CI Example**:
```yaml
validate_gpu_config:
  stage: test
  image: ghcr.io/nvidia/eidos:latest
  script:
    - eidos snapshot --format json > snapshot.json
    # Validate against known-good baseline
    - diff -u expected_snapshot.json snapshot.json
    # Or use OPA policy (future enhancement)
    # - eidos validate --policy policies/gpu_baseline.rego snapshot.json
  only:
    - merge_requests
  artifacts:
    when: on_failure
    paths:
      - snapshot.json
```

**GitHub Actions Example**:
```yaml
name: Validate GPU Configuration
on:
  pull_request:
    paths:
      - 'ansible/**'
      - 'terraform/**'

jobs:
  validate:
    runs-on: [self-hosted, gpu]
    steps:
      - uses: actions/checkout@v4
      
      - name: Install eidos
        run: |
          curl -sfL https://raw.githubusercontent.com/.../installer | bash -s --
          echo "$HOME/.local/bin" >> $GITHUB_PATH
      
      - name: Capture snapshot
        run: eidos snapshot --format yaml --output snapshot.yaml
      
      - name: Generate recipe
        run: eidos recipe --os ubuntu --gpu h100 > recipe.yaml
      
      - name: Compare configurations
        run: |
          yq eval '.measurements[] | select(.type=="GPU")' snapshot.yaml > actual_gpu.yaml
          yq eval '.measurements[] | select(.type=="GPU")' recipe.yaml > expected_gpu.yaml
          diff -u expected_gpu.yaml actual_gpu.yaml || \
            (echo "::error::GPU configuration drift detected" && exit 1)
      
      - name: Upload artifact
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: configuration-drift
          path: |
            snapshot.yaml
            recipe.yaml
```

**Jenkins Pipeline**:
```groovy
pipeline {
    agent { label 'gpu-node' }
    
    stages {
        stage('Snapshot') {
            steps {
                sh 'eidos snapshot --format json > snapshot.json'
            }
        }
        
        stage('Validate') {
            steps {
                script {
                    def snapshot = readJSON file: 'snapshot.json'
                    def gpuDriver = snapshot.measurements
                        .find { it.type == 'GPU' }
                        .subtypes.find { it.subtype == 'smi' }
                        .data.'driver-version'
                    
                    if (gpuDriver != '570.158.01') {
                        error("Incorrect GPU driver: ${gpuDriver}")
                    }
                }
            }
        }
    }
    
    post {
        always {
            archiveArtifacts artifacts: 'snapshot.json', fingerprint: true
        }
    }
}
```

### Pattern 2: Scheduled Auditing

**Use Case**: Nightly configuration drift detection across fleet

**Kubernetes CronJob**:
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: eidos-audit
  namespace: monitoring
spec:
  schedule: "0 2 * * *"  # 2 AM daily
  concurrencyPolicy: Forbid  # Prevent overlapping runs
  successfulJobsHistoryLimit: 7
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        metadata:
          labels:
            app: eidos-audit
        spec:
          serviceAccountName: eidos
          nodeSelector:
            node-role.kubernetes.io/gpu: "true"
          tolerations:
          - key: nvidia.com/gpu
            operator: Exists
            effect: NoSchedule
          containers:
          - name: eidos
            image: ghcr.io/nvidia/eidos:v0.6.4
            command:
              - /bin/sh
              - -c
              - |
                set -e
                TIMESTAMP=$(date +%Y%m%d-%H%M%S)
                HOSTNAME=$(hostname)
                
                # Capture snapshot
                eidos snapshot --format yaml > /tmp/snapshot.yaml
                
                # Store as ConfigMap with retention
                kubectl create configmap \
                  "eidos-snapshot-${HOSTNAME}-${TIMESTAMP}" \
                  --from-file=snapshot=/tmp/snapshot.yaml \
                  --dry-run=client -o yaml | \
                kubectl apply -f -
                
                # Cleanup old snapshots (keep last 30 days)
                kubectl get configmaps -l eidos-snapshot=true \
                  --sort-by=.metadata.creationTimestamp | \
                head -n -30 | \
                xargs -r kubectl delete configmap
            resources:
              limits:
                memory: 256Mi
              requests:
                cpu: 100m
                memory: 128Mi
          restartPolicy: OnFailure
```

**Systemd Timer (Bare Metal)**:
```ini
# /etc/systemd/system/eidos-audit.service
[Unit]
Description=Eidos Configuration Audit
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/eidos snapshot --format json --output /var/log/eidos/snapshot-%Y%m%d.json
User=eidos
Group=eidos

# Hardening
PrivateTmp=true
NoNewPrivileges=true
ReadOnlyPaths=/usr /etc
ReadWritePaths=/var/log/eidos

[Install]
WantedBy=multi-user.target

# /etc/systemd/system/eidos-audit.timer
[Unit]
Description=Eidos Audit Timer

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

Enable with:
```bash
sudo systemctl enable --now eidos-audit.timer
sudo systemctl list-timers eidos-audit.timer
```

### Pattern 3: Fleet Management

**Use Case**: Collect snapshots from 100s of GPU nodes in parallel

**Ansible Playbook**:
```yaml
---
- name: Collect Eidos Snapshots from GPU Fleet
  hosts: gpu_nodes
  gather_facts: yes
  serial: 10  # Process 10 nodes at a time
  tasks:
    - name: Ensure eidos is installed
      stat:
        path: /usr/local/bin/eidos
      register: eidos_binary
      failed_when: not eidos_binary.stat.exists
    
    - name: Collect snapshot
      shell: eidos snapshot --format json
      register: snapshot
      changed_when: false
      failed_when: snapshot.rc != 0
    
    - name: Upload to S3
      aws_s3:
        bucket: fleet-snapshots
        object: "{{ inventory_hostname }}/{{ ansible_date_time.iso8601 }}.json"
        content: "{{ snapshot.stdout }}"
        mode: put
      delegate_to: localhost
      run_once: false
    
    - name: Validate against baseline
      shell: |
        echo '{{ snapshot.stdout }}' | \
        jq '.measurements[] | select(.type=="GPU") | .subtypes[] | 
            select(.subtype=="smi") | .data."driver-version"'
      register: driver_version
      failed_when: driver_version.stdout != '"570.158.01"'
      changed_when: false

- name: Generate Fleet Report
  hosts: localhost
  tasks:
    - name: Download all snapshots
      aws_s3:
        bucket: fleet-snapshots
        mode: list
      register: s3_objects
    
    - name: Aggregate results
      script: scripts/aggregate_snapshots.py
      args:
        snapshots: "{{ s3_objects.s3_keys }}"
```

**Terraform Provisioning**:
```hcl
resource "null_resource" "eidos_snapshot" {
  count = length(var.gpu_instance_ids)
  
  provisioner "remote-exec" {
    inline = [
      "eidos snapshot --format json > /tmp/snapshot.json",
      "aws s3 cp /tmp/snapshot.json s3://fleet-snapshots/${self.id}/"
    ]
    
    connection {
      type        = "ssh"
      host        = element(var.gpu_instance_ips, count.index)
      user        = "ubuntu"
      private_key = file("~/.ssh/id_rsa")
    }
  }
  
  triggers = {
    instance_id = element(var.gpu_instance_ids, count.index)
    timestamp   = timestamp()
  }
}

data "aws_s3_objects" "snapshots" {
  bucket     = "fleet-snapshots"
  depends_on = [null_resource.eidos_snapshot]
}

output "snapshot_count" {
  value = length(data.aws_s3_objects.snapshots.keys)
}
```

### Pattern 4: Real-Time Monitoring

**Use Case**: Continuous configuration monitoring with Prometheus alerting

**Prometheus Exporter** (future enhancement):
```go
package main

import (
    "context"
    "net/http"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

var (
    gpuDriverVersion = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "eidos_gpu_driver_version",
            Help: "NVIDIA driver version (encoded as float)",
        },
        []string{"node", "gpu_model"},
    )
    
    k8sVersion = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "eidos_k8s_version",
            Help: "Kubernetes version (encoded)",
        },
        []string{"node"},
    )
)

func init() {
    prometheus.MustRegister(gpuDriverVersion, k8sVersion)
}

func collectMetrics() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        snapshot, err := snapshotter.Measure(ctx)
        cancel()
        
        if err != nil {
            log.Printf("Snapshot failed: %v", err)
            continue
        }
        
        // Extract and export GPU driver version
        for _, m := range snapshot.Measurements {
            if m.Type == "GPU" {
                for _, st := range m.Subtypes {
                    if st.Subtype == "smi" {
                        version := st.Data["driver-version"]
                        encoded := encodeVersion(version)
                        gpuModel := st.Data["gpu-name"]
                        gpuDriverVersion.WithLabelValues(hostname, gpuModel).Set(encoded)
                    }
                }
            }
        }
    }
}

func main() {
    go collectMetrics()
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":9090", nil)
}
```

**Prometheus Alerting Rules**:
```yaml
groups:
- name: eidos_configuration
  interval: 60s
  rules:
  - alert: GPUDriverVersionMismatch
    expr: |
      count(count by (eidos_gpu_driver_version) (eidos_gpu_driver_version)) > 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Multiple GPU driver versions detected in cluster"
      description: "{{ $value }} different driver versions found"
  
  - alert: KubernetesVersionSkew
    expr: |
      abs(eidos_k8s_version - scalar(avg(eidos_k8s_version))) > 0.01
    for: 10m
    labels:
      severity: critical
    annotations:
      summary: "Kubernetes version skew detected on {{ $labels.node }}"
      description: "Node version differs from cluster average"
```

## Advanced Usage Patterns

### Snapshot Diffing with jq

```bash
#!/bin/bash
# Capture baseline before changes
eidos snapshot --format json > baseline.json

# Apply configuration changes (Ansible, Terraform, etc.)
# ...

# Capture new snapshot
eidos snapshot --format json > current.json

# Diff specific sections
echo "=== GPU Configuration Changes ==="
diff -u \
  <(jq -S '.measurements[] | select(.type=="GPU")' baseline.json) \
  <(jq -S '.measurements[] | select(.type=="GPU")' current.json)

echo "=== Kernel Parameter Changes ==="
diff -u \
  <(jq -S '.measurements[] | select(.type=="os") | .subtypes[] | 
           select(.subtype=="sysctl")' baseline.json) \
  <(jq -S '.measurements[] | select(.type=="os") | .subtypes[] | 
           select(.subtype=="sysctl")' current.json)

# Count total changes
changes=$(diff <(jq -S . baseline.json) <(jq -S . current.json) | grep -c '^[<>]')
echo "Total configuration changes: $changes"
```

### Recipe Generation Pipeline

```bash
#!/bin/bash
# Generate recipes for all supported configurations

set -euo pipefail

OUTPUT_DIR="recipes"
mkdir -p "$OUTPUT_DIR"

# GPU types from NVIDIA product line
GPU_TYPES=("h100" "gb200" "a100" "l40" "l4")

# Kubernetes services
K8S_SERVICES=("eks" "gke" "aks" "self-managed")

# OS distributions
OS_TYPES=("ubuntu" "rhel" "cos")

total=0
for gpu in "${GPU_TYPES[@]}"; do
  for service in "${K8S_SERVICES[@]}"; do
    for os in "${OS_TYPES[@]}"; do
      output="${OUTPUT_DIR}/${os}-${service}-${gpu}.yaml"
      
      # Generate recipe
      if eidos recipe --os "$os" --service "$service" --gpu "$gpu" \
           --format yaml > "$output" 2>/dev/null; then
        echo "✓ Generated $output"
        ((total++))
      else
        echo "✗ Failed: $os $service $gpu"
      fi
    done
  done
done

echo "Generated $total recipes"

# Validate all recipes
echo "Validating recipes..."
find "$OUTPUT_DIR" -name '*.yaml' -exec yq eval '.' {} \; > /dev/null
echo "All recipes valid"

# Create index
cat > "$OUTPUT_DIR/README.md" <<EOF
# Configuration Recipes

Generated on $(date -Iseconds)

Total recipes: $total

## Available Configurations

| OS | Service | GPU | File |
|----|---------|-----|------|
EOF

find "$OUTPUT_DIR" -name '*.yaml' -type f | sort | while read -r file; do
  base=$(basename "$file" .yaml)
  IFS='-' read -ra parts <<< "$base"
  echo "| ${parts[0]} | ${parts[1]} | ${parts[2]} | $file |" >> "$OUTPUT_DIR/README.md"
done
```

### Automated Remediation

```bash
#!/bin/bash
# Apply recommended configuration from recipe
# WARNING: Modifies system configuration - use with caution

set -euo pipefail

# Capture current state
current=$(eidos snapshot --format json)

# Generate recommended recipe
recipe=$(eidos recipe --os ubuntu --gpu h100 --format json)

# Extract recommended GRUB parameters
recommended_grub=$(echo "$recipe" | jq -r '
  .measurements[] | 
  select(.type=="os") | 
  .subtypes[] | 
  select(.subtype=="grub") | 
  .data | 
  to_entries[] | 
  "\(.key)=\(.value)"' | tr '\n' ' ')

# Extract current GRUB parameters
current_grub=$(echo "$current" | jq -r '
  .measurements[] | 
  select(.type=="os") | 
  .subtypes[] | 
  select(.subtype=="grub") | 
  .data | 
  to_entries[] | 
  "\(.key)=\(.value)"' | tr '\n' ' ')

# Show diff
echo "Current GRUB parameters:"
echo "$current_grub"
echo ""
echo "Recommended GRUB parameters:"
echo "$recommended_grub"
echo ""

# Prompt for confirmation
read -p "Apply changes? (yes/no): " confirm
if [[ "$confirm" != "yes" ]]; then
  echo "Aborted"
  exit 0
fi

# Apply GRUB changes (requires root)
sudo grubby --update-kernel=ALL --args="$recommended_grub"
echo "GRUB configuration updated. Reboot required."

# Apply sysctl changes
echo "$recipe" | jq -r '
  .measurements[] | 
  select(.type=="os") | 
  .subtypes[] | 
  select(.subtype=="sysctl") | 
  .data | 
  to_entries[] | 
  "\(.key) = \(.value)"' | \
sudo tee /etc/sysctl.d/99-eidos-recommended.conf

sudo sysctl --system
echo "Sysctl parameters applied"

# Log changes
echo "$(date -Iseconds): Applied Eidos recommendations" | \
sudo tee -a /var/log/eidos-remediation.log
```

## Troubleshooting Guide

### Issue: "nvidia-smi not found"

**Symptoms**: GPU measurements empty, error in logs  
**Root Cause**: NVIDIA driver not installed or not in PATH  

**Diagnosis**:
```bash
# Check if nvidia-smi exists
which nvidia-smi
# Expected: /usr/bin/nvidia-smi

# Verify driver installation
nvidia-smi --version
# Expected: NVIDIA-SMI 570.158.01

# Check kernel modules
lsmod | grep nvidia
# Expected: nvidia, nvidia_uvm, nvidia_modeset

# Verify device nodes
ls -l /dev/nvidia*
# Expected: /dev/nvidia0, /dev/nvidiactl, /dev/nvidia-uvm
```

**Resolution**:
```bash
# Ubuntu: Install NVIDIA driver
sudo apt-get update
sudo apt-get install -y nvidia-driver-570

# RHEL: Install from CUDA repo
sudo dnf config-manager --add-repo \
  https://developer.download.nvidia.com/compute/cuda/repos/rhel8/x86_64/cuda-rhel8.repo
sudo dnf install -y nvidia-driver:570

# Verify installation
sudo nvidia-smi

# If PATH issue, add to shell profile
echo 'export PATH="/usr/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Issue: "Kubernetes API server unreachable"

**Symptoms**: K8s measurements empty, "connection refused" error  
**Root Cause**: Not running in cluster, or kubeconfig missing/invalid  

**Diagnosis**:
```bash
# Verify cluster connectivity
kubectl cluster-info
# Expected: Kubernetes control plane is running at https://...

# Check kubeconfig
echo $KUBECONFIG
cat ~/.kube/config

# Test API access
kubectl get nodes
# Expected: List of nodes

# Check service account (in-cluster)
ls -l /var/run/secrets/kubernetes.io/serviceaccount/
# Expected: token, ca.crt, namespace
```

**Resolution**:
```bash
# Option 1: Set KUBECONFIG explicitly
export KUBECONFIG=~/.kube/config
eidos snapshot

# Option 2: Copy admin kubeconfig
sudo cp /etc/kubernetes/admin.conf ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config

# Option 3: Use service account token (in-cluster)
kubectl create serviceaccount eidos
kubectl create clusterrolebinding eidos --clusterrole=view --serviceaccount=default:eidos

# Option 4: Debug with kubectl proxy
kubectl proxy &
export KUBERNETES_SERVICE_HOST=localhost
export KUBERNETES_SERVICE_PORT=8001
eidos snapshot
```

### Issue: "Snapshot too slow (> 5s)"

**Symptoms**: Long execution time, timeouts in CI/CD  
**Root Cause**: Large cluster (1000s of pods), slow API server, many GPUs  

**Diagnosis**:
```bash
# Enable debug logging to identify slow collectors
eidos --debug snapshot 2>&1 | grep -E 'collector|duration'
# Expected output shows timing per collector:
# time="..." level=debug msg="k8s collector finished" duration=3.2s
# time="..." level=debug msg="gpu collector finished" duration=0.8s

# Check cluster size
kubectl get pods --all-namespaces --no-headers | wc -l
# Large: > 1000 pods

# Check GPU count
nvidia-smi --list-gpus | wc -l
# Many: > 8 GPUs

# Profile execution
time eidos snapshot > /dev/null
```

**Resolution**:
```bash
# Option 1: Filter to specific collectors (future enhancement)
eidos snapshot --filter gpu,os  # Skip K8s (saves 60-70% time)

# Option 2: Increase timeout (future enhancement)
eidos snapshot --timeout 30s

# Option 3: Use caching for repeated calls
eidos snapshot > /tmp/snapshot.json
# Reuse /tmp/snapshot.json for subsequent analysis

# Option 4: Optimize K8s collector
# Reduce API calls by using label selectors (code change):
# clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
#     LabelSelector: "app=gpu-operator",
# })

# Option 5: Run in parallel with errgroup limit
# Already implemented in code, but can tune:
# g.SetLimit(runtime.NumCPU())  // Current: 2
```

### Issue: "Out of memory during snapshot"

**Symptoms**: Process killed, OOMKilled in K8s, segfault  
**Root Cause**: Large measurement data (10k+ pods, many images)  

**Diagnosis**:
```bash
# Check memory usage during snapshot
/usr/bin/time -v eidos snapshot > /dev/null 2>&1
# Look for "Maximum resident set size"

# Monitor memory in real-time
# Terminal 1:
watch -n 1 'ps aux | grep eidos'
# Terminal 2:
eidos snapshot

# In Kubernetes, check OOMKilled events
kubectl get events --field-selector reason=OOMKilling
```

**Resolution**:
```bash
# Option 1: Use streaming serialization (already implemented)
# Data never fully materialized in memory
eidos snapshot --format json > snapshot.json

# Option 2: Increase memory limit in Kubernetes
kubectl set resources deployment eidos-agent \
  --limits=memory=1Gi \
  --requests=memory=512Mi

# Option 3: Filter measurements (future enhancement)
eidos snapshot --filter gpu,os  # Exclude large K8s data

# Option 4: Optimize code to reduce allocations
# Use object pooling for repeated structs:
var measurementPool = sync.Pool{
    New: func() interface{} {
        return &measurement.Measurement{}
    },
}

# Option 5: Process in batches (code change needed)
# For K8s pods, paginate API calls:
pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
    Limit: 100,
    Continue: continueToken,
})
```

## Performance Tuning

### CPU Profiling

```bash
# Build with profiling enabled
go build -o eidos cmd/eidos/main.go

# Capture CPU profile
./eidos snapshot --cpuprofile=cpu.prof

# Analyze profile
go tool pprof cpu.prof
(pprof) top10
# Shows top 10 functions by CPU time

(pprof) list collectContainerImages
# Shows line-by-line CPU usage in specific function

(pprof) web
# Opens interactive graph in browser (requires graphviz)

# Example output interpretation:
# If collectContainerImages is > 50% CPU:
# - Optimize pod iteration
# - Reduce string allocations
# - Cache image parsing results
```

### Memory Profiling

```bash
# Capture memory profile
./eidos snapshot --memprofile=mem.prof

# Analyze allocations
go tool pprof -alloc_space mem.prof
(pprof) top10
# Shows top 10 functions by allocations

(pprof) list BuildRecipe
# Check for unnecessary allocations

# Example fixes:
# Before: strings.Split() allocates slice
# After: strings.Index() + slicing avoids allocation

# Before: fmt.Sprintf("%s:%s", name, tag)
# After: var b strings.Builder; b.WriteString(name); b.WriteString(":");
```

### Benchmarking

```bash
# Benchmark snapshot performance (10 iterations)
for i in {1..10}; do
  time eidos snapshot --format json > /dev/null
done 2>&1 | grep real | awk '{print $2}' | \
sed 's/0m//' | sed 's/s//' | \
awk '{sum+=$1; count++} END {printf "Average: %.3fs\n", sum/count}'

# Compare formats
echo "JSON:"
time eidos snapshot --format json > /dev/null
echo "YAML:"
time eidos snapshot --format yaml > /dev/null
echo "Table:"
time eidos snapshot --format table > /dev/null

# Expected results:
# JSON:  ~50ms  (fastest, minimal processing)
# YAML:  ~80ms  (indentation overhead)
# Table: ~100ms (string formatting, column alignment)

# Benchmark with different cluster sizes
for pods in 10 100 1000 5000; do
  # Scale test deployment
  kubectl scale deployment test-app --replicas=$pods
  kubectl wait --for=condition=ready pod -l app=test-app --timeout=5m
  
  echo "Cluster with $pods pods:"
  time eidos snapshot --format json > /dev/null
done
```

### Optimization Recommendations

1. **Reduce String Allocations**  
   **Current**: `fmt.Sprintf("%s:%s", name, tag)` allocates  
   **Optimized**: Use `strings.Builder` for concatenation  
   **Savings**: 20-30% fewer allocations in image collector

2. **Preallocate Slices**  
   **Current**: `measurements := []Measurement{}`  
   **Optimized**: `measurements := make([]Measurement, 0, expectedSize)`  
   **Benefit**: Avoids slice growth reallocations  
   **When**: Size predictable (e.g., GPU count known)

3. **Pool Large Objects**  
   **Use Case**: Measurement structs allocated repeatedly  
   **Implementation**:
   ```go
   var measurementPool = sync.Pool{
       New: func() interface{} {
           return &measurement.Measurement{}
       },
   }
   
   m := measurementPool.Get().(*measurement.Measurement)
   defer measurementPool.Put(m)
   ```
   **Reference**: [sync.Pool](https://pkg.go.dev/sync#Pool)

4. **Avoid Reflection**  
   **Current**: `encoding/json` uses reflection  
   **Optimized**: Code-generated marshaling with `easyjson`  
   **Benefit**: 2-3x faster JSON serialization  
   **Trade-off**: Build complexity vs performance  
   **Reference**: [easyjson](https://github.com/mailru/easyjson)

5. **Batch API Operations**  
   **Current**: Multiple API calls per collector  
   **Optimized**: Aggregate calls where possible  
   **Example**: List all pods once, filter in memory  
   **Benefit**: Reduces API server load, faster execution

6. **Concurrent Collectors**  
   **Current**: `errgroup` with limit  
   **Tuning**: Adjust limit based on collector type  
   ```go
   g.SetLimit(runtime.NumCPU())  // CPU-bound collectors
   g.SetLimit(runtime.NumCPU() * 2)  // I/O-bound collectors
   ```
   **Reference**: [errgroup SetLimit](https://pkg.go.dev/golang.org/x/sync/errgroup#Group.SetLimit)

## Security Best Practices

### Running as Non-Root

**CLI**:
```bash
# CLI runs as current user (no special privileges needed)
eidos snapshot  # Works as non-root

# Verify no setuid/setgid
ls -l $(which eidos)
# Expected: -rwxr-xr-x (not -rwsr-xr-x)

# Verify no capabilities
getcap $(which eidos)
# Expected: (no output)
```

**Kubernetes Job**:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: eidos
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: eidos
        image: ghcr.io/nvidia/eidos:latest
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
        volumeMounts:
        - name: tmp
          mountPath: /tmp
      volumes:
      - name: tmp
        emptyDir: {}
```

### Secrets Management

```bash
# Never log sensitive data
# eidos already filters passwords/tokens from output

# Verify no secrets in snapshot
eidos snapshot --format json | \
  jq '.measurements[].subtypes[].data | 
      keys | map(select(test("(?i)(password|token|key|secret)"))) | 
      unique'
# Expected: []

# Use environment variables for API credentials (future feature)
export EIDOS_API_TOKEN=$(vault kv get -field=token secret/eidos)
eidos recipe --os ubuntu --gpu h100

# Or use Kubernetes secrets
kubectl create secret generic eidos-api-creds \
  --from-literal=token=$(vault kv get -field=token secret/eidos)

# Mount in pod:
volumeMounts:
- name: api-creds
  mountPath: /var/run/secrets/eidos
  readOnly: true
volumes:
- name: api-creds
  secret:
    secretName: eidos-api-creds
```

### Input Validation

**CLI validates all inputs before processing**:

```bash
# Invalid OS type
eidos recipe --os invalid_os
# Error: invalid os type "invalid_os", must be one of: ubuntu, rhel, cos

# Invalid version format
eidos recipe --osv -1.0
# Error: invalid version "-1.0": negative version components not allowed

# Invalid GPU type
eidos recipe --gpu h100@latest
# Error: invalid gpu type "h100@latest": special characters not allowed

# Invalid format
eidos snapshot --format xml
# Error: invalid format "xml", must be one of: json, yaml, table

# Path traversal prevention
eidos snapshot --output ../../etc/passwd
# Error: output path escapes current directory

# Verify validation in code:
# pkg/cli/recipe.go:
if !isValidOS(os) {
    return fmt.Errorf("invalid os type %q", os)
}
```

### Network Security

```bash
# Verify TLS for API calls (future feature)
eidos recipe --os ubuntu --gpu h100 --debug 2>&1 | grep -i tls
# Expected: "Using TLS 1.3"

# Certificate pinning (future enhancement)
export EIDOS_API_CERT_FINGERPRINT="sha256:abc123..."
eidos recipe --os ubuntu --gpu h100

# Use corporate proxy with authentication
export HTTPS_PROXY=https://proxy.corp.com:8080
export EIDOS_PROXY_CA_CERT=/etc/ssl/certs/corp-ca.pem
eidos recipe --os ubuntu --gpu h100
```

## References

### Official Documentation
- [urfave/cli Framework](https://cli.urfave.org/) - CLI framework used by eidos  
- [errgroup Patterns](https://pkg.go.dev/golang.org/x/sync/errgroup) - Concurrent error handling  
- [YAML v3 Library](https://pkg.go.dev/gopkg.in/yaml.v3) - YAML parsing and serialization  
- [Structured Logging (slog)](https://pkg.go.dev/log/slog) - Standard library logging  
- [Context Package](https://pkg.go.dev/context) - Cancellation and deadlines

### Kubernetes Integration
- [client-go Documentation](https://github.com/kubernetes/client-go) - Official K8s client  
- [Dynamic Client](https://pkg.go.dev/k8s.io/client-go/dynamic) - Unstructured resource access  
- [CronJob Best Practices](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) - Scheduled job patterns  
- [RBAC Authorization](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) - Permission model

### NVIDIA Tools
- [NVIDIA SMI](https://developer.nvidia.com/nvidia-system-management-interface) - GPU management  
- [NVML Library](https://developer.nvidia.com/nvml) - Programmatic GPU access  
- [CUDA Toolkit](https://developer.nvidia.com/cuda-toolkit) - GPU computing platform  
- [GPU Operator](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/) - K8s GPU automation

### Best Practices
- [Semantic Versioning](https://semver.org/) - Version comparison algorithm  
- [The Twelve-Factor App](https://12factor.net/) - Cloud-native application patterns  
- [Release Engineering Best Practices](https://sre.google/workbook/release-engineering/) - Google SRE  
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) - Idiomatic Go

### Security
- [OWASP Secure Coding Practices](https://owasp.org/www-project-secure-coding-practices-quick-reference-guide/)  
- [Kubernetes Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)  
- [NIST 800-190: Container Security](https://csrc.nist.gov/publications/detail/sp/800-190/final)  
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks) - Security configuration baselines
