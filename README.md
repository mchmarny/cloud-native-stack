# Cloud Native Stack

Cloud Native Stack (CNS) provides tooling and comprehensive documentation to help you deploy, validate, and operate optimized AI workloads in your GPU-accelerated Kubernetes clusters:

- **Documentation** – Installation guides, playbooks, optimizations, and troubleshooting for GPU infrastructure
- **CLI** – Command-line tool for system snapshots and CNS recipe generation
- **Agent** – Kubernetes job for automated cluster configuration and optimization using NCS recipes
- **API** - REST API for optimized configuration generation based on input parameters

## Quick Start

### Install the `eidos` CLI

You can find the latest release [here](https://github.com/mchmarny/cloud-native-stack/releases/latest). You can install the latest version manually from the [releases section of the repo](https://github.com/mchmarny/cloud-native-stack/releases/latest) or using provided script:

```shell
curl -sfL https://raw.githubusercontent.com/mchmarny/cloud-native-stack/refs/heads/main/install | bash -s --
```

Verify installation:

```shell
eidos --version
```

### CLI Commands

#### Snapshot System Configuration

Capture a comprehensive snapshot of your system including CPU/GPU settings, kernel parameters, systemd services, and Kubernetes configuration:

```shell
# Output to stdout (YAML)
eidos snapshot

# Save to file (JSON format)
eidos snapshot --output system.yaml --format json
```

> NOTE: Eidos is fully self-contained CLI. Id does not connect to any external resources and never emits any of the captured configuration outside of the cluster. 

**The snapshot includes:**

- **SystemD Services** – Complete service configurations including:
  - containerd, docker, kubelet service states and settings
  - Active state, startup configuration, and resource limits
  - Service dependencies and execution parameters

- **OS Configuration** – 4 subtypes capturing system-level settings:
  - `grub` – Boot parameters and kernel arguments (hugepages, numa_balancing, security settings)
  - `sysctl` – All kernel parameters from `/proc/sys` (networking, filesystem, memory tuning)
  - `kmod` – Loaded kernel modules with their configurations
  - `release` – OS identification from `/etc/os-release` (ID, VERSION_ID, PRETTY_NAME, VERSION_CODENAME)

- **Kubernetes** – Complete cluster configuration with 3 subtypes:
  - `server` – Kubernetes version with vendor-specific format support (e.g., `v1.33.5-eks-3025e55`), Go version, platform
  - `image` – All deployed container images with full registry paths and versions
  - `policy` – Complete GPU Operator ClusterPolicy configuration (100+ settings including driver, device plugin, MIG, CDI)

- **GPU** – Comprehensive hardware and driver information:
  - GPU model, architecture, and compute capability
  - Driver version, CUDA version, and firmware details
  - GPU-specific settings (MIG mode, persistence mode, addressing mode)

**Snapshot** output formats**: JSON, YAML, table

#### Generate Configuration Recipe

Generate optimized configuration recipes based on your environment. The recipe command supports two modes:

**Query Mode** - Generate recipes using direct system parameters:

```shell
# Basic recipe for Ubuntu on EKS with H100 GPUs
eidos recipe --os ubuntu --service eks --gpu h100

# Full specification with context optimized for training use-case
eidos recipe \
  --os ubuntu \
  --osv 24.04 \
  --kernel 5.15.0 \
  --service eks \
  --k8s v1.28.0 \
  --gpu gb200 \
  --intent training \
  --context \
  --format yaml
```

**Snapshot Mode** - Generate recipes from captured system snapshots:

```shell
# Generate recipe from snapshot for training workloads
eidos recipe --snapshot system.yaml --intent training

# Output recipe to file in YAML format
eidos recipe \
  --snapshot system.yaml \
  --intent inference \
  --format yaml \
  --output recipe.yaml
```

**Available flags:**
- `--os` – Operating system (ubuntu, cos, etc.)
- `--osv` – OS version (e.g., 24.04)
- `--kernel` – Kernel version (supports vendor suffixes like `6.8.0-1028-aws`)
- `--service` – Kubernetes service (eks, gke, aks, self-managed)
- `--k8s` – Kubernetes version (supports vendor formats like `v1.33.5-eks-3025e55`)
- `--gpu` – GPU type (h100, gb200, etc.)
- `--intent` – Workload intent (training, inference)
- `--snapshot`, `-f` – Path to snapshot file (enables snapshot mode)
- `--context` – Include metadata in response
- `--format` – Output format (json, yaml, table)
- `--output` – Save to file (default: stdout)

The recipe command analyzes your environment (from query parameters or snapshot) and generates optimized configuration recommendations based on the specified workload intent.

#### Generate Deployment Bundle

Generate deployment-ready bundles from recipes containing Helm values, Kubernetes manifests, installation scripts, and documentation:

```shell
# Generate GPU Operator bundle from recipe
eidos bundle --recipe recipe.yaml --output ./bundles

# Generate from snapshot with workload intent
eidos bundle --snapshot system.yaml --intent training --output ./bundles

# Specify bundler types explicitly
eidos bundle --recipe recipe.yaml --bundler gpu-operator --output ./bundles
```

**Bundle contents** for GPU Operator:
```
gpu-operator/
├── values.yaml                    # Helm chart configuration
├── manifests/
│   └── clusterpolicy.yaml        # ClusterPolicy custom resource
├── scripts/
│   ├── install.sh                # Installation automation
│   └── uninstall.sh              # Cleanup automation
├── README.md                      # Deployment instructions
└── checksums.txt                  # SHA256 verification
```

**Available flags:**
- `--recipe` – Path to recipe file
- `--snapshot` – Path to snapshot file (generates recipe first)
- `--intent` – Workload intent when using snapshot (training, inference)
- `--bundler` – Bundler type(s) to use (gpu-operator, network-operator)
- `--output` – Output directory (default: ./bundles)
- `--namespace` – Kubernetes namespace for deployment
- `--dry-run` – Validate without creating files

**Installation using generated bundle:**
```bash
cd bundles/gpu-operator
chmod +x scripts/install.sh
./scripts/install.sh
```

### Deploy the Eidos Agent

Eidos can also be deployed as an agent into your Kubernetes cluster as a Job to automatically capture cluster configuration snapshots. This is useful for auditing, troubleshooting, and configuration management.

> NOTE: Eidos is fully self-contained CLI. Id does not connect to any external resources and never emits any of the captured configuration outside of the cluster. 

#### Prerequisites

- Kubernetes cluster with GPU nodes
- `kubectl` configured with cluster access
- GPU Operator installed (agent runs in `gpu-operator` namespace)

#### Installation

1. Apply the required RBAC permissions and service account:

```shell
kubectl apply -f https://raw.githubusercontent.com/mchmarny/cloud-native-stack/main/deployments/eidos-agent/1-deps.yaml
```

2. Deploy the agent job:

```shell
kubectl apply -f https://raw.githubusercontent.com/mchmarny/cloud-native-stack/main/deployments/eidos-agent/2-job.yaml
```

#### Customization

Before deploying, you may need to customize the Job manifest:

**Node Selection** – Update `nodeSelector` to target specific GPU nodes:
```yaml
nodeSelector:
  nodeGroup: your-gpu-node-group
```

**Tolerations** – Adjust tolerations for your node taints:
```yaml
tolerations:
  - key: nvidia.com/gpu
    operator: Exists
    effect: NoSchedule
```

**Image Version** – Use a specific version by replacing `latest` with any of the [CNS release versions](https://github.com/mchmarny/cloud-native-stack/releases):
```yaml
image: ghcr.io/mchmarny/eidos-api-server:latest
```

#### View Agent Output

Check job status:
```shell
kubectl get jobs -n gpu-operator
```

View snapshot output:
```shell
kubectl logs -n gpu-operator job/eidos
```

The agent outputs a YAML snapshot of the cluster node configuration to stdout.

### Query the API

The Eidos API provides a REST interface for generating configuration recipes without requiring CLI installation. It's designed for integration with automation tools, CI/CD pipelines, and web applications.

> NOTE: The API does not store any query data. All recipe generation happens in real-time using embedded configuration templates.

#### API Endpoint

**Base URL:** https://cns.dgxc.io

**Recipe Generation:** `GET /v1/recipe`

#### Query Parameters

Generate optimized configuration recipes by specifying environment parameters:

**Available query string parameters:**
- `os` – Operating system family: `ubuntu`, `rhel`, `cos`, `all` (default: `all`)
- `osv` – OS version (e.g., `24.04`, `22.04`) (default: `all`)
- `kernel` – Kernel version (supports vendor suffixes like `6.8.0-1028-aws`) (default: `all`)
- `service` – Kubernetes service: `eks`, `gke`, `aks`, `oke`, `self-managed`, `all` (default: `all`)
- `k8s` – Kubernetes version (supports vendor formats like `v1.33.5-eks-3025e55`) (default: `all`)
- `gpu` – GPU type: `h100`, `gb200`, `a100`, `l40`, `all` (default: `all`)
- `intent` – Workload intent: `training`, `inference`, `any` (default: `any`)
- `context` – Include metadata in response: `true`, `false` (default: `false`)

#### Usage Examples

**Basic query** – Ubuntu on EKS with GB200 GPUs:
```shell
curl "https://cns.dgxc.io/v1/recipe?os=ubuntu&gpu=gb200&service=eks"
```

**Full specification** – Optimized for training workloads with context:
```shell
curl "https://cns.dgxc.io/v1/recipe?os=ubuntu&osv=24.04&kernel=6.8&service=eks&k8s=1.33&gpu=h100&intent=training&context=true"
```

**Using with jq** – Extract specific settings:
```shell
curl -s "https://cns.dgxc.io/v1/recipe?gpu=gb200&intent=inference" | jq '.measurements[] | select(.type=="GPU")'
```

**Save to file:**
```shell
curl "https://cns.dgxc.io/v1/recipe?os=ubuntu&gpu=h100&service=gke" -o recipe.json
```

#### Response Format

The API returns JSON responses containing configuration recipes:

```json
{
  "apiVersion": "recipe.dgxc.io/v1",
  "kind": "Recipe",
  "metadata": {
    "created": "2025-12-27T10:30:00Z",
    "recipe-version": "v1.0.0"
  },
  "request": {
    "os": "ubuntu",
    "gpu": "h100",
    "service": "eks",
    "intent": "training"
  },
  "matchedRules": [
    "os:ubuntu",
    "gpu:h100",
    "intent:training"
  ],
  "measurements": [
    {
      "type": "K8s",
      "subtypes": [
        {
          "subtype": "cluster",
          "data": {
            "gpu-operator-version": "25.3.1",
            "enable-mig": "false"
          }
        }
      ]
    },
    {
      "type": "GPU",
      "subtypes": [
        {
          "subtype": "driver",
          "data": {
            "version": "570.158.01",
            "cuda-version": "12.7"
          }
        }
      ]
    }
  ]
}
```

#### Response Headers

The API includes helpful response headers:

- `Content-Type: application/json` – Response format
- `X-Request-Id` – Unique request identifier for tracing
- `Cache-Control: public, max-age=300` – Response caching guidance
- `X-RateLimit-Limit` – Maximum requests per window
- `X-RateLimit-Remaining` – Requests remaining in current window
- `X-RateLimit-Reset` – Unix timestamp when window resets

#### Error Handling

The API returns standard HTTP status codes with detailed error messages:

**400 Bad Request** – Invalid parameter values:
```json
{
  "code": "INVALID_PARAMETER",
  "message": "invalid gpu type: must be one of h100, gb200, a100, l40, all",
  "details": {
    "parameter": "gpu",
    "provided": "invalid-gpu"
  },
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-27T10:30:00Z",
  "retryable": false
}
```

**404 Not Found** – No matching configuration:
```json
{
  "code": "NO_MATCHING_RULE",
  "message": "no configuration recipe found for the specified parameters",
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-27T10:30:00Z",
  "retryable": false
}
```

**429 Too Many Requests** – Rate limit exceeded:
```json
{
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "rate limit exceeded, please retry after indicated time",
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-27T10:30:00Z",
  "retryable": true
}
```

Response includes `Retry-After` header indicating when to retry.

**500 Internal Server Error** – Server-side error:
```json
{
  "code": "INTERNAL_ERROR",
  "message": "an internal error occurred processing your request",
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-27T10:30:00Z",
  "retryable": true
}
```

#### Rate Limiting

The API implements rate limiting to ensure fair usage:

- **Rate:** 100 requests per second per IP
- **Burst:** 200 requests
- **Headers:** All responses include rate limit status in headers

When rate limited, wait for the time specified in the `Retry-After` header before retrying.

#### Health Checks

Monitor API availability using health endpoints:

**Liveness probe:**
```shell
curl https://cns.dgxc.io/health
```

**Readiness probe:**
```shell
curl https://cns.dgxc.io/ready
```

Both return `200 OK` when healthy with a simple status JSON.

#### Integration Examples

**Python with requests:**
```python
import requests

params = {
    'os': 'ubuntu',
    'gpu': 'h100',
    'service': 'eks',
    'intent': 'training'
}

response = requests.get('https://cns.dgxc.io/v1/recipe', params=params)
if response.status_code == 200:
    recipe = response.json()
    print(f"Matched {len(recipe['matchedRules'])} configuration rules")
else:
    print(f"Error: {response.json()['message']}")
```

**Go with net/http:**
```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

func main() {
    url := "https://cns.dgxc.io/v1/recipe?os=ubuntu&gpu=gb200&service=eks"
    
    resp, err := http.Get(url)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    
    var recipe map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&recipe); err != nil {
        panic(err)
    }
    
    fmt.Printf("Recipe: %+v\n", recipe)
}
```

**Shell script automation:**
```bash
#!/bin/bash

# Generate recipe and extract GPU driver version
DRIVER_VERSION=$(curl -s "https://cns.dgxc.io/v1/recipe?gpu=h100&intent=training" \
  | jq -r '.measurements[] | select(.type=="GPU") | .subtypes[] | select(.subtype=="driver") | .data.version')

echo "Recommended driver version: $DRIVER_VERSION"
```

## Security and Verification

### Container Image Attestations

All container images published from tagged releases include build provenance attestations, signed using GitHub's OIDC identity. These attestations provide verifiable evidence of how the images were built.

#### Verify Image Attestations

You can verify the authenticity and provenance of container images using the GitHub CLI:

```shell
# Verify the eidos CLI image
gh attestation verify oci://ghcr.io/mchmarny/eidos:TAG --owner mchmarny

# Verify the eidos-api-server image  
gh attestation verify oci://ghcr.io/mchmarny/eidos-api-server:TAG --owner mchmarny
```

Replace `TAG` with the specific version you want to verify (e.g., `v0.6.4` or `latest`).

The attestations include:
- **Build provenance** – Complete record of the build environment and process
- **Source repository** – Link to the exact source code commit
- **Build workflow** – GitHub Actions workflow that produced the image
- **Signed metadata** – Cryptographically signed using Sigstore

For more information on attestations, see the [GitHub Artifact Attestations documentation](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations).

## Documentation

Comprehensive deployment and operations guides:

- **[Installation Guides](docs/install-guides)** – Step-by-step setup for various platforms
- **[Playbooks](docs/playbooks)** – Ansible automation for CNS deployment
- **[Optimizations](docs/optimizations)** – Hardware-specific performance tuning
- **[Troubleshooting](docs/troubleshooting)** – Common issues and solutions
- **[Full Documentation](docs/README.md)** – Complete reference

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup and workflow
- Code quality standards
- Pull request process
- Building and testing locally

## Support

- **Releases**: [GitHub Releases](https://github.com/NVIDIA/cloud-native-stack/releases)
- **Issues**: [GitHub Issues](https://github.com/NVIDIA/cloud-native-stack/issues)
- **Questions**: Open a discussion or issue on GitHub
