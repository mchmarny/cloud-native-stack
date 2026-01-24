# Installation Guide

This guide describes how to install Cloud Native Stack (CNS) CLI tool (`cnsctl`) on Linux, macOS, or Windows.

**What is Cloud Native Stack**: CNS generates validated configurations for GPU-accelerated Kubernetes deployments. See [README](../../README.md) for project overview.

## Prerequisites

- **Operating System**: Linux, macOS, or Windows (via WSL)
- **Kubernetes Cluster** (optional): For agent deployment or bundle generation testing
- **GPU Hardware** (optional): NVIDIA GPUs for full system snapshot capabilities
- **kubectl** (optional): For Kubernetes agent deployment

## Install cnsctl CLI

### Option 1: Automated Installation (Recommended)

Install the latest version using the installation script:

```shell
curl -sfL https://raw.githubusercontent.com/nvidia/cloud-native-stack/refs/heads/main/install | bash -s --
```

This script:
- Detects your OS and architecture automatically
- Downloads the appropriate binary from GitHub releases
- Installs to `/usr/local/bin/cnsctl` (requires sudo)
- Verifies the installation

> **Supply Chain Security**: CNS includes SLSA Build Level 3 compliance with signed SBOMs and verifiable attestations. See [SECURITY](../SECURITY.md#supply-chain-security) for verification instructions.

### Option 2: Manual Installation

1. **Download the latest release**

Visit the [releases page](https://github.com/nvidia/cloud-native-stack/releases/latest) and download the appropriate binary for your platform:

- **macOS ARM64** (M1/M2/M3): `cnsctl_v0.22.0_darwin_arm64`
- **macOS Intel**: `cnsctl_v0.22.0_darwin_amd64`
- **Linux ARM64**: `cnsctl_v0.22.0_linux_arm64`
- **Linux x86_64**: `cnsctl_v0.22.0_linux_amd64`

1. **Extract and install**

```shell
# Example for Linux x86_64
tar -xzf cns_linux_amd64.tar.gz
sudo mv cnsctl /usr/local/bin/
sudo chmod +x /usr/local/bin/cnsctl
```

3. **Verify installation**

```shell
cnsctl --version
```

### Option 3: Build from Source

**Requirements:**
- Go 1.25 or higher

```shell
go install github.com/NVIDIA/cloud-native-stack/cmd/cnsctl@latest
```

## Verify Installation

Check that cnsctl is correctly installed:

```shell
# Check version
cnsctl --version

# View available commands
cnsctl --help

# Test snapshot (requires GPU)
cnsctl snapshot --format json | jq '.measurements | length'
```

Expected output shows version information and available commands.

## Post-Installation

### Shell Completion (Optional)

Enable shell auto-completion for command and flag names:

**Bash:**
```shell
# Add to ~/.bashrc
source <(cnsctl completion bash)
```

**Zsh:**
```shell
# Add to ~/.zshrc
source <(cnsctl completion zsh)
```

**Fish:**
```shell
# Add to ~/.config/fish/config.fish
cnsctl completion fish | source
```

## Container Images

CNS is also available as container images for integration into automated pipelines:

### CLI Image
```shell
docker pull ghcr.io/nvidia/cns:latest
docker run ghcr.io/nvidia/cns:latest --version
```

### API Server Image (Self-hosting)
```shell
docker pull ghcr.io/nvidia/cnsd:latest
docker run -p 8080:8080 ghcr.io/nvidia/cnsd:latest
```

## Next Steps

See [CLI Reference](cli-reference.md) for command usage

## Troubleshooting

### Command Not Found

If `cnsctl` is not found after installation:

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
sudo chmod +x /usr/local/bin/cnsctl
```

### GPU Detection Issues

Snapshot GPU measurements require `nvidia-smi` in PATH:

```shell
# Verify NVIDIA drivers
nvidia-smi

# If missing, install NVIDIA drivers for your platform
```

## Uninstall

```shell
# Remove binary
sudo rm /usr/local/bin/cnsctl

# Remove shell completion (if configured)
# Remove the source line from your shell RC file
```

## Getting Help

- **Documentation**: [User Guide](../user-guide/)
- **Issues**: [GitHub Issues](https://github.com/NVIDIA/cloud-native-stack/issues)
- **API Server**: See [Integration Guide](../integration/api-reference.md)
