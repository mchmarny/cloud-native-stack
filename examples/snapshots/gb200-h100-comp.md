# H100 vs GB200 Node Snapshot Comparison Report

## Files Compared

| System | Source | Node | Snapshot Version |
|--------|--------|------|------------------|
| H100   | AWS EKS | `ip-10-0-158-18.ec2.internal` | v0.7.10 |
| GB200  | AWS EKS | `ip-10-0-160-248.ec2.internal` | v0.7.10 |

Both snapshots use `cns.nvidia.com/v1alpha1` API version with v0.7.10 format.

> Meaningful config and capability diffs only. Ignores order, timestamps, and other expected runtime noise.

## Snapshot Structure (v0.7.10)

Both systems use the enhanced v0.7.10 snapshot format with four measurement types:

1. **SystemD Services** – Configuration of containerd, docker, kubelet, and system services
2. **OS Configuration** – 4 subtypes (release subtype in v0.7.0+):
   - `grub` – Boot parameters and kernel arguments
   - `sysctl` – Kernel parameters from `/proc/sys`
   - `kmod` – Loaded kernel modules  
   - `release` – OS identification from `/etc/os-release` (added in v0.7.0)
3. **Kubernetes** – 3 subtypes:
   - `server` – Server version with vendor-specific format support (e.g., `v1.30.14-eks-3025e55`)
   - `image` – All deployed container images with versions
   - `policy` – Complete GPU Operator ClusterPolicy configuration (100+ settings)
4. **GPU** – Hardware details and driver information

⸻

## 1. High-Level Summary

| Category | Classification | Notes |
|----------|----------------|-------|
| Kernel & Boot | Different | Same kernel family (6.8 AWS), different patch levels |
| CPU Architecture | **Same** | Both are **ARM64** |
| Platform | Different | H100 reports amd64; GB200 reports arm64 for K8s |
| GPU Architecture | **Different** | H100 is **Hopper**; GB200 is **Blackwell** |
| GPU Count | Different | H100: 8 GPUs; GB200: 4 GPUs |
| GPU Driver | Different | H100: 570.133.20; GB200: 580.82.07 |
| CUDA Version | Different | H100: 12.8; GB200: 13.1 |
| Kubernetes | Different | H100: v1.30.14-eks; GB200: v1.33.5-eks |
| GPU Operator | Different | H100: v25.3.0; GB200: v25.3.3 |
| Container Runtime | Equivalent | containerd on both |
| DCGM Version | Different | H100: 4.1.1-2; GB200: 4.3.1-1 |
| Container Toolkit | Different | H100: v1.17.5; GB200: v1.17.8 |

⸻

## 2. GPU Hardware Comparison

### GPU Architecture & Model

| Attribute | H100 | GB200 |
|-----------|------|-------|
| Model | NVIDIA H100 80GB HBM3 | NVIDIA GB200 |
| Architecture | Hopper | Blackwell |
| GPU Count | 8 | 4 |
| Driver Version | 570.133.20 | 580.82.07 |
| CUDA Version | 12.8 | 13.1 |
| Addressing Mode | HMM | ATS |
| Display Mode | Enabled | Requested functionality has been deprecated |
| Display Active | Disabled | Disabled |
| Persistence Mode | Disabled | Disabled |
| GSP Firmware | 570.133.20 | 580.82.07 |
| VBIOS Version | 96.00.BC.00.01 | 97.00.B9.00.69 |

**Classification:** Fundamental architectural difference. GB200 represents next-gen Blackwell architecture vs H100's Hopper. GB200 uses ATS (Address Translation Services) vs H100's HMM (Heterogeneous Memory Management).

⸻

## 3. Kubernetes & Container Stack

### Kubernetes Version

| System | K8s Version | Go Version | Platform |
|--------|-------------|------------|----------|
| H100 | v1.30.14-eks-3025e55 | go1.24.9 | linux/amd64 |
| GB200 | v1.33.5-eks-3025e55 | go1.24.6 | linux/arm64 |

**Classification:** GB200 runs newer K8s (v1.33 vs v1.30), and platform reporting now correctly shows H100 as amd64 and GB200 as arm64.

### GPU Operator & DCGM Stack

| Component | H100 | GB200 |
|-----------|------|-------|
| GPU Operator | v25.3.0 | v25.3.3 |
| DCGM | 4.1.1-2-ubuntu22.04 | 4.3.1-1-ubuntu22.04 |
| DCGM Exporter | 4.1.1-4.0.4-ubuntu22.04 | 4.3.1-4.4.0-ubuntu22.04 |
| Device Plugin | v0.17.1 | v0.17.4 |
| Driver Manager | v0.8.0 | v0.8.1 |
| MIG Manager | v0.12.1-ubuntu20.04 | v0.12.3-ubuntu20.04 |
| Container Toolkit | v1.17.5-ubuntu20.04 | v1.17.8-ubuntu20.04 |

**Classification:** GB200 runs newer versions across the entire GPU stack.

### Container Images Deployed

#### Unique to H100
- Run:ai suite (scheduler, workload controllers, pod-group controllers, etc.)
- Knative serving stack (activator, autoscaler, controller, webhook, kourier)
- NIM Operator (k8s-nim-operator: v2.0.1)
- NATS messaging (nats, nats-box, nats-server-config-reloader)
- LeaderWorkerSet (lws: 0.7.0)
- Grafana Alloy (alloy: v1.4.3)
- Kyverno policy engine (v1.14.3-nv.2)
- Grove Operator (v0.1.0-alpha.3)

#### Unique to GB200
- DRA Driver (dra-driver: v25.8.0)
- ArgoCD (v2.14.3)
- Bitnami images (kubectl, mongodb, nginx)
- Cert Manager suite
- Janitor (dgxc-janitor: 1.17.3)
- Enhanced NVSentinel suite (fault-quarantine, fault-remediation modules)

#### Common Components (Different Versions)
| Component | H100 | GB200 |
|-----------|------|-------|
| amazon-k8s-cni | v1.19.2-eksbuild.1 | v1.20.1-eksbuild.1 |
| kube-proxy | v1.30.6-minimal | v1.33.0 |
| driver | 570.133.20 | 580.82.07 |
| gpu-operator | v25.3.0 | v25.3.3 |

**Assessment:** H100 is heavily Run:ai-focused with workload orchestration. GB200 has DRA support and ArgoCD for GitOps.

⸻

## 4. OS Release Information (New in v0.7.0)

Both systems report identical OS release information via the new `release` subtype:

```yaml
release:
  ID: ubuntu
  VERSION_ID: "24.04"
  PRETTY_NAME: Ubuntu 24.04.3 LTS
  VERSION_CODENAME: noble
  NAME: Ubuntu
  VERSION: 24.04.3 LTS (Noble Numbat)
```

**Classification:** Identical OS base for both systems.

⸻

## 5. GPU Operator ClusterPolicy Configuration

### Key Policy Differences

#### Device Plugin Strategy
- **H100:** Standard configuration
- **GB200:** Enhanced with `DP_DISABLE_HEALTHCHECKS=109` and `DEVICE_LIST_STRATEGY=volume-mounts`

#### Driver Configuration
| Setting | H100 | GB200 |
|---------|------|-------|
| driver.version | 570.133.20 | 580.82.07 |
| driver.upgradePolicy.autoUpgrade | true | true |
| driver.rdma.enabled | true | true |
| driver.rdma.useHostMofed | false | false |

#### MIG Configuration
| Setting | H100 | GB200 |
|---------|------|-------|
| mig.strategy | single | single |
| migManager.config.default | all-disabled | all-disabled |
| migManager.config.name | default-mig-parted-config | default-mig-parted-config |

#### CDI (Container Device Interface)
- **H100:** cdi.enabled=true, cdi.default=false
- **GB200:** cdi.enabled=true, cdi.default=false

**Classification:** Both have similar policy structure with version-appropriate configurations. GB200 uses newer driver and has enhanced device plugin settings.

⸻

## 5. Kernel & Boot Configuration

### Kernel Version (with Vendor Extras)

Both systems use AWS-optimized Ubuntu kernels with vendor-specific patch levels:

| System | Kernel Version | Extras |
|--------|----------------|--------|
| H100 | 6.8.0 | -1024-aws |
| GB200 | 6.8.0 | -1028-aws |

**Classification:** GB200 uses slightly newer kernel patch level. The version parser in v0.7.0 now correctly handles vendor extras like `-1028-aws`.

**Note:** The new `Extras` field in the version parser preserves vendor-specific suffixes while maintaining semver comparison compatibility.

### Boot Flags – Key Differences

| Flag | H100 | GB200 |
|------|------|-------|
| hugepages | 5128 | 5128 |
| hugepagesz | 2M | 2M |
| numa_balancing | default | disable |
| init_on_alloc | not set | 0 |
| nokaslr | enabled | enabled |
| audit | 1 | 1 |
| audit_backlog_limit | 8192 | 8192 |

**Interpretation:** GB200 explicitly disables NUMA auto-balancing and init-on-alloc for tighter control over memory placement and determinism. Both have identical security auditing and huge pages configuration.

⸻

## 6. systemd Services

### containerd.service

- Active and enabled on both nodes
- Same drop-ins and configuration structure
- Identical cgroup delegation, limits, restart policy
- Runtime differences (CPUUsageNSec, MemoryCurrent) are expected variance

**Classification:** No meaningful configuration drift.

⸻

## 7. Key Functional Differences Summary

### 1. Snapshot Format (v0.7.0)
Both systems use the enhanced v0.7.0 snapshot format with:
- **4 OS subtypes** (new `release` subtype captures `/etc/os-release` data)
- **3 K8s subtypes** (`server`, `image`, and comprehensive `policy` with 100+ GPU operator settings)
- **Version parsing** that handles vendor-specific extras (e.g., `v1.33.5-eks-3025e55`)

### 2. GPU Generation Gap
- **H100:** Hopper architecture (compute capability 9.0), HMM addressing
- **GB200:** Blackwell architecture (next-gen), ATS addressing
- **Impact:** GB200 has architectural advances in memory coherence and compute capabilities

### 3. Workload Orchestration
- **H100:** Run:ai-centric with advanced scheduling, pod groups, queue management
- **GB200:** ArgoCD GitOps-focused, DRA support, enhanced monitoring
- **Impact:** Different operational models - H100 for complex multi-tenant scheduling, GB200 for GitOps and modern resource management

### 4. Kubernetes Version Skew (Vendor Format Support)
- **H100:** v1.30.14-eks-3025e55 (stable, Run:ai validated)
- **GB200:** v1.33.5-eks-3025e55 (latest, DRA support)
- **Impact:** GB200 can leverage newer K8s features like DRA for GPU resource allocation
- **Note:** v0.7.0 version parser correctly handles EKS vendor suffixes

### 5. GPU Stack Versions
- GB200 runs newer versions across driver, DCGM, operators
- **Impact:** Access to latest bug fixes, features, and performance improvements

### 6. Observability Stack
- **H100:** Alloy + Prometheus + Run:ai metrics
- **GB200:** Prometheus + Enhanced NVSentinel (fault detection/remediation)
- **Impact:** Different monitoring philosophies - H100 more observability-focused, GB200 more fault-prevention focused

⸻

## 8. Recommendations

1. **Version Alignment:** Consider standardizing on similar K8s and GPU Operator versions if both systems need to run identical workloads

2. **Run:ai on GB200:** If Run:ai orchestration is needed on GB200, deploy the Run:ai suite

3. **DRA on H100:** If H100 needs DRA support, upgrade to K8s v1.26+ and deploy DRA driver

4. **Driver Parity:** Both use appropriate drivers for their GPU architecture; maintain version currency for security

5. **Policy Harmonization:** The ClusterPolicy configurations are largely compatible; consider using a common baseline with architecture-specific overrides

6. **Monitoring Strategy:** Evaluate whether to standardize on one observability stack vs maintaining architecture-specific tooling
7. **Snapshot Format:** Both systems use v0.7.0 format, which includes:
   - Enhanced OS information with `release` subtype for OS identification
   - Complete K8s `policy` subtype capturing all GPU operator settings
   - Version parsing that handles vendor-specific suffixes (kernel: `-1028-aws`, K8s: `-eks-3025e55`)
   - This enables accurate comparison and configuration management across heterogeneous clusters

⸻

## Appendix: v0.7.0 Snapshot Enhancements

The v0.7.0 snapshot format introduces several key improvements:

### New OS Release Subtype
Captures comprehensive OS identification:
```yaml
release:
  ID: ubuntu
  VERSION_ID: "24.04"
  PRETTY_NAME: Ubuntu 24.04.3 LTS
  VERSION_CODENAME: noble
```

### Enhanced Version Parsing
- Kernel versions: `6.8.0-1028-aws` → Correctly parsed with extras `-1028-aws`
- K8s versions: `v1.33.5-eks-3025e55` → Correctly parsed with extras `-eks-3025e55`
- Extras preserved but don't affect version comparison for compatibility matching

### Comprehensive K8s Policy Capture
The `policy` subtype now captures 100+ GPU operator ClusterPolicy settings, enabling:
- Detailed configuration comparison between clusters
- Configuration drift detection
- Baseline policy enforcement

These enhancements make the v0.7.0 format significantly more useful for production cluster management, troubleshooting, and configuration auditing.