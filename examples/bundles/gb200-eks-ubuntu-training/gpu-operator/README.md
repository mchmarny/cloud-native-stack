# NVIDIA GPU Operator Bundle

```shell
Generated from Cloud Native Stack Recipe  
Timestamp: 2026-01-02T18:02:58Z
Bundler Version: 0.8.12
Recipe Bundler Version: 0.8.12
```

## Overview

This bundle contains everything needed to deploy the NVIDIA GPU Operator on your Kubernetes cluster.

## Contents

- `values.yaml` - Helm values configuration
- `clusterpolicy.yaml` - ClusterPolicy manifest
- `scripts/install.sh` - Installation script
- `scripts/uninstall.sh` - Uninstallation script
- `checksums.txt` - File integrity checksums

## Prerequisites

- Kubernetes cluster (1.33.5)
- Helm 3
- kubectl configured
- GPU nodes with gb200 GPUs

## Installation

### Option 1: Using the Installation Script

```bash
chmod +x scripts/install.sh
./scripts/install.sh
```

### Option 2: Manual Installation with Helm

```bash
# Add Helm repository
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update

# Create namespace
kubectl create namespace gpu-operator

# Install GPU Operator
helm upgrade --install gpu-operator nvidia/gpu-operator \
  --namespace gpu-operator \
  --values values.yaml \
  --wait
```

### Option 3: Using kubectl

```bash
kubectl apply -f clusterpolicy.yaml
```

## Verification

Check the GPU Operator pods:

```bash
kubectl get pods -n gpu-operator
```

Verify GPU nodes:

```bash
kubectl get nodes -o json | jq '.items[].status.capacity."nvidia.com/gpu"'
```

Test GPU access in a pod:

```bash
kubectl run gpu-test --rm -it --restart=Never \
  --image=nvidia/cuda:12.0.0-base-ubuntu22.04 \
  --limits=nvidia.com/gpu=1 \
  -- nvidia-smi
```

## Configuration

### Key Settings
- **Driver Version**: 580.82.07
- **GPUDirect Storage**: Enabled
- **CDI**: Enabled

### Customization

Edit `values.yaml` to customize the deployment. Key parameters:

- `driver.version` - GPU driver version
- `mig.strategy` - MIG configuration strategy
- `gds.enabled` - Enable GPUDirect Storage
- `nodeSelector` - Target specific nodes

## Uninstallation

Using the uninstallation script:

```bash
chmod +x scripts/uninstall.sh
./scripts/uninstall.sh
```

Or manually:

```bash
helm uninstall gpu-operator -n gpu-operator
kubectl delete crd clusterpolicies.nvidia.com
```

## Troubleshooting

### Check GPU Operator logs

```bash
kubectl logs -n gpu-operator -l app=nvidia-driver-daemonset
```

### Check ClusterPolicy status

```bash
kubectl describe clusterpolicy -n gpu-operator
```

### Common Issues

1. **Driver pods not starting**: Check node compatibility
2. **CUDA version mismatch**: Verify driver version
3. **MIG configuration issues**: Review MIG strategy settings

## Support

- [GPU Operator Documentation](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/)
- [Cloud Native Stack Repository](https://github.com/NVIDIA/cloud-native-stack)

## Generated Configuration
**Recipe Query:**
- OS: ubuntu
- OS Version: 24.4
- Service: eks
- GPU: gb200
- Intent: training
