# Cloud Native Stack

Cloud Native Stack (CNS) provides tooling and comprehensive documentation to help you deploy, validate, and operate optimized AI workloads in your GPU-accelerated Kubernetes clusters:

- **Documentation** – Installation guides, playbooks, optimizations, and troubleshooting for GPU infrastructure
- **Eidos CLI** – Command-line tool for system snapshots and CNS recipe generation
- **Eidos Agent** – Kubernetes job for automated cluster configuration and optimization using NCS recipes

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

Generate optimized configuration recipes based on your environment:

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

**Available flags:**
- `--os` – Operating system (ubuntu, cos, etc.)
- `--osv` – OS version (e.g., 24.04)
- `--kernel` – Kernel version (supports vendor suffixes like `6.8.0-1028-aws`)
- `--service` – Kubernetes service (eks, gke, aks, self-managed)
- `--k8s` – Kubernetes version (supports vendor formats like `v1.33.5-eks-3025e55`)
- `--gpu` – GPU type (h100, gb200, etc.)
- `--intent` – Workload intent (training, inference)
- `--context` – Include metadata in response
- `--format` – Output format (json, yaml, table)
- `--output` – Save to file (default: stdout)

#### Generate Recommendations from Snapshot

Analyze a captured snapshot and generate configuration recommendations:

```shell
# Generate recommendations for training workloads
eidos recommend --snapshot system.yaml --intent training

# Output recommendations to file in YAML format
eidos recommend \
  --snapshot system.yaml \
  --intent inference \
  --format yaml \
  --output recommendations.yaml
```

**Available flags:**
- `--snapshot`, `-f` – Path to snapshot file (required)
- `--intent`, `-i` – Workload intent: `training`, `inference`, or `any` (required)
- `--output`, `-o` – Output file path (default: stdout)
- `--format` – Output format (json, yaml, table)

The `recommend` command analyzes your system snapshot and provides tailored configuration recommendations based on the specified workload intent.

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
