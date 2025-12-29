# GPU Operator Bundle - Helm Values Gap Analysis

**Date**: December 29, 2025  
**Analysis Scope**: GPU Operator Helm values template vs official Helm chart capabilities  
**Purpose**: Identify configuration gaps and recipe measurement enhancements needed for production deployments

## Executive Summary

The current GPU Operator bundler template covers **basic deployment scenarios** with ~15 core parameters. The official GPU Operator Helm chart v25.3.1 supports **120+ configuration parameters** across 20+ components. Legacy Cloud Native Stack playbooks demonstrate usage of **40+ parameters** across 20+ distinct deployment scenarios.

**Coverage Assessment**:
- ✅ **Basic Parameters**: 100% (operator version, driver, device plugin, toolkit)
- ⚠️ **Security Parameters**: 20% (missing imagePullSecrets, licensing, registry auth)
- ⚠️ **Advanced Features**: 30% (MIG strategy only, missing profiles, confidential computing, secure boot)
- ⚠️ **Networking**: 25% (basic RDMA, missing useHostMofed, detailed configs)
- ⚠️ **Operational**: 15% (missing resources, tolerations, affinity, update strategies)
- ❌ **Enterprise**: 10% (missing vGPU details, NLS licensing, topology configs)

**Priority Classification**:
- **Critical Gaps** (P0): 12 parameters - Security, registry authentication, licensing
- **Important Gaps** (P1): 25 parameters - Resource limits, tolerations, upgrade policies
- **Optional Gaps** (P2): 80+ parameters - Advanced tuning, niche features

---

## 1. Current Template Analysis

### 1.1 Template Location
```
pkg/bundler/gpuoperator/templates/values.yaml.tmpl
```

### 1.2 Current Coverage (15 Parameters)

| Category | Parameters | Coverage |
|----------|-----------|----------|
| **Operator** | `operator.repository`, `operator.image`, `version` | ✅ Full |
| **Driver** | `driver.enabled`, `driver.version`, `driver.useOpenKernelModules`, `driver.repository` | ✅ Full |
| **Container Toolkit** | `toolkit.enabled`, `toolkit.version` | ✅ Full |
| **Device Plugin** | `devicePlugin.enabled`, `devicePlugin.version` | ✅ Full |
| **DCGM** | `dcgm.enabled`, `dcgmExporter.enabled`, `dcgm.version` | ✅ Full |
| **MIG** | `mig.strategy` | ⚠️ Basic |
| **GDS** | `gds.enabled` | ✅ Full |
| **vGPU Manager** | `vgpuManager.enabled`, `vgpuManager.licenseServerURL` | ⚠️ Basic |
| **Sandbox Workloads** | `sandboxWorkloads.enabled`, `sandboxWorkloads.defaultWorkload` | ✅ Full |
| **Node Selector** | Custom node labels | ✅ Full |
| **Namespace** | Deployment namespace | ✅ Full |

### 1.3 Template Generation Logic

The template extracts values from recipe measurements:
```go
// From pkg/bundler/gpuoperator/helm.go
func GenerateHelmValues(recipe *recipe.Recipe, cfg *config.Config) (*HelmValues, error) {
    values := &HelmValues{
        GPUOperatorVersion: extractVersion(recipe, "gpu-operator"),
        DriverEnabled:      true,
        DriverVersion:      extractVersion(recipe, "gpu-driver"),
        // ... 15 total fields
    }
}
```

---

## 2. Official Helm Chart Capabilities

### 2.1 Chart Version
**Version**: v25.3.1 (NVIDIA GPU Operator)  
**Repository**: `nvidia/gpu-operator`  
**Total Parameters**: 120+ across 20+ components

### 2.2 Component Breakdown

| Component | Parameters | Template Coverage |
|-----------|-----------|-------------------|
| Platform | 2 (openshift, etc.) | ❌ 0% |
| NFD | 4 (enabled, nodefeaturerules, worker config) | ❌ 0% |
| CDI | 3 (enabled, default, config) | ❌ 0% |
| Sandbox Workloads | 2 (enabled, defaultWorkload) | ✅ 100% |
| Host Paths | 2 (rootFS, driverInstallDir) | ❌ 0% |
| Daemonsets | 7 (labels, annotations, priorityClassName, tolerations, updateStrategy, rollingUpdate) | ❌ 0% |
| Validator | 8 (repository, image, version, imagePullPolicy, imagePullSecrets, env, args, resources, plugin) | ❌ 0% |
| Operator | 15 (repository, image, version, imagePullPolicy, imagePullSecrets, runtimeClass, tolerations, annotations, affinity, logging, resources) | ⚠️ 20% |
| MIG | 2 (strategy, migManager) | ⚠️ 50% |
| Driver | 25+ (enabled, nvidiaDriverCRD, kernelModuleType, usePrecompiled, repository, image, version, imagePullSecrets, startupProbe, rdma, upgradePolicy, manager, env, resources, repoConfig, certConfig, licensingConfig, virtualTopology, kernelModuleConfig) | ⚠️ 30% |
| Toolkit | 8 (enabled, repository, image, version, imagePullPolicy, imagePullSecrets, env, resources, installDir) | ⚠️ 25% |
| Device Plugin | 12+ (enabled, repository, image, version, imagePullSecrets, args, env, resources, config, mps) | ⚠️ 20% |
| DCGM | 7 (enabled, repository, image, version, imagePullPolicy, args, env, resources) | ⚠️ 30% |
| DCGM Exporter | 10+ (enabled, repository, image, version, env, resources, service, serviceMonitor, config) | ⚠️ 30% |
| GFD | 7 (enabled, repository, image, version, imagePullSecrets, env, resources) | ❌ 0% |
| MIG Manager | 10+ (enabled, repository, image, version, imagePullSecrets, env, resources, config, gpuClientsConfig) | ❌ 0% |
| Node Status Exporter | 6 (enabled, repository, image, version, imagePullSecrets, resources) | ❌ 0% |
| GDS | 7 (enabled, repository, image, version, imagePullPolicy, imagePullSecrets, env, args) | ⚠️ 15% |
| GDRCopy | 7 (enabled, repository, image, version, imagePullPolicy, imagePullSecrets, env, args) | ❌ 0% |
| vGPU Manager | 12+ (enabled, repository, image, version, imagePullPolicy, imagePullSecrets, env, resources, driverManager) | ⚠️ 20% |
| vGPU Device Manager | 7 (enabled, repository, image, version, imagePullSecrets, env, config) | ❌ 0% |
| VFIO Manager | 12+ (enabled, repository, image, version, imagePullSecrets, env, resources, driverManager) | ❌ 0% |
| Kata Manager | 8+ (enabled, config, repository, image, version, imagePullSecrets, env, resources) | ❌ 0% |
| Sandbox Device Plugin | 7 (enabled, repository, image, version, imagePullSecrets, args, env, resources) | ❌ 0% |
| CC Manager | 8 (enabled, defaultMode, repository, image, version, imagePullSecrets, env, resources) | ❌ 0% |
| Node Feature Discovery | 15+ (priorityClassName, gc, worker, master configs) | ❌ 0% |

---

## 3. Legacy Playbook Usage Analysis

### 3.1 Playbook Parameters (40+)

From `docs/playbooks/cns_values_*.yaml` and `operators-install.yaml`:

| Parameter | Usage | Template Support |
|-----------|-------|------------------|
| `confidential_computing` | Enable CC features | ❌ No |
| `gpu_driver_version` | Driver version | ✅ Yes |
| `use_open_kernel_module` | Open vs proprietary | ✅ Yes (useOpenKernelModules) |
| `enable_mig` | Enable MIG | ⚠️ Partial (strategy only) |
| `mig_profile` | MIG geometry (all-disabled, custom) | ❌ No |
| `mig_strategy` | single/mixed | ✅ Yes |
| `enable_gds` | GPUDirect Storage | ✅ Yes |
| `enable_secure_boot` | Secure boot support | ❌ No |
| `enable_cdi` | Container Device Interface | ❌ No |
| `enable_vgpu` | vGPU support | ✅ Yes (vgpuManager.enabled) |
| `vgpu_license_server` | NLS server URL | ✅ Yes (licenseServerURL) |
| `helm_repository` | Custom Helm repo | ❌ No |
| `gpu_operator_helm_chart` | Chart name | ❌ No |
| `gpu_operator_driver_registry` | Driver image registry | ✅ Yes |
| `gpu_operator_registry_password` | Registry password | ❌ No |
| `gpu_operator_registry_email` | Registry email | ❌ No |
| `gpu_operator_registry_username` | Registry username | ❌ No |
| `ngc_registry_password` | NGC API key | ❌ No |
| `ngc_registry_email` | NGC email | ❌ No |
| `ngc_registry_username` | NGC username ($oauthtoken) | ❌ No |
| `enable_rdma` | RDMA support | ❌ No (driver.rdma not in template) |
| `proxy` | HTTP proxy | ❌ No |
| `http_proxy` | HTTP proxy URL | ❌ No |
| `https_proxy` | HTTPS proxy URL | ❌ No |
| `cns_docker` | Docker runtime | ❌ No |
| `cns_nvidia_driver` | Pre-installed driver | ⚠️ Partial (driver.enabled) |
| `gpu_operator_version` | Operator version | ✅ Yes |

### 3.2 Deployment Scenarios from `operators-install.yaml`

The playbook contains **20+ distinct installation commands** with different `--set` combinations:

#### Scenario 1: vGPU with Licensing
```bash
helm install ... --set \
  driver.repository='{{ gpu_operator_driver_registry }}', \
  driver.version='{{ gpu_driver_version }}', \
  driver.imagePullSecrets[0]=registry-secret, \
  driver.licensingConfig.configMapName=licensing-config
```
**Gap**: Template doesn't support `driver.imagePullSecrets` or `driver.licensingConfig.configMapName`

#### Scenario 2: RDMA Enabled
```bash
helm install ... --set \
  driver.rdma.enabled=true, \
  mig.strategy='{{ mig_strategy }}', \
  driver.version='{{ gpu_driver_version }}'
```
**Gap**: Template doesn't set `driver.rdma.enabled`

#### Scenario 3: RDMA with Host MOFED
```bash
helm install ... --set \
  driver.rdma.enabled=true, \
  driver.rdma.useHostMofed=true, \
  mig.strategy='{{ mig_strategy }}'
```
**Gap**: Template doesn't support `driver.rdma.useHostMofed`

#### Scenario 4: GDS with NGC Secret
```bash
helm install ... --set \
  driver.rdma.enabled=true, \
  driver.imagePullSecrets[0]=ngc-secret, \
  gds.enabled=true
```
**Gap**: Template has `gds.enabled` but missing `driver.imagePullSecrets`

#### Scenario 5: Open Kernel Modules with GDS
```bash
helm install ... --set \
  driver.rdma.enabled=true, \
  driver.imagePullSecrets[0]=ngc-secret, \
  gds.enabled=true, \
  driver.useOpenKernelModules=true
```
**Gap**: Multiple missing parameters

#### Scenario 6: CDI Enabled
```bash
helm install ... --set \
  cdi.enabled=true, \
  driver.rdma.enabled=true, \
  driver.imagePullSecrets[0]=ngc-secret, \
  gds.enabled=true
```
**Gap**: Template doesn't support `cdi.enabled`

#### Scenario 7: Pre-installed Driver
```bash
helm install ... --set \
  driver.enabled=false, \
  toolkit.enabled=false
```
**Gap**: Template has `driver.enabled` but not `toolkit.enabled`

---

## 4. Critical Gaps (P0) - Security & Registry

### 4.1 Image Pull Secrets (Critical)

**Official Parameter**: `driver.imagePullSecrets`, `toolkit.imagePullSecrets`, `devicePlugin.imagePullSecrets`, etc.

**Current Template**: Missing

**Impact**: Cannot deploy from private registries or NGC without authentication

**Example from Playbook**:
```yaml
driver.imagePullSecrets[0]=ngc-secret
```

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: registry
    data:
      pullSecretName: "ngc-secret"  # or "registry-secret"
```

**Template Enhancement**:
```yaml
{{- if .RegistryPullSecret }}
driver:
  imagePullSecrets:
    - {{ .RegistryPullSecret }}
toolkit:
  imagePullSecrets:
    - {{ .RegistryPullSecret }}
devicePlugin:
  imagePullSecrets:
    - {{ .RegistryPullSecret }}
dcgm:
  imagePullSecrets:
    - {{ .RegistryPullSecret }}
{{- end }}
```

### 4.2 vGPU Licensing Configuration (Critical)

**Official Parameters**:
- `driver.licensingConfig.configMapName`
- `driver.licensingConfig.nlsEnabled`

**Current Template**: Only `vgpuManager.licenseServerURL`

**Impact**: Cannot deploy vGPU with proper NLS licensing

**Example from Playbook**:
```yaml
driver.licensingConfig.configMapName=licensing-config
```

**Recipe Measurement Required**:
```yaml
type: GPU
subtypes:
  - name: vgpu
    data:
      enabled: "true"
      licenseServer: "nlsserver.example.com"
      licensingConfigMap: "licensing-config"  # NEW
      nlsEnabled: "true"  # NEW
```

**Template Enhancement**:
```yaml
{{- if .VGPUEnabled }}
vgpuManager:
  enabled: true
{{- if .VGPULicenseServer }}
driver:
  licensingConfig:
    configMapName: {{ .VGPULicensingConfigMap }}
    nlsEnabled: {{ .VGPUNLSEnabled }}
{{- end }}
{{- end }}
```

### 4.3 Certificate Configuration (Critical for HTTPS registries)

**Official Parameter**: `driver.certConfig.name`

**Current Template**: Missing

**Impact**: Cannot use custom CA certificates for registry access

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: registry
    data:
      certConfigMap: "custom-ca-certs"
```

### 4.4 Repository Configuration (Important for air-gapped)

**Official Parameter**: `driver.repoConfig.configMapName`

**Current Template**: Missing

**Impact**: Cannot use custom package repositories for driver installation

---

## 5. Important Gaps (P1) - Operational

### 5.1 RDMA Configuration (Important)

**Official Parameters**:
- `driver.rdma.enabled`
- `driver.rdma.useHostMofed`

**Current Template**: Missing both

**Impact**: Cannot enable RDMA for GPUDirect, common for HPC/AI workloads

**Usage Count**: 17 matches in `operators-install.yaml`

**Recipe Measurement Required**:
```yaml
type: GPU
subtypes:
  - name: network
    data:
      rdmaEnabled: "true"
      useHostMofed: "false"  # or "true" for pre-installed MOFED
```

**Template Enhancement**:
```yaml
{{- if .RDMAEnabled }}
driver:
  rdma:
    enabled: true
    useHostMofed: {{ .UseHostMofed }}
{{- end }}
```

### 5.2 CDI (Container Device Interface) (Important)

**Official Parameters**:
- `cdi.enabled`
- `cdi.default`

**Current Template**: Missing

**Impact**: Cannot use CDI mode (default in v25.x+), required for Dynamic Resource Allocation

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: features
    data:
      cdiEnabled: "true"
      cdiDefault: "false"
```

### 5.3 MIG Configuration Details (Important)

**Official Parameters**:
- `mig.strategy` ✅ (have this)
- `migManager.enabled`
- `migManager.config.name`
- `migManager.config.create`
- `migManager.config.default`
- `migManager.config.data`

**Current Template**: Only `mig.strategy`

**Impact**: Cannot customize MIG profiles (all-disabled, custom geometries)

**Example from Playbook**:
```yaml
mig_profile: all-disabled  # or custom profiles
```

**Recipe Measurement Required**:
```yaml
type: GPU
subtypes:
  - name: mig
    data:
      enabled: "true"
      strategy: "single"  # or "mixed"
      profile: "all-disabled"  # NEW: or "custom-profile-name"
      configMapName: ""  # NEW: empty = use default
```

**Template Enhancement**:
```yaml
{{- if .MIGEnabled }}
mig:
  strategy: {{ .MIGStrategy }}
migManager:
  enabled: true
  config:
    default: {{ .MIGProfile }}
{{- end }}
```

### 5.4 Secure Boot (Important for confidential computing)

**Official Parameters**: None directly, but affects driver installation

**Current Template**: Missing

**Impact**: Cannot deploy on secure boot enabled systems

**Recipe Measurement Required**:
```yaml
type: OS
subtypes:
  - name: security
    data:
      secureBoot: "true"
```

**Note**: Secure boot support requires specific driver signing configurations

### 5.5 Confidential Computing (Important)

**Official Parameters**:
- `ccManager.enabled`
- `ccManager.defaultMode`
- `ccManager.env` (CC_CAPABLE_DEVICE_IDS)
- `nfd.nodefeaturerules` (true for CC)

**Current Template**: Missing all

**Impact**: Cannot deploy confidential computing features

**Recipe Measurement Required**:
```yaml
type: GPU
subtypes:
  - name: confidential-computing
    data:
      enabled: "true"
      defaultMode: "off"  # or "on", "devtools"
      capableDeviceIDs: "0x2339,0x2331,0x2330,0x2324,0x2322,0x233d"
```

### 5.6 Toolkit Configuration (Important)

**Official Parameter**: `toolkit.enabled`

**Current Template**: Has `toolkit.enabled` but no conditional logic

**Impact**: Cannot handle pre-installed toolkit scenarios

**Example from Playbook**:
```bash
--set driver.enabled=false,toolkit.enabled=false
```

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: preinstalled
    data:
      driver: "false"
      toolkit: "false"
```

### 5.7 Resource Limits (Important for production)

**Official Parameters**: Resources for all components
- `operator.resources`
- `driver.resources`
- `toolkit.resources`
- `devicePlugin.resources`
- `dcgm.resources`
- `dcgmExporter.resources`
- etc.

**Current Template**: Missing all

**Impact**: No resource guarantees or limits in production

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: resources
    data:
      operatorCPULimit: "500m"
      operatorMemoryLimit: "350Mi"
      # ... for each component
```

### 5.8 Tolerations and Affinity (Important for scheduling)

**Official Parameters**:
- `daemonsets.tolerations`
- `operator.tolerations`
- `operator.affinity`

**Current Template**: Has conditional nodeSelector but missing tolerations/affinity

**Impact**: Limited scheduling control in heterogeneous clusters

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: scheduling
    data:
      tolerations:
        - key: "nvidia.com/gpu"
          operator: "Exists"
          effect: "NoSchedule"
```

### 5.9 Update Strategy (Important for operations)

**Official Parameters**:
- `daemonsets.updateStrategy` ("OnDelete" or "RollingUpdate")
- `daemonsets.rollingUpdate.maxUnavailable`
- `driver.upgradePolicy` (multiple sub-parameters)

**Current Template**: Missing

**Impact**: No control over update behavior, default rolling updates may disrupt workloads

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: update-policy
    data:
      strategy: "RollingUpdate"
      maxUnavailable: "1"
      autoUpgrade: "true"
      maxParallelUpgrades: "1"
```

### 5.10 Node Feature Discovery (Important)

**Official Parameters**:
- `nfd.enabled`
- `nfd.nodefeaturerules`
- Full NFD configuration

**Current Template**: Missing (assumes NFD enabled by default)

**Impact**: Cannot disable NFD if pre-installed or customize NFD behavior

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: features
    data:
      nfdEnabled: "true"
      nfdNodeFeatureRules: "false"  # true for CC
```

---

## 6. Optional Gaps (P2) - Advanced Features

### 6.1 GDRCopy (Optional)

**Official Parameters**: 7 parameters (enabled, repository, image, version, imagePullPolicy, imagePullSecrets, env, args)

**Current Template**: Missing

**Impact**: Cannot enable GDRCopy for low-latency GPU-CPU transfers

### 6.2 Kata Manager (Optional for VM workloads)

**Official Parameters**: 8+ parameters for Kata runtime support

**Current Template**: Missing

**Impact**: Cannot run GPU workloads in Kata containers

### 6.3 VFIO Manager (Optional for VM passthrough)

**Official Parameters**: 12+ parameters

**Current Template**: Missing

**Impact**: Cannot use VFIO for GPU passthrough scenarios

### 6.4 Platform Specific (Optional)

**Official Parameter**: `platform.openshift`

**Current Template**: Missing

**Impact**: No OpenShift-specific optimizations

### 6.5 Component Repository Overrides (Optional)

**Official Parameters**: Each component has `repository`, `image`, `version`

**Current Template**: Only operator and driver repositories

**Impact**: Cannot use custom images for all components

### 6.6 Environment Variables (Optional per component)

**Official Parameters**: `driver.env`, `toolkit.env`, `devicePlugin.env`, etc.

**Current Template**: Missing

**Impact**: Cannot set custom environment variables

### 6.7 Startup Probe Configuration (Optional)

**Official Parameters**: `driver.startupProbe` (initialDelaySeconds, periodSeconds, timeoutSeconds, failureThreshold)

**Current Template**: Missing

**Impact**: No customization of driver startup behavior

### 6.8 Service Monitor (Optional for Prometheus)

**Official Parameters**: `dcgmExporter.serviceMonitor` (enabled, interval, honorLabels, additionalLabels, relabelings)

**Current Template**: Missing

**Impact**: Cannot customize DCGM metrics collection

### 6.9 Device Plugin Configuration (Optional)

**Official Parameters**: `devicePlugin.config` (ConfigMap with device plugin configurations)

**Current Template**: Missing

**Impact**: Cannot use time-slicing or custom device plugin configs

### 6.10 Priority Class Names (Optional)

**Official Parameters**: `daemonsets.priorityClassName`, `operator.priorityClassName`

**Current Template**: Missing

**Impact**: Default priority (system-node-critical), no customization

---

## 7. Summary of Gaps by Category

| Category | Total Available | Currently Supported | Gap Count | Priority |
|----------|----------------|---------------------|-----------|----------|
| **Security** | 15 | 3 (20%) | 12 | P0 Critical |
| **RDMA/Networking** | 4 | 0 (0%) | 4 | P1 Important |
| **MIG Configuration** | 10 | 1 (10%) | 9 | P1 Important |
| **Confidential Computing** | 8 | 0 (0%) | 8 | P1 Important |
| **CDI** | 3 | 0 (0%) | 3 | P1 Important |
| **Pre-installed Components** | 4 | 1 (25%) | 3 | P1 Important |
| **Resources & Scheduling** | 25 | 1 (4%) | 24 | P1 Important |
| **Update Policies** | 12 | 0 (0%) | 12 | P1 Important |
| **NFD** | 15 | 0 (0%) | 15 | P1 Important |
| **GDRCopy** | 7 | 0 (0%) | 7 | P2 Optional |
| **Kata/VFIO** | 20 | 0 (0%) | 20 | P2 Optional |
| **Component Images** | 30 | 4 (13%) | 26 | P2 Optional |
| **Environment Variables** | 20+ | 0 (0%) | 20+ | P2 Optional |
| **Probes & Monitors** | 15 | 0 (0%) | 15 | P2 Optional |

**Total Identified Gaps**: 180+ parameters  
**P0 Critical**: 12 parameters (security, authentication, licensing)  
**P1 Important**: 78 parameters (RDMA, MIG, CC, CDI, resources, scheduling, updates)  
**P2 Optional**: 90+ parameters (advanced features, niche use cases)

---

## 8. Recipe Measurement Schema Enhancements

To close the identified gaps, the recipe measurement schema needs the following additions:

### 8.1 Enhanced K8s Measurement

```yaml
type: K8s
subtypes:
  - name: registry
    data:
      pullSecretName: "ngc-secret"
      certConfigMap: "custom-ca-certs"
      driverRepoConfig: "repo-config"
  
  - name: features
    data:
      cdiEnabled: "true"
      cdiDefault: "false"
      nfdEnabled: "true"
      nfdNodeFeatureRules: "false"
  
  - name: preinstalled
    data:
      driver: "false"
      toolkit: "false"
  
  - name: resources
    data:
      operatorCPULimit: "500m"
      operatorMemoryLimit: "350Mi"
      driverCPULimit: "1000m"
      driverMemoryLimit: "2Gi"
      # ... for each component
  
  - name: scheduling
    data:
      priorityClassName: "system-node-critical"
      tolerations: |
        - key: nvidia.com/gpu
          operator: Exists
          effect: NoSchedule
      nodeAffinity: |
        # YAML affinity spec
  
  - name: update-policy
    data:
      strategy: "RollingUpdate"
      maxUnavailable: "1"
      autoUpgrade: "true"
      maxParallelUpgrades: "1"
      drainEnabled: "false"
```

### 8.2 Enhanced GPU Measurement

```yaml
type: GPU
subtypes:
  - name: network
    data:
      rdmaEnabled: "true"
      useHostMofed: "false"
  
  - name: mig
    data:
      enabled: "true"
      strategy: "single"
      profile: "all-disabled"
      configMapName: ""
  
  - name: confidential-computing
    data:
      enabled: "false"
      defaultMode: "off"
      capableDeviceIDs: "0x2339,0x2331,0x2330,0x2324,0x2322,0x233d"
  
  - name: vgpu
    data:
      enabled: "false"
      licenseServer: ""
      licensingConfigMap: "licensing-config"
      nlsEnabled: "true"
  
  - name: advanced
    data:
      gdsEnabled: "false"
      gdrcopyEnabled: "false"
```

### 8.3 New OS Measurement Subtype

```yaml
type: OS
subtypes:
  - name: security
    data:
      secureBoot: "false"
```

---

## 9. Recommended Implementation Phases

### Phase 1: Critical Security Gaps (P0) - Target: Q1 2026

**Goal**: Enable secure production deployments with registry authentication

1. Add `driver.imagePullSecrets` support (all components)
2. Add `driver.licensingConfig` for vGPU NLS
3. Add `driver.certConfig` for custom CA certs
4. Add `driver.repoConfig` for custom package repos

**Recipe Changes**:
- Add `K8s/registry` subtype
- Extend `GPU/vgpu` subtype

**Template Changes**:
- Add conditional imagePullSecrets to all components
- Add licensingConfig section
- Add certConfig section
- Add repoConfig section

**Estimated Effort**: 2-3 days

### Phase 2: RDMA and Networking (P1) - Target: Q1 2026

**Goal**: Support HPC/AI workloads requiring GPUDirect

1. Add `driver.rdma.enabled`
2. Add `driver.rdma.useHostMofed`

**Recipe Changes**:
- Add `GPU/network` subtype

**Template Changes**:
- Add rdma configuration section

**Estimated Effort**: 1 day

### Phase 3: Advanced GPU Features (P1) - Target: Q2 2026

**Goal**: Support MIG, CDI, and Confidential Computing

1. Add full MIG configuration (profile, configMap)
2. Add CDI support
3. Add Confidential Computing Manager
4. Add NFD configuration options

**Recipe Changes**:
- Extend `GPU/mig` subtype
- Add `K8s/features` subtype
- Add `GPU/confidential-computing` subtype

**Template Changes**:
- Extend MIG section with profile support
- Add CDI configuration
- Add CCManager configuration
- Add NFD configuration

**Estimated Effort**: 3-4 days

### Phase 4: Operational Excellence (P1) - Target: Q2 2026

**Goal**: Production-ready with resource management and update policies

1. Add resource limits/requests for all components
2. Add tolerations and affinity
3. Add update strategies and upgrade policies
4. Add pre-installed component handling

**Recipe Changes**:
- Add `K8s/resources` subtype
- Add `K8s/scheduling` subtype
- Add `K8s/update-policy` subtype
- Extend `K8s/preinstalled` subtype

**Template Changes**:
- Add resources section for all components
- Add tolerations and affinity
- Add update strategy configuration
- Add conditional logic for pre-installed components

**Estimated Effort**: 4-5 days

### Phase 5: Optional Advanced Features (P2) - Target: Q3 2026

**Goal**: Support niche use cases (GDRCopy, Kata, VFIO, etc.)

1. Add GDRCopy support
2. Add Kata Manager
3. Add VFIO Manager
4. Add component repository overrides
5. Add environment variable customization
6. Add probe configuration
7. Add service monitor customization

**Recipe Changes**:
- Add `GPU/advanced` subtype extensions
- Add per-component configuration options

**Template Changes**:
- Add GDRCopy section
- Add Kata Manager section
- Add VFIO Manager section
- Add per-component customization

**Estimated Effort**: 5-7 days

---

## 10. Validation Plan

### 10.1 Test Scenarios

For each implemented phase, validate against these scenarios:

1. **Basic Deployment** (Baseline)
   - Fresh cluster, no pre-existing components
   - Default configurations
   
2. **Private Registry** (Phase 1)
   - NGC registry with authentication
   - Custom private registry
   - Air-gapped environment

3. **vGPU with Licensing** (Phase 1)
   - NLS server configuration
   - License configMap

4. **RDMA Enabled** (Phase 2)
   - RDMA without host MOFED
   - RDMA with host MOFED

5. **MIG Configurations** (Phase 3)
   - All-disabled profile
   - Custom MIG geometry
   - Single vs mixed strategy

6. **CDI Mode** (Phase 3)
   - CDI enabled with K8s 1.31+
   - DRA feature gates

7. **Confidential Computing** (Phase 3)
   - H100 with CC enabled
   - CCManager deployed

8. **Resource Constrained** (Phase 4)
   - Resource limits enforced
   - Scheduling with node affinity

9. **Pre-installed Components** (Phase 4)
   - Pre-installed driver
   - Pre-installed toolkit
   - Both pre-installed

10. **Update Scenarios** (Phase 4)
    - Rolling update with maxUnavailable
    - Driver upgrade policy

### 10.2 Acceptance Criteria

Each scenario must:
- ✅ Generate syntactically valid Helm values
- ✅ Deploy successfully via generated install script
- ✅ Pass GPU Operator validation
- ✅ Match equivalent manual `helm install --set` command
- ✅ Handle error cases gracefully (missing measurements)

---

## 11. Migration Path for Existing Users

### 11.1 For Legacy Playbook Users

Users migrating from Ansible playbooks should:

1. **Phase 1-2**: Basic migration possible with security and RDMA
   - Map existing `cns_values.yaml` to recipe measurements
   - Generate bundles with security configurations
   - Validate against existing clusters

2. **Phase 3**: Full feature parity
   - Migrate MIG, CDI, CC configurations
   - Test advanced GPU features

3. **Phase 4**: Operational parity
   - Migrate resource limits from playbook
   - Configure update policies
   - Handle pre-installed scenarios

### 11.2 For New Users

New users should:

1. Start with basic recipes (current template)
2. Add security configurations (Phase 1)
3. Progressively enable advanced features as needed

---

## 12. Conclusion

The current GPU Operator bundler provides a **solid foundation for basic deployments** but has significant gaps for **production and enterprise use cases**. The identified gaps span:

- **12 critical security parameters** needed for private registries and licensing
- **78 important operational parameters** for RDMA, MIG, resources, and updates
- **90+ optional parameters** for advanced features

**Recommendation**: Implement in phases, prioritizing P0 (security) and P1 (RDMA, MIG, resources) gaps to achieve production readiness within 6 months.

**Next Steps**:
1. Review and approve this gap analysis
2. Create GitHub issues for each phase
3. Design recipe measurement schema enhancements
4. Implement Phase 1 (security) as highest priority
5. Create similar analysis for Network Operator

---

## Appendix A: Reference Commands from Legacy Playbooks

### A.1 Basic Installation
```bash
helm install --version {{ gpu_operator_version }} \
  --create-namespace --namespace nvidia-gpu-operator \
  '{{ gpu_operator_helm_chart }}' \
  --set driver.version='{{ gpu_driver_version }}' \
  --wait --generate-name
```

### A.2 With RDMA
```bash
helm install --version {{ gpu_operator_version }} \
  --create-namespace --namespace nvidia-gpu-operator \
  '{{ gpu_operator_helm_chart }}' \
  --set driver.rdma.enabled=true, \
       driver.version='{{ gpu_driver_version }}', \
       mig.strategy='{{ mig_strategy }}' \
  --wait --generate-name
```

### A.3 With NGC Authentication and GDS
```bash
helm install --version {{ gpu_operator_version }} \
  --create-namespace --namespace nvidia-gpu-operator \
  '{{ gpu_operator_helm_chart }}' \
  --set driver.rdma.enabled=true, \
       driver.version='{{ gpu_driver_version }}', \
       driver.imagePullSecrets[0]=ngc-secret, \
       gds.enabled=true, \
       driver.useOpenKernelModules=true \
  --wait --generate-name
```

### A.4 vGPU with Licensing
```bash
helm install --version {{ gpu_operator_version }} \
  --create-namespace --namespace nvidia-gpu-operator \
  '{{ gpu_operator_helm_chart }}' \
  --set driver.repository='{{ gpu_operator_driver_registry }}', \
       driver.version='{{ gpu_driver_version }}', \
       driver.imagePullSecrets[0]=registry-secret, \
       driver.licensingConfig.configMapName=licensing-config \
  --wait --generate-name
```

### A.5 Pre-installed Driver
```bash
helm install --version {{ gpu_operator_version }} \
  --create-namespace --namespace nvidia-gpu-operator \
  '{{ gpu_operator_helm_chart }}' \
  --set driver.enabled=false, \
       toolkit.enabled=false \
  --wait --generate-name
```

---

**Document Version**: 1.0  
**Last Updated**: December 29, 2025  
**Author**: Cloud Native Stack Engineering  
**Review Status**: Draft - Pending Approval
