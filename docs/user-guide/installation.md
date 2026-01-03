# Installation Guide

This guide covers installing and setting up Cloud Native Stack (Eidos) for your environment.

## Prerequisites

- **Operating System**: Linux, macOS, or Windows (via WSL)
- **Kubernetes Cluster** (optional): For agent deployment or bundle generation testing
- **GPU Hardware** (optional): NVIDIA GPUs for full system snapshot capabilities
- **kubectl** (optional): For Kubernetes agent deployment

## Install Eidos CLI

### Option 1: Automated Installation (Recommended)

Install the latest version using the installation script:

```shell
curl -sfL https://raw.githubusercontent.com/mchmarny/cloud-native-stack/refs/heads/main/install | bash -s --
```

This script:
- Detects your OS and architecture automatically
- Downloads the appropriate binary from GitHub releases
- Installs to `/usr/local/bin/eidos` (requires sudo)
- Verifies the installation

> **Supply Chain Security** – CNS includes SLSA Build Level 3 compliance with signed SBOMs and verifiable attestations (see [SECURITY](../SECURITY.md#supply-chain-security))

### Option 2: Manual Installation

1. **Download the latest release**

Visit the [releases page](https://github.com/mchmarny/cloud-native-stack/releases/latest) and download the appropriate binary for your platform:

- **macOS ARM64** (M1/M2/M3): `eidos_darwin_arm64.tar.gz`
- **macOS Intel**: `eidos_darwin_amd64.tar.gz`
- **Linux ARM64**: `eidos_linux_arm64.tar.gz`
- **Linux x86_64**: `eidos_linux_amd64.tar.gz`

2. **Extract and install**

```shell
# Example for Linux x86_64
tar -xzf eidos_linux_amd64.tar.gz
sudo mv eidos /usr/local/bin/
sudo chmod +x /usr/local/bin/eidos
```

3. **Verify installation**

```shell
eidos --version
```

### Option 3: Build from Source

**Requirements:**
- Go 1.21 or higher
- `make`

```shell
# Clone repository
git clone https://github.com/NVIDIA/cloud-native-stack.git
cd cloud-native-stack

# Build
make build

# Binary location
./dist/eidos_<platform>/eidos
```

## Verify Installation

Check that Eidos is correctly installed:

```shell
# Check version
eidos --version

# View available commands
eidos --help

# Test snapshot (requires GPU)
eidos snapshot --format json | jq '.measurements | length'
```

Expected output shows version information and available commands.

## Post-Installation

### Shell Completion (Optional)

Enable shell auto-completion for better CLI experience:

**Bash:**
```shell
# Add to ~/.bashrc
source <(eidos completion bash)
```

**Zsh:**
```shell
# Add to ~/.zshrc
source <(eidos completion zsh)
```

**Fish:**
```shell
# Add to ~/.config/fish/config.fish
eidos completion fish | source
```

### Kubernetes Access (Optional)

If you plan to use the agent or generate bundles for Kubernetes, ensure kubectl is configured:

```shell
# Test Kubernetes connectivity
kubectl cluster-info

# Verify GPU nodes (if applicable)
kubectl get nodes -l nvidia.com/gpu.present=true
```

## Container Images

Eidos is also available as container images for integration into automated pipelines:

### CLI Image
```shell
docker pull ghcr.io/nvidia/eidos:latest
docker run ghcr.io/nvidia/eidos:latest --version
```

### API Server Image
```shell
docker pull ghcr.io/nvidia/eidos-api-server:latest
docker run -p 8080:8080 ghcr.io/nvidia/eidos-api-server:latest
```

**Production API Server**: The API server is deployed at https://cns.dgxc.io with auto-scaling and SLSA Build Level 3 attestations.

## E2E Testing

Validate the complete workflow with the e2e testing script:

```shell
# Clone repository
git clone https://github.com/NVIDIA/cloud-native-stack.git
cd cloud-native-stack

# Test complete workflow: agent → snapshot → recipe → bundle
./tools/e2e -s examples/snapshots/h100.yaml \
           -r examples/recipes/h100-eks-ubuntu-training.yaml \
           -b examples/bundles/h100-eks-ubuntu-training

# Test just snapshot capture
./tools/e2e -s snapshot.yaml

# Test recipe and bundle generation
./tools/e2e -r recipe.yaml -b ./bundles
```

The e2e script:
- Deploys agent Job with RBAC
- Waits for snapshot to be written to ConfigMap
- Generates recipe and bundle from ConfigMap
- Validates each step completes successfully
- Preserves resources on failure for debugging

## Next Steps

- **Users**: See [CLI Reference](cli-reference.md) for command usage
- **Kubernetes Users**: See [Agent Deployment](agent-deployment.md) to deploy the snapshot agent
- **Integrators**: See [API Reference](../integration/api-reference.md) for programmatic access

## Troubleshooting

### Command Not Found

If `eidos` is not found after installation:

```shell
# Check if binary is in PATH
echo $PATH | grep -q /usr/local/bin && echo "OK" || echo "Add /usr/local/bin to PATH"

# Add to PATH (bash)
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Permission Denied

```shell
# Make binary executable
sudo chmod +x /usr/local/bin/eidos
```

### GPU Detection Issues

Snapshot GPU measurements require `nvidia-smi` in PATH:

```shell
# Verify NVIDIA drivers
nvidia-smi

# If missing, install NVIDIA drivers for your platform
```

### Kubernetes Connection Issues

```shell
# Check kubeconfig
kubectl config current-context

# Verify cluster access
kubectl get nodes
```

## Uninstall

```shell
# Remove binary
sudo rm /usr/local/bin/eidos

# Remove shell completion (if configured)
# Remove the source line from your shell RC file
```

## Getting Help

- **Documentation**: [User Guide](../user-guide/)
- **Issues**: [GitHub Issues](https://github.com/NVIDIA/cloud-native-stack/issues)
- **API Server**: See [Integration Guide](../integration/api-reference.md)
