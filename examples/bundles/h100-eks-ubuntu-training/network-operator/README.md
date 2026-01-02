# Network Operator Deployment Bundle

```shell
Generated from Cloud Native Stack Recipe
Timestamp: 2026-01-02T18:01:22Z
Bundler Version: 0.8.12
Recipe Bundler Version: 0.8.12
```

## Overview

This bundle contains everything needed to deploy the NVIDIA Network Operator on your Kubernetes cluster. The Network Operator enables advanced networking features including RDMA, SR-IOV, and high-performance network device plugins.

## Prerequisites

- Kubernetes cluster (version 1.30.14 or compatible)
- kubectl configured with cluster access
- Helm 3.x installed
- Cluster admin permissions
- NVIDIA Mellanox ConnectX NICs (for RDMA/SR-IOV features)

## Bundle Contents

```
network-operator/
├── values.yaml                    # Helm chart configuration
├── manifests/
│   └── nicclusterpolicy.yaml     # NicClusterPolicy custom resource
├── scripts/
│   ├── install.sh                # Automated installation
│   └── uninstall.sh              # Cleanup script
├── README.md                      # This file
└── checksums.txt                  # SHA256 checksums
```

## Configuration

This bundle is configured with the following settings:

**Network Operator Version:** {v25.4.0 Override from bundler configuration}
**OFED Driver Version:** {<nil> }
**Namespace:** network-operator

### Features Enabled
- ✅ RDMA Shared Device Plugin
- ✅ SR-IOV Device Plugin
- ✅ Host Device Plugin
- ✅ NVIDIA IPAM Plugin
- ✅ Multus CNI
- ✅ OFED Driver Deployment

## Quick Start

### Option 1: Automated Installation (Recommended)

The easiest way to install is using the provided installation script:

```bash
# Verify checksums
sha256sum -c checksums.txt

# Make script executable
chmod +x scripts/install.sh

# Run installation
./scripts/install.sh
```

The script will:
1. Check prerequisites (kubectl, helm)
2. Create the namespace
3. Add the Helm repository
4. Install the Network Operator with your configuration
5. Apply the NicClusterPolicy
6. Wait for pods to be ready

### Option 2: Manual Installation

If you prefer manual control:

```bash
# Create namespace
kubectl create namespace network-operator

# Add Helm repository
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update

# Install Network Operator
helm install network-operator nvidia/network-operator \
    --namespace network-operator \
    --values values.yaml \
    --create-namespace

# Apply NicClusterPolicy
kubectl apply -f manifests/nicclusterpolicy.yaml
```

## Verification

Check the installation status:

```bash
# Check Network Operator pods
kubectl get pods -n network-operator

# Check NicClusterPolicy
kubectl get nicclusterpolicy -n network-operator

# Check operator logs
kubectl logs -n network-operator -l app.kubernetes.io/component=network-operator
```

### Verify RDMA

Check that RDMA devices are discovered and available:

```bash
# Check node allocatable resources
kubectl get nodes -o json | jq '.items[].status.allocatable' | grep rdma

# Check RDMA shared device plugin
kubectl get pods -n network-operator -l app=rdma-shared-device-plugin
```

### Verify SR-IOV

Check SR-IOV network node states:

```bash
# List SR-IOV network node states
kubectl get sriovnetworknodestates -n network-operator

# Check SR-IOV device plugin
kubectl get pods -n network-operator -l app=sriov-device-plugin
```

## Configuration Customization

### Modifying values.yaml

You can customize the deployment by editing `values.yaml` before installation:

```yaml
# Example: Enable additional features
nvIPAM:
  deploy: true
  config:
    enableWebhook: true

# Example: Change OFED version
ofedDriver:
  version: "24.01-0.3.3.1"
```

### Node Selection

To deploy Network Operator components on specific nodes, add node selectors:

```yaml
nodeAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
    nodeSelectorTerms:
    - matchExpressions:
      - key: node-role.kubernetes.io/worker
        operator: In
        values:
        - "true"
```

## Troubleshooting

### Pods Not Starting

Check pod status and events:

```bash
kubectl get pods -n network-operator
kubectl describe pod <pod-name> -n network-operator
```

### OFED Driver Issues

If OFED driver pods fail to start:

```bash
# Check driver container logs
kubectl logs -n network-operator -l app=mofed-driver -c mofed-container

# Verify kernel compatibility
kubectl get nodes -o wide
```

### RDMA Device Not Found

Verify NICs are properly detected:

```bash
# On the node, check lspci
lspci | grep Mellanox

# Check device plugin configuration
kubectl get cm -n network-operator rdma-devices -o yaml
```

### Network Policy Issues

Check NicClusterPolicy status:

```bash
kubectl get nicclusterpolicy -n network-operator -o yaml
kubectl describe nicclusterpolicy -n network-operator
```

## Uninstallation

To remove Network Operator from your cluster:

```bash
# Using the uninstall script (recommended)
chmod +x scripts/uninstall.sh
./scripts/uninstall.sh

# Or manually
kubectl delete nicclusterpolicy --all -n network-operator
helm uninstall network-operator -n network-operator
kubectl delete namespace network-operator
```

**Warning:** Uninstalling Network Operator will remove all network configurations and may disrupt running workloads using RDMA or SR-IOV devices.

## Additional Resources

- [Network Operator Documentation](https://docs.nvidia.com/networking/display/COKAN10/Network+Operator)
- [RDMA Configuration Guide](https://docs.nvidia.com/networking/display/COKAN10/RDMA+Shared+Device+Plugin)
- [SR-IOV Setup Guide](https://docs.nvidia.com/networking/display/COKAN10/SR-IOV+Device+Plugin)
- [NVIDIA IPAM Plugin](https://github.com/Mellanox/nvidia-k8s-ipam)

## Support

For issues specific to this deployment bundle, refer to the Cloud Native Stack documentation.

For Network Operator issues:
- GitHub Issues: https://github.com/Mellanox/network-operator/issues
- NVIDIA Enterprise Support: https://www.nvidia.com/en-us/support/

---

**Note:** This bundle was automatically generated from your system configuration and workload requirements. Verify all settings align with your environment before deployment.
