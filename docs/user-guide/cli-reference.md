# CLI Reference

Complete reference for the `eidos` command-line interface.

## Overview

Eidos provides a three-step workflow for optimizing GPU infrastructure:

```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│   Snapshot   │─────▶│    Recipe    │─────▶│    Bundle    │
└──────────────┘      └──────────────┘      └──────────────┘
```

**Step 1**: Capture system configuration  
**Step 2**: Generate optimization recipes  
**Step 3**: Create deployment bundles  

## Global Flags

Available for all commands:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--debug` | `-d` | bool | false | Enable debug logging (text mode with full metadata) |
| `--log-json` | | bool | false | Enable JSON logging (structured output for machine parsing) |
| `--help` | `-h` | bool | false | Show help |
| `--version` | `-v` | bool | false | Show version |

### Logging Modes

Eidos supports three logging modes:

1. **CLI Mode (default)**: Minimal user-friendly output
   - Just message text without timestamps or metadata
   - Error messages display in red (ANSI color)
   - Example: `Snapshot captured successfully`

2. **Text Mode (`--debug`)**: Debug output with full metadata
   - Key=value format with time, level, source location
   - Example: `time=2025-01-06T10:30:00.123Z level=INFO module=eidos version=v1.0.0 msg="snapshot started"`

3. **JSON Mode (`--log-json`)**: Structured JSON for automation
   - Machine-readable format for log aggregation
   - Example: `{"time":"2025-01-06T10:30:00.123Z","level":"INFO","msg":"snapshot started"}`

**Examples:**
```shell
# Default: Clean CLI output
eidos snapshot

# Debug mode: Full metadata
eidos --debug snapshot

# JSON mode: Structured logs
eidos --log-json snapshot

# Combine with other flags
eidos --debug --output system.yaml snapshot
```

## Commands

### eidos snapshot

Capture comprehensive system configuration including OS, GPU, Kubernetes, and SystemD settings.

**Synopsis:**
```shell
eidos snapshot [flags]
```

**Flags:**
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | stdout | Output destination: file path, ConfigMap URI (cm://namespace/name), or stdout |
| `--format` | `-f` | string | yaml | Output format: json, yaml, table |
| `--kubeconfig` | `-k` | string | ~/.kube/config | Path to kubeconfig file (overrides KUBECONFIG env) |
| `--deploy-agent` | | bool | false | Deploy Kubernetes Job to capture snapshot on cluster nodes |
| `--namespace` | `-n` | string | gpu-operator | Kubernetes namespace for agent deployment |
| `--image` | | string | ghcr.io/nvidia/eidos:latest | Container image for agent Job |
| `--job-name` | | string | eidos | Name for the agent Job |
| `--service-account-name` | | string | eidos | ServiceAccount name for agent Job |
| `--node-selector` | | string[] | | Node selector for agent scheduling (key=value, repeatable) |
| `--toleration` | | string[] | all taints | Tolerations for agent scheduling (key=value:effect, repeatable). **Default: all taints tolerated** (uses `operator: Exists`). Only specify to restrict which taints are tolerated. |
| `--timeout` | | duration | 5m | Timeout for agent Job completion |
| `--cleanup` | | bool | false | Delete Job and RBAC on completion. **Default: keeps resources for debugging**. Cleanup runs regardless of success/failure when enabled. |

**Output Destinations:**
- **stdout**: Default when no `-o` flag specified
- **File**: Local file path (`/path/to/snapshot.yaml`)
- **ConfigMap**: Kubernetes ConfigMap URI (`cm://namespace/configmap-name`)

**What it captures:**
- **SystemD Services**: containerd, docker, kubelet configurations
- **OS Configuration**: grub, kmod, sysctl, release info
- **Kubernetes**: server version, images, ClusterPolicy
- **GPU**: driver version, CUDA, MIG settings, hardware info

**Examples:**

```shell
# Output to stdout (YAML)
eidos snapshot

# Save to file (JSON)
eidos snapshot --output system.json --format json

# Save to Kubernetes ConfigMap (requires cluster access)
eidos snapshot --output cm://gpu-operator/eidos-snapshot

# Debug mode
eidos --debug snapshot

# Table format (human-readable)
eidos snapshot --format table

# Agent deployment mode: Deploy Job to capture snapshot on cluster node
eidos snapshot --deploy-agent

# Agent deployment with custom kubeconfig
eidos snapshot --deploy-agent --kubeconfig ~/.kube/prod-cluster

# Agent deployment targeting specific nodes
eidos snapshot --deploy-agent \
  --namespace gpu-operator \
  --node-selector accelerator=nvidia-h100 \
  --node-selector zone=us-west1-a

# Agent deployment with tolerations for tainted nodes
# (By default all taints are tolerated - only needed to restrict tolerations)
eidos snapshot --deploy-agent \
  --toleration nvidia.com/gpu=present:NoSchedule

# Agent deployment: Full example with all options
eidos snapshot --deploy-agent \
  --kubeconfig ~/.kube/config \
  --namespace gpu-operator \
  --image ghcr.io/nvidia/eidos:v0.8.0 \
  --job-name snapshot-gpu-nodes \
  --service-account-name eidos \
  --node-selector accelerator=nvidia-h100 \
  --toleration nvidia.com/gpu:NoSchedule \
  --timeout 10m \
  --output cm://gpu-operator/eidos-snapshot \
  --cleanup
```

**Agent Deployment Mode:**

When `--deploy-agent` is specified, Eidos deploys a Kubernetes Job to capture the snapshot instead of running locally:

1. **Deploys RBAC**: ServiceAccount, Role, RoleBinding, ClusterRole, ClusterRoleBinding
2. **Creates Job**: Runs `eidos snapshot` as a container on the target node
3. **Waits for completion**: Monitors Job status with configurable timeout
4. **Retrieves snapshot**: Reads snapshot from ConfigMap after Job completes
5. **Writes output**: Saves snapshot to specified output destination
6. **Cleanup**: Optionally deletes Job and RBAC (default: keeps for debugging)

**Benefits of agent deployment:**
- Capture configuration from actual cluster nodes (not local machine)
- No need to run kubectl manually
- Programmatic deployment for automation/CI/CD
- Reusable RBAC resources across multiple runs

**Agent deployment requirements:**
- Kubernetes cluster access (via kubeconfig)
- Cluster admin permissions (for RBAC creation)
- GPU nodes with nvidia-smi (for GPU metrics)

```

**ConfigMap Output:**

When using ConfigMap URIs (`cm://namespace/name`), the snapshot is stored directly in Kubernetes:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: eidos-snapshot
  namespace: gpu-operator
  labels:
    app.kubernetes.io/name: eidos
    app.kubernetes.io/component: snapshot
data:
  snapshot.yaml: |
    # Full snapshot content
  format: yaml
  timestamp: "2025-12-31T10:30:00Z"
```

**Snapshot Structure:**
```yaml
apiVersion: cns.nvidia.com/v1alpha1
kind: Snapshot
metadata:
  created: "2025-12-31T10:30:00Z"
  hostname: gpu-node-1
measurements:
  - type: SystemD
    subtypes: [...]
  - type: OS
    subtypes: [...]
  - type: K8s
    subtypes: [...]
  - type: GPU
    subtypes: [...]
```

---

### eidos recipe

Generate optimized configuration recipes from query parameters or captured snapshots.

**Synopsis:**
```shell
eidos recipe [flags]
```

**Modes:**

#### Query Mode
Generate recipes using direct system parameters:

**Flags:**
| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--os` | | string | OS family: ubuntu, rhel, cos |
| `--osv` | | string | OS version: 24.04, 22.04 |
| `--kernel` | | string | Kernel version: 6.8, 5.15 |
| `--service` | | string | K8s service: eks, gke, aks, self-managed |
| `--k8s` | | string | Kubernetes version: v1.33, 1.32 |
| `--gpu` | | string | GPU type: h100, gb200, a100, l40 |
| `--intent` | | string | Workload intent: training, inference |
| `--context` | | bool | Include context metadata in response |
| `--output` | `-o` | string | Output file (default: stdout) |
| `--format` | `-f` | string | Format: json, yaml, table (default: json) |

**Examples:**
```shell
# Basic recipe for Ubuntu on EKS with H100
eidos recipe --os ubuntu --service eks --gpu h100

# Full specification with context
eidos recipe \
  --os ubuntu \
  --osv 24.04 \
  --kernel 6.8 \
  --service eks \
  --k8s 1.33 \
  --gpu gb200 \
  --intent training \
  --context \
  --format yaml

# Save to file
eidos recipe --os ubuntu --gpu h100 --output recipe.yaml
```

#### Snapshot Mode
Generate recipes from captured snapshots:

**Flags:**
| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--snapshot` | `-f` | string | Path/URI to snapshot (file path, URL, or cm://namespace/name) |
| `--intent` | `-i` | string | Workload intent: training, inference |
| `--output` | `-o` | string | Output destination (file, ConfigMap URI, or stdout) |
| `--format` | | string | Format: json, yaml, table (default: json) |
| `--context` | | bool | Include context metadata |
| `--kubeconfig` | `-k` | string | Path to kubeconfig file (for ConfigMap URIs, overrides KUBECONFIG env) |

**Snapshot Sources:**
- **File**: Local file path (`./snapshot.yaml`)
- **URL**: HTTP/HTTPS URL (`https://example.com/snapshot.yaml`)
- **ConfigMap**: Kubernetes ConfigMap URI (`cm://namespace/configmap-name`)

**Examples:**
```shell
# Generate recipe from local snapshot file
eidos recipe --snapshot system.yaml --intent training

# From ConfigMap (requires cluster access)
eidos recipe --snapshot cm://gpu-operator/eidos-snapshot --intent training

# From ConfigMap with custom kubeconfig
eidos recipe \
  --snapshot cm://gpu-operator/eidos-snapshot \
  --kubeconfig ~/.kube/prod-cluster \
  --intent training

# Output to ConfigMap
eidos recipe -f system.yaml -o cm://gpu-operator/eidos-recipe

# Chain snapshot → recipe with ConfigMaps
eidos snapshot -o cm://default/snapshot
eidos recipe -f cm://default/snapshot -o cm://default/recipe

# With custom output
eidos recipe -f system.yaml -i inference -o recipe.yaml --format yaml
```

**Output structure:**
```yaml
apiVersion: cns.nvidia.com/v1alpha1
kind: Recipe
metadata:
  created: "2025-12-31T10:30:00Z"
request:
  os: ubuntu
  gpu: h100
  service: eks
matchedRules:
  - "OS: ubuntu, GPU: h100, Service: eks"
measurements:
  - type: K8s
    subtypes: [...]
  - type: GPU
    subtypes: [...]
```

---

### eidos bundle

Generate deployment-ready bundles from recipes containing Helm values, manifests, scripts, and documentation.

**Synopsis:**
```shell
eidos bundle [flags]
```

**Flags:**
| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--recipe` | `-f` | string | Path to recipe file (required) |
| `--bundlers` | `-b` | string[] | Bundler types to execute (repeatable) |
| `--output` | `-o` | string | Output directory (default: current dir) |
| `--set` | | string[] | Override values in bundle files (repeatable) |

**Available bundlers:**
- `gpu-operator` - NVIDIA GPU Operator deployment bundle
- `network-operator` - NVIDIA Network Operator deployment bundle
- `cert-manager` - cert-manager deployment bundle
- `nvsentinel` - NVSentinel deployment bundle
- `skyhook` - Skyhook node optimization deployment bundle

**Behavior:**
- If `--bundlers` is omitted, **all registered bundlers** execute
- Bundlers run in **parallel** by default
- Each bundler creates a subdirectory in the output directory

**Value Overrides (`--set`):**

Override any value in the generated bundle files using dot notation:

```shell
--set bundler:path.to.field=value
```

**Format:** `bundler:path=value` where:
- `bundler` - Bundler name (e.g., `gpuoperator`, `networkoperator`, `certmanager`, `skyhook`, `nvsentinel`)
- `path` - Dot-separated path to the field
- `value` - New value to set

**Examples:**
```shell
# Generate all bundles
eidos bundle --recipe recipe.yaml --output ./bundles

# Generate specific bundler only
eidos bundle -f recipe.yaml -b gpu-operator -o ./deployment

# Multiple specific bundlers
eidos bundle -f recipe.yaml \
  -b gpu-operator \
  -b network-operator \
  -o ./bundles

# Override values in GPU Operator bundle
eidos bundle -f recipe.yaml -b gpu-operator \
  --set gpuoperator:gds.enabled=true \
  --set gpuoperator:driver.version=570.86.16 \
  -o ./bundles

# Override multiple bundlers
eidos bundle -f recipe.yaml \
  -b gpu-operator \
  -b network-operator \
  --set gpuoperator:mig.strategy=mixed \
  --set networkoperator:rdma.enabled=true \
  --set networkoperator:sriov.enabled=true \
  -o ./bundles

# Override cert-manager resources
eidos bundle -f recipe.yaml -b certmanager \
  --set certmanager:controller.resources.memory.limit=512Mi \
  --set certmanager:webhook.resources.cpu.limit=200m \
  -o ./bundles

# Override Skyhook manager resources
eidos bundle -f recipe.yaml -b skyhook \
  --set skyhook:manager.resources.cpu.limit=500m \
  --set skyhook:manager.resources.memory.limit=256Mi \
  -o ./bundles
```

**Bundle structure** (GPU Operator example):
```
gpu-operator/
├── values.yaml                    # Helm chart configuration
├── manifests/
│   └── clusterpolicy.yaml        # ClusterPolicy CR
├── scripts/
│   ├── install.sh                # Installation script
│   └── uninstall.sh              # Cleanup script
├── README.md                      # Deployment guide
└── checksums.txt                  # SHA256 checksums
```

**Deploying a bundle:**
```shell
# Navigate to bundle
cd bundles/gpu-operator

# Review configuration
cat values.yaml
cat README.md

# Verify integrity
sha256sum -c checksums.txt

# Deploy to cluster
chmod +x scripts/install.sh
./scripts/install.sh
```

---

## Complete Workflow Examples

### File-Based Workflow

```shell
# Step 1: Capture system configuration
eidos snapshot --output snapshot.yaml

# Step 2: Generate optimized recipe for training workloads
eidos recipe \
  --snapshot snapshot.yaml \
  --intent training \
  --output recipe.yaml

# Step 3: Create deployment bundle
eidos bundle \
  --recipe recipe.yaml \
  --bundlers gpu-operator \
  --output ./deployment

# Step 4: Deploy to cluster
cd deployment/gpu-operator
./scripts/install.sh

# Step 5: Verify deployment
kubectl get pods -n gpu-operator
kubectl logs -n gpu-operator -l app=nvidia-operator-validator
```

### ConfigMap-Based Workflow (Kubernetes-Native)

```shell
# Step 1: Agent captures snapshot to ConfigMap (using CLI deployment)
eidos snapshot --deploy-agent --output cm://gpu-operator/eidos-snapshot

# Alternative: Manual kubectl deployment
kubectl apply -f deployments/eidos-agent/1-deps.yaml
kubectl apply -f deployments/eidos-agent/2-job.yaml
kubectl wait --for=condition=complete job/eidos -n gpu-operator --timeout=5m

# Step 2: Generate recipe from ConfigMap
eidos recipe \
  --snapshot cm://gpu-operator/eidos-snapshot \
  --intent training \
  --output recipe.yaml

# Alternative: Write recipe to ConfigMap as well
eidos recipe \
  --snapshot cm://gpu-operator/eidos-snapshot \
  --intent training \
  --output cm://gpu-operator/eidos-recipe

# With custom kubeconfig (if not using default)
eidos recipe \
  --snapshot cm://gpu-operator/eidos-snapshot \
  --kubeconfig ~/.kube/prod-cluster \
  --intent training \
  --output recipe.yaml

# Step 3: Create bundle from recipe
eidos bundle \
  --recipe recipe.yaml \
  --bundlers gpu-operator \
  --output ./deployment

# Step 4: Deploy to cluster
cd deployment/gpu-operator
./scripts/install.sh

# Step 5: Verify deployment
kubectl get pods -n gpu-operator
kubectl logs -n gpu-operator -l app=nvidia-operator-validator
```

### E2E Testing

Validate the complete workflow:

```shell
# Test full workflow: agent → snapshot → recipe → bundle
./tools/e2e -s snapshot.yaml -r recipe.yaml -b ./bundles

# Test just snapshot capture to ConfigMap
./tools/e2e -s snapshot.yaml

# Test recipe and bundle generation from ConfigMap
./tools/e2e -r recipe.yaml -b ./bundles
```

## Shell Completion

Generate shell completion scripts:

```shell
# Bash
eidos completion bash

# Zsh
eidos completion zsh

# Fish
eidos completion fish

# PowerShell
eidos completion powershell
```

**Installation:**

**Bash:**
```shell
source <(eidos completion bash)
# Or add to ~/.bashrc for persistence
echo 'source <(eidos completion bash)' >> ~/.bashrc
```

**Zsh:**
```shell
source <(eidos completion zsh)
# Or add to ~/.zshrc
echo 'source <(eidos completion zsh)' >> ~/.zshrc
```

## Environment Variables

Eidos respects standard environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `KUBECONFIG` | Path to Kubernetes config file | `~/.kube/config` |
| `LOG_LEVEL` | Logging level: debug, info, warn, error | info |
| `NO_COLOR` | Disable colored output | false |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | File I/O error |
| 4 | Kubernetes connection error |
| 5 | Recipe generation error |

## Common Usage Patterns

### Quick Recipe Generation
```shell
eidos recipe --os ubuntu --gpu h100 | jq '.measurements[]'
```

### Save All Steps
```shell
eidos snapshot -o snapshot.yaml
eidos recipe -f snapshot.yaml -i training -o recipe.yaml
eidos bundle -f recipe.yaml -o ./bundles
```

### JSON Processing
```shell
# Extract GPU driver version from recipe
eidos recipe --os ubuntu --gpu h100 --format json | \
  jq -r '.measurements[] | select(.type=="GPU") | 
         .subtypes[] | select(.subtype=="driver") | 
         .data.version'
```

### Multiple Environments
```shell
# Generate recipes for different cloud providers
for service in eks gke aks; do
  eidos recipe --os ubuntu --service $service --gpu h100 \
    --output recipe-${service}.yaml
done
```

## Troubleshooting

### Snapshot Fails
```shell
# Check GPU drivers
nvidia-smi

# Check Kubernetes access
kubectl cluster-info

# Run with debug
eidos --debug snapshot
```

### Recipe Not Found
```shell
# Query parameters may not match any overlay
# Try broader query:
eidos recipe --os ubuntu --gpu h100
```

### Bundle Generation Fails
```shell
# Verify recipe file
cat recipe.yaml

# Check bundler is valid
eidos bundle --help  # Shows available bundlers

# Run with debug
eidos --debug bundle -f recipe.yaml -b gpu-operator
```

## See Also

- [Installation Guide](installation.md) - Install Eidos
- [Agent Deployment](agent-deployment.md) - Kubernetes agent setup
- [API Reference](../integration/api-reference.md) - Programmatic access
- [Architecture Docs](../architecture/README.md) - Internal architecture
