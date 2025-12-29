# Network Operator Bundle - Helm Values Gap Analysis

**Date**: December 29, 2025  
**Analysis Scope**: Network Operator Helm values template vs official Helm chart capabilities  
**Purpose**: Identify configuration gaps and recipe measurement enhancements needed for production deployments

## Executive Summary

The current Network Operator bundler template covers **basic deployment scenarios** with ~13 core parameters focused on RDMA and SR-IOV. The official Network Operator Helm chart v25.4.0 supports **150+ configuration parameters** across 10+ components including SR-IOV Network Operator, NIC Configuration Operator, and Maintenance Operator. Legacy Cloud Native Stack playbooks demonstrate usage of **3 primary parameters** with focus on RDMA and OFED deployment.

**Coverage Assessment**:
- ✅ **Basic RDMA Parameters**: 100% (RDMA shared device plugin, basic config)
- ✅ **Basic SR-IOV Parameters**: 60% (device plugin only, missing resource configs)
- ✅ **IPAM Parameters**: 50% (basic nvIPAM, missing advanced configs)
- ⚠️ **Secondary Networking**: 50% (Multus, Whereabouts basics only)
- ⚠️ **OFED Driver**: 30% (deploy and version, missing upgrade policies and repository configs)
- ❌ **Security Parameters**: 0% (missing imagePullSecrets, admission controllers, certificates)
- ❌ **SR-IOV Advanced**: 10% (missing operator config, resource pools, network policies)
- ❌ **NIC Configuration**: 0% (operator not supported)
- ❌ **Maintenance Operator**: 0% (operator not supported)
- ❌ **Operational**: 5% (missing resources, tolerations, affinity, update strategies)

**Priority Classification**:
- **Critical Gaps** (P0): 8 parameters - Security (imagePullSecrets), SR-IOV resource configuration
- **Important Gaps** (P1): 35 parameters - OFED upgrade policies, resource limits, admission controllers
- **Optional Gaps** (P2): 100+ parameters - NIC Configuration Operator, Maintenance Operator, advanced tuning

---

## 1. Current Template Analysis

### 1.1 Template Location
```
pkg/bundler/networkoperator/templates/values.yaml.tmpl
```

### 1.2 Current Coverage (13 Parameters)

| Category | Parameters | Coverage |
|----------|-----------|----------|
| **Operator** | `operator.tag` | ✅ Full |
| **Deploy CR** | `deployCR` | ✅ Full (hardcoded true) |
| **OFED Driver** | `ofedDriver.deploy`, `ofedDriver.version` | ⚠️ Basic |
| **RDMA Device Plugin** | `rdmaSharedDevicePlugin.deploy`, `rdmaSharedDevicePlugin.config.resources` | ✅ Full |
| **SR-IOV Device Plugin** | `sriovDevicePlugin.deploy` | ⚠️ Basic |
| **IPAM** | `nvIPAM.deploy`, `nvIPAM.config.enableWebhook` | ⚠️ Basic |
| **Secondary Network** | `secondaryNetwork.deploy`, `secondaryNetwork.multus.deploy`, `secondaryNetwork.ipamPlugin.deploy` | ⚠️ Basic |
| **NIC Feature Discovery** | `nicFeatureDiscovery.deploy` | ✅ Full (hardcoded true) |
| **NFD** | `nfd.enabled`, `nfd.deployNodeFeatureRules` | ✅ Full |
| **Node Affinity** | Custom node labels | ✅ Full |
| **Container Runtime** | `containerRuntime.socket` | ✅ Full |
| **Namespace** | Deployment namespace | ✅ Full |

### 1.3 Template Generation Logic

The template extracts values from recipe measurements:
```go
// From pkg/bundler/networkoperator/helm.go
func GenerateHelmValues(recipe *recipe.Recipe, cfg *config.Config) (*HelmValues, error) {
    values := &HelmValues{
        NetworkOperatorVersion: extractVersion(recipe, "network-operator"),
        DeployOFED:            extractBool(recipe, "deploy-ofed"),
        OFEDVersion:           extractVersion(recipe, "ofed-driver"),
        EnableRDMA:            extractBool(recipe, "rdma-enabled"),
        EnableSRIOV:           extractBool(recipe, "sriov-enabled"),
        // ... 14 total fields
    }
}
```

---

## 2. Official Helm Chart Capabilities

### 2.1 Chart Version
**Version**: v25.4.0 (NVIDIA Network Operator)  
**Repository**: `nvidia/network-operator`  
**Total Parameters**: 150+ across 10+ components

### 2.2 Component Breakdown

| Component | Parameters | Template Coverage |
|-----------|-----------|-------------------|
| NFD | 4 (enabled, deployNodeFeatureRules) | ✅ 100% |
| Upgrade CRDs | 1 (upgradeCRDs) | ❌ 0% |
| SR-IOV Network Operator | 30+ (enabled, images, config, admissionControllers, certificates) | ⚠️ 10% |
| NIC Configuration Operator | 10+ (enabled, operator image, daemon image) | ❌ 0% |
| Maintenance Operator | 15+ (enabled, operator image, admission controller, certificates) | ❌ 0% |
| Node Feature Discovery (sub-chart) | 25+ (enableNodeFeatureApi, featureGates, master config, gc, worker config) | ⚠️ 15% |
| SR-IOV Network Operator (sub-chart) | 40+ (operator config, resource prefix, admission controllers, images, sriovOperatorConfig) | ⚠️ 5% |
| Operator General | 20+ (resources, tolerations, nodeSelector, affinity, repository, image, nameOverride, cniBinDirectory, maintenanceOperator, useDTK, admissionController, ofedDriver init container) | ⚠️ 15% |
| Image Pull Secrets | 1 (imagePullSecrets) | ❌ 0% |

**Key Sub-Components**:
- **SR-IOV Components**: 8 images (operator, config daemon, SR-IOV CNI, IB SR-IOV CNI, OVS CNI, device plugin, resources injector, webhook)
- **SR-IOV Operator Config**: Node selector, daemon configuration
- **Admission Controllers**: Operator, SR-IOV operator, maintenance operator
- **Certificate Management**: Custom certs, cert-manager integration for 3 operators

---

## 3. Legacy Playbook Usage Analysis

### 3.1 Playbook Parameters (3 Primary)

From `docs/playbooks/cns_values_*.yaml`:

| Parameter | Usage | Template Support |
|-----------|-------|------------------|
| `enable_network_operator` | Enable/disable deployment | ✅ Yes (implicit) |
| `enable_rdma` | Enable RDMA support | ✅ Yes (rdmaSharedDevicePlugin.deploy) |
| `deploy_ofed` | Deploy MLNX-OFED driver | ✅ Yes (ofedDriver.deploy) |
| `network_operator_version` | Operator version | ✅ Yes (operator.tag) |

**Note**: Legacy playbooks have minimal Network Operator configuration compared to GPU Operator, primarily focusing on RDMA enablement.

### 3.2 Deployment Scenarios from Playbooks

Unlike GPU Operator, Network Operator has fewer documented scenarios in legacy playbooks:

#### Scenario 1: Basic Network Operator with RDMA
```yaml
enable_network_operator: yes
enable_rdma: yes
deploy_ofed: no  # Use host MOFED
```
**Template Support**: ✅ Fully supported

#### Scenario 2: Network Operator with OFED Deployment
```yaml
enable_network_operator: yes
enable_rdma: yes
deploy_ofed: yes  # Deploy MLNX-OFED container
```
**Template Support**: ✅ Fully supported

#### Scenario 3: Network Operator Disabled
```yaml
enable_network_operator: no
```
**Template Support**: ✅ Supported (don't deploy bundle)

---

## 4. Critical Gaps (P0) - Security & SR-IOV Configuration

### 4.1 Image Pull Secrets (Critical)

**Official Parameter**: `imagePullSecrets` (top-level)

**Current Template**: Missing

**Impact**: Cannot deploy from private registries or NGC without authentication. Affects all Network Operator images.

**Example Usage**:
```yaml
imagePullSecrets:
  - name: ngc-secret
```

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: registry
    data:
      pullSecretName: "ngc-secret"
```

**Template Enhancement**:
```yaml
{{- if .RegistryPullSecret }}
imagePullSecrets:
  - name: {{ .RegistryPullSecret }}
{{- end }}
```

### 4.2 SR-IOV Operator Configuration (Critical)

**Official Parameters**:
- `sriov-network-operator.operator.resourcePrefix` (default: nvidia.com)
- `sriov-network-operator.sriovOperatorConfig.deploy`
- `sriov-network-operator.sriovOperatorConfig.configDaemonNodeSelector`

**Current Template**: Missing all

**Impact**: Cannot customize SR-IOV resource naming or node selection for SR-IOV configuration daemon

**Recipe Measurement Required**:
```yaml
type: Network
subtypes:
  - name: sriov-operator
    data:
      resourcePrefix: "nvidia.com"
      deployConfig: "true"
      configDaemonNodeSelector: |
        beta.kubernetes.io/os: "linux"
        network.nvidia.com/operator.mofed.wait: "false"
```

**Template Enhancement**:
```yaml
sriov-network-operator:
  operator:
    resourcePrefix: {{ .SRIOVResourcePrefix | default "nvidia.com" }}
  sriovOperatorConfig:
    deploy: {{ .SRIOVDeployConfig | default true }}
    {{- if .SRIOVConfigDaemonNodeSelector }}
    configDaemonNodeSelector:
      {{ .SRIOVConfigDaemonNodeSelector | nindent 6 }}
    {{- end }}
```

### 4.3 SR-IOV Network Operator Images (Critical)

**Official Parameters**: 8 image references in `sriov-network-operator.images`:
- `operator`
- `sriovConfigDaemon`
- `sriovCni`
- `ibSriovCni`
- `ovsCni`
- `sriovDevicePlugin`
- `resourcesInjector`
- `webhook`

**Current Template**: Uses defaults from chart

**Impact**: Cannot use custom images or air-gapped registries for SR-IOV components

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: sriov-images
    data:
      operator: "nvcr.io/nvidia/mellanox/sriov-network-operator:network-operator-25.4.0"
      sriovConfigDaemon: "nvcr.io/nvidia/mellanox/sriov-network-operator-config-daemon:network-operator-25.4.0"
      # ... other images
```

### 4.4 OFED Driver Init Container (Important)

**Official Parameters**:
- `operator.ofedDriver.initContainer.enable`
- `operator.ofedDriver.initContainer.repository`
- `operator.ofedDriver.initContainer.image`
- `operator.ofedDriver.initContainer.version`

**Current Template**: Missing (uses defaults)

**Impact**: Cannot customize or disable OFED init container behavior

**Recipe Measurement Required**:
```yaml
type: Network
subtypes:
  - name: ofed-init
    data:
      enabled: "true"
      repository: "ghcr.io/mellanox"
      image: "network-operator-init-container"
      version: "v0.0.3"
```

---

## 5. Important Gaps (P1) - Operational & Advanced Features

### 5.1 Operator Resources & Scheduling (Important)

**Official Parameters**:
- `operator.resources` (limits, requests)
- `operator.tolerations`
- `operator.nodeSelector`
- `operator.affinity`

**Current Template**: Missing all (except nodeAffinity for custom labels)

**Impact**: No resource guarantees, limited scheduling control

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: operator-resources
    data:
      cpuLimit: "500m"
      memoryLimit: "128Mi"
      cpuRequest: "5m"
      memoryRequest: "64Mi"
  
  - name: operator-scheduling
    data:
      tolerations: |
        - key: "node-role.kubernetes.io/master"
          operator: "Equal"
          value: ""
          effect: "NoSchedule"
        - key: "node-role.kubernetes.io/control-plane"
          operator: "Equal"
          value: ""
          effect: "NoSchedule"
```

**Template Enhancement**:
```yaml
operator:
  {{- if .OperatorResources }}
  resources:
    limits:
      cpu: {{ .OperatorCPULimit }}
      memory: {{ .OperatorMemoryLimit }}
    requests:
      cpu: {{ .OperatorCPURequest }}
      memory: {{ .OperatorMemoryRequest }}
  {{- end }}
  {{- if .OperatorTolerations }}
  tolerations:
    {{ .OperatorTolerations | nindent 4 }}
  {{- end }}
```

### 5.2 Admission Controllers (Important for validation)

**Official Parameters**:
- `operator.admissionController.enabled`
- `operator.admissionController.useCertManager`
- `operator.admissionController.certificate` (custom certs)
- `sriov-network-operator.operator.admissionControllers.enabled`
- `sriov-network-operator.operator.admissionControllers.certificates`
- `maintenance-operator-chart.operator.admissionController.enable`

**Current Template**: Missing all

**Impact**: No validation webhooks for NicClusterPolicy or SR-IOV resources, potential misconfigurations

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: admission-control
    data:
      networkOperatorEnabled: "false"  # default
      networkOperatorUseCertManager: "true"
      sriovOperatorEnabled: "false"  # default
      sriovOperatorUseCertManager: "true"
```

**Template Enhancement**:
```yaml
operator:
  admissionController:
    enabled: {{ .NetworkOperatorAdmissionEnabled | default false }}
    useCertManager: {{ .NetworkOperatorUseCertManager | default true }}

sriov-network-operator:
  operator:
    admissionControllers:
      enabled: {{ .SRIOVAdmissionEnabled | default false }}
```

### 5.3 CNI Bin Directory (Important for runtime compatibility)

**Official Parameter**: `operator.cniBinDirectory`

**Current Template**: Missing (uses default `/opt/cni/bin`)

**Impact**: May not work with custom CNI configurations or alternative container runtimes

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: cni-config
    data:
      binDirectory: "/opt/cni/bin"
```

### 5.4 Maintenance Operator Integration (Important for updates)

**Official Parameters**:
- `operator.maintenanceOperator.useRequestor`
- `operator.maintenanceOperator.requestorID`
- `operator.maintenanceOperator.nodeMaintenanceNamePrefix`
- `operator.maintenanceOperator.nodeMaintenanceNamespace`
- `maintenanceOperator.enabled`

**Current Template**: Missing all

**Impact**: No automated maintenance mode during upgrades, manual node cordoning required

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: maintenance
    data:
      enabled: "false"
      useRequestor: "false"
      requestorID: "nvidia.network.operator"
      nodeMaintenanceNamePrefix: "network-operator"
      nodeMaintenanceNamespace: "default"
```

### 5.5 Node Feature Discovery Advanced Config (Important)

**Official Parameters**: 25+ parameters in `node-feature-discovery` sub-chart
- `node-feature-discovery.enableNodeFeatureApi`
- `node-feature-discovery.featureGates.NodeFeatureAPI`
- `node-feature-discovery.postDeleteCleanup`
- `node-feature-discovery.master.config`
- `node-feature-discovery.gc` configuration
- `node-feature-discovery.worker.tolerations`
- `node-feature-discovery.worker.config`

**Current Template**: Only `nfd.enabled` and `nfd.deployNodeFeatureRules`

**Impact**: Cannot customize NFD behavior, feature gate configuration, or cleanup policies

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: nfd-config
    data:
      enableNodeFeatureApi: "true"
      nodeFeatureAPIGate: "true"
      postDeleteCleanup: "false"
      extraLabelNs: "nvidia.com"
      gcEnabled: "true"
      gcReplicaCount: "1"
```

### 5.6 SR-IOV Operator Enabled (Important)

**Official Parameter**: `sriovNetworkOperator.enabled`

**Current Template**: Missing (relies on SR-IOV device plugin only)

**Impact**: Cannot deploy full SR-IOV Network Operator capabilities (network policies, resource configs)

**Recipe Measurement Required**:
```yaml
type: Network
subtypes:
  - name: sriov
    data:
      operatorEnabled: "false"  # Currently using device plugin only
```

### 5.7 NIC Configuration Operator (Important)

**Official Parameters**:
- `nicConfigurationOperator.enabled` (deprecated but still available)
- `nic-configuration-operator-chart.operator.image`
- `nic-configuration-operator-chart.configDaemon.image`

**Current Template**: Missing all

**Impact**: Cannot use NIC Configuration Operator for NIC firmware and settings management

**Recipe Measurement Required**:
```yaml
type: Network
subtypes:
  - name: nic-config
    data:
      enabled: "false"
      operatorRepository: "ghcr.io/mellanox"
      operatorImage: "nic-configuration-operator"
      operatorTag: "v1.0.3"
```

### 5.8 OpenShift Driver ToolKit (Important for OpenShift)

**Official Parameter**: `operator.useDTK`

**Current Template**: Missing (defaults to true)

**Impact**: Cannot disable DTK on non-OpenShift or customize driver compilation

**Recipe Measurement Required**:
```yaml
type: OS
subtypes:
  - name: platform
    data:
      openshift: "false"
      useDTK: "false"
```

### 5.9 Upgrade CRDs (Important for upgrades)

**Official Parameter**: `upgradeCRDs`

**Current Template**: Missing (defaults to true)

**Impact**: CRDs always upgraded with helm hooks, may want manual control

**Recipe Measurement Required**:
```yaml
type: K8s
subtypes:
  - name: upgrade-policy
    data:
      upgradeCRDs: "true"
```

### 5.10 Container Runtime Socket (Important)

**Official Parameter**: `containerRuntime.socket` (already in template but hardcoded)

**Current Template**: Has parameter but value comes from static field

**Impact**: Limited - already supported but could be more flexible

**Template Enhancement**: Already present, extraction from recipe works correctly

---

## 6. Optional Gaps (P2) - Advanced Features

### 6.1 Maintenance Operator (Optional for automated maintenance)

**Official Parameters**: 15+ parameters in `maintenance-operator-chart`
- `operator.image` (repository, name, tag)
- `operator.admissionController.enable`
- `operator.admissionController.certificates`

**Current Template**: Missing all (operator not deployed)

**Impact**: No automated maintenance mode orchestration

### 6.2 Name Overrides (Optional)

**Official Parameters**:
- `operator.nameOverride`
- `operator.fullnameOverride`

**Current Template**: Missing

**Impact**: Cannot customize resource naming

### 6.3 Certificate Management (Optional for production)

**Official Parameters**: Custom certificate support for:
- Network Operator admission controller
- SR-IOV Network Operator admission controller and webhook
- Maintenance Operator admission controller

**Current Template**: Missing all, relies on cert-manager defaults

**Impact**: Cannot use custom PKI or external certificate management

### 6.4 Test Configuration (Optional)

**Official Parameter**: `test.pf` (physical function for testing)

**Current Template**: Missing

**Impact**: Cannot customize helm test behavior

### 6.5 Platform-Specific Settings (Optional)

**Official Parameter**: `platform.openshift` (in sub-charts)

**Current Template**: Missing

**Impact**: No OpenShift-specific optimizations

---

## 7. Summary of Gaps by Category

| Category | Total Available | Currently Supported | Gap Count | Priority |
|----------|----------------|---------------------|-----------|----------|
| **Security** | 2 | 0 (0%) | 2 | P0 Critical |
| **SR-IOV Configuration** | 15 | 1 (7%) | 14 | P0 Critical |
| **SR-IOV Images** | 8 | 0 (0%) | 8 | P0 Critical |
| **OFED Advanced** | 8 | 2 (25%) | 6 | P1 Important |
| **Resources & Scheduling** | 12 | 1 (8%) | 11 | P1 Important |
| **Admission Controllers** | 10 | 0 (0%) | 10 | P1 Important |
| **NFD Advanced** | 25 | 2 (8%) | 23 | P1 Important |
| **Maintenance Operator** | 20 | 0 (0%) | 20 | P1 Important |
| **NIC Configuration Operator** | 10 | 0 (0%) | 10 | P2 Optional |
| **Certificate Management** | 15 | 0 (0%) | 15 | P2 Optional |
| **Platform Settings** | 5 | 0 (0%) | 5 | P2 Optional |
| **Name Overrides** | 2 | 0 (0%) | 2 | P2 Optional |

**Total Identified Gaps**: 126 parameters  
**P0 Critical**: 24 parameters (security, SR-IOV resource configuration and images)  
**P1 Important**: 70 parameters (OFED advanced, resources, admission controllers, NFD, maintenance)  
**P2 Optional**: 32 parameters (NIC config operator, certificates, platform settings)

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
  
  - name: operator-resources
    data:
      cpuLimit: "500m"
      memoryLimit: "128Mi"
      cpuRequest: "5m"
      memoryRequest: "64Mi"
  
  - name: operator-scheduling
    data:
      tolerations: |
        - key: "node-role.kubernetes.io/master"
          operator: "Equal"
          value: ""
          effect: "NoSchedule"
      nodeSelector: {}
  
  - name: admission-control
    data:
      networkOperatorEnabled: "false"
      networkOperatorUseCertManager: "true"
      sriovOperatorEnabled: "false"
      sriovOperatorUseCertManager: "true"
  
  - name: cni-config
    data:
      binDirectory: "/opt/cni/bin"
  
  - name: maintenance
    data:
      enabled: "false"
      useRequestor: "false"
      requestorID: "nvidia.network.operator"
  
  - name: upgrade-policy
    data:
      upgradeCRDs: "true"
  
  - name: nfd-config
    data:
      enableNodeFeatureApi: "true"
      nodeFeatureAPIGate: "true"
      postDeleteCleanup: "false"
  
  - name: sriov-images
    data:
      operator: "nvcr.io/nvidia/mellanox/sriov-network-operator:network-operator-25.4.0"
      # ... other images
```

### 8.2 New Network Measurement Type

```yaml
type: Network
subtypes:
  - name: sriov-operator
    data:
      enabled: "false"
      resourcePrefix: "nvidia.com"
      deployConfig: "true"
      configDaemonNodeSelector: |
        beta.kubernetes.io/os: "linux"
  
  - name: ofed-init
    data:
      enabled: "true"
      repository: "ghcr.io/mellanox"
      image: "network-operator-init-container"
      version: "v0.0.3"
  
  - name: nic-config
    data:
      enabled: "false"
      operatorRepository: "ghcr.io/mellanox"
      operatorImage: "nic-configuration-operator"
      operatorTag: "v1.0.3"
```

### 8.3 Enhanced OS Measurement

```yaml
type: OS
subtypes:
  - name: platform
    data:
      openshift: "false"
      useDTK: "false"
```

---

## 9. Recommended Implementation Phases

### Phase 1: Critical Security & SR-IOV (P0) - Target: Q1 2026

**Goal**: Enable secure deployments and proper SR-IOV configuration

1. Add `imagePullSecrets` support
2. Add `sriov-network-operator.operator.resourcePrefix`
3. Add `sriov-network-operator.sriovOperatorConfig` parameters
4. Add `sriov-network-operator.images` configuration for all 8 images
5. Add `operator.ofedDriver.initContainer` configuration

**Recipe Changes**:
- Add `K8s/registry` subtype
- Add `Network/sriov-operator` subtype
- Add `K8s/sriov-images` subtype
- Add `Network/ofed-init` subtype

**Template Changes**:
- Add imagePullSecrets at top level
- Add sriov-network-operator configuration section
- Add OFED init container section

**Estimated Effort**: 2-3 days

### Phase 2: Resources and Scheduling (P1) - Target: Q1 2026

**Goal**: Production-ready with resource management

1. Add `operator.resources` (limits/requests)
2. Add `operator.tolerations`
3. Add `operator.nodeSelector`
4. Add `operator.affinity`

**Recipe Changes**:
- Add `K8s/operator-resources` subtype
- Add `K8s/operator-scheduling` subtype

**Template Changes**:
- Add resources section
- Add tolerations section
- Extend affinity support

**Estimated Effort**: 1-2 days

### Phase 3: Admission Controllers & Validation (P1) - Target: Q2 2026

**Goal**: Enable validation webhooks for safety

1. Add `operator.admissionController` configuration
2. Add `sriov-network-operator.operator.admissionControllers`
3. Add cert-manager integration options

**Recipe Changes**:
- Add `K8s/admission-control` subtype

**Template Changes**:
- Add admission controller sections for both operators
- Add certificate configuration options

**Estimated Effort**: 2-3 days

### Phase 4: Advanced Features (P1) - Target: Q2 2026

**Goal**: Support advanced deployment scenarios

1. Add CNI bin directory configuration
2. Add maintenance operator integration
3. Add NFD advanced configuration
4. Add SR-IOV Network Operator enablement
5. Add upgrade CRDs configuration

**Recipe Changes**:
- Add `K8s/cni-config` subtype
- Add `K8s/maintenance` subtype
- Add `K8s/nfd-config` subtype
- Extend `Network/sriov` subtype
- Add `K8s/upgrade-policy` subtype

**Template Changes**:
- Add CNI bin directory
- Add maintenance operator configuration
- Extend NFD configuration
- Add SR-IOV operator enablement logic
- Add upgradeCRDs parameter

**Estimated Effort**: 3-4 days

### Phase 5: Optional Operators (P2) - Target: Q3 2026

**Goal**: Support niche use cases

1. Add NIC Configuration Operator
2. Add Maintenance Operator full deployment
3. Add custom certificate management
4. Add name overrides
5. Add platform-specific settings

**Recipe Changes**:
- Add `Network/nic-config` subtype
- Add `OS/platform` subtype
- Add certificate configuration options

**Template Changes**:
- Add NIC Configuration Operator section
- Add Maintenance Operator section
- Add certificate sections
- Add name override parameters
- Add platform settings

**Estimated Effort**: 4-5 days

---

## 10. Validation Plan

### 10.1 Test Scenarios

For each implemented phase, validate against these scenarios:

1. **Basic Deployment** (Baseline)
   - Fresh cluster, no pre-existing OFED
   - RDMA device plugin only
   - Default configurations

2. **Private Registry** (Phase 1)
   - NGC registry with authentication
   - Custom private registry
   - SR-IOV images from private registry

3. **SR-IOV Configuration** (Phase 1)
   - Custom resource prefix
   - SR-IOV operator config with node selector
   - Multiple SR-IOV network device plugins

4. **OFED Deployment** (Phase 1)
   - Deploy OFED driver container
   - Custom OFED version
   - OFED init container configuration

5. **Resource Constrained** (Phase 2)
   - Resource limits enforced
   - Scheduling with tolerations and affinity

6. **Admission Controllers** (Phase 3)
   - Network operator webhook validation
   - SR-IOV operator webhook validation
   - Custom certificates vs cert-manager

7. **Advanced Networking** (Phase 4)
   - Multus with SR-IOV
   - nvIPAM with Whereabouts
   - Multiple network attachments

8. **Maintenance Mode** (Phase 4)
   - Maintenance operator integration
   - Automated node cordoning during upgrades

9. **NIC Configuration** (Phase 5)
   - NIC Configuration Operator deployment
   - Firmware management
   - NIC settings automation

10. **OpenShift** (Phase 5)
    - Driver Toolkit usage
    - OpenShift-specific configurations

### 10.2 Acceptance Criteria

Each scenario must:
- ✅ Generate syntactically valid Helm values
- ✅ Deploy successfully via generated install script
- ✅ Pass Network Operator validation
- ✅ NicClusterPolicy reaches ready state
- ✅ RDMA/SR-IOV devices correctly configured
- ✅ Handle error cases gracefully (missing measurements)

---

## 11. Comparison with GPU Operator

### 11.1 Complexity Comparison

| Aspect | GPU Operator | Network Operator |
|--------|--------------|------------------|
| **Total Parameters** | 120+ | 150+ |
| **Current Coverage** | ~15 (12.5%) | ~13 (8.7%) |
| **Critical Gaps** | 12 | 24 |
| **Important Gaps** | 78 | 70 |
| **Legacy Usage** | 40+ parameters, 20+ scenarios | 3 parameters, 3 scenarios |
| **Sub-Charts** | 1 (NFD) | 3 (NFD, SR-IOV, Maintenance) |
| **Image Registry Count** | ~10 components | ~15 components (including SR-IOV) |
| **Admission Controllers** | 1 (optional) | 3 (optional, per operator) |

### 11.2 Implementation Effort

| Operator | Phase 1 | Phase 2 | Phase 3 | Phase 4 | Phase 5 | Total |
|----------|---------|---------|---------|---------|---------|-------|
| **GPU Operator** | 2-3 days | 1 day | 3-4 days | 4-5 days | 5-7 days | 15-20 days |
| **Network Operator** | 2-3 days | 1-2 days | 2-3 days | 3-4 days | 4-5 days | 12-17 days |

Network Operator implementation is slightly less effort due to:
- Simpler legacy playbook usage (fewer scenarios to support)
- More modular sub-chart structure
- Less complex component interactions

However, Network Operator has more critical gaps (24 vs 12) due to SR-IOV configuration requirements.

---

## 12. Migration Path for Existing Users

### 12.1 For Legacy Playbook Users

Users migrating from Ansible playbooks should:

1. **Phase 1**: Basic migration possible with security and SR-IOV
   - Map existing `enable_rdma` and `deploy_ofed` to recipe measurements
   - Generate bundles with registry authentication
   - Validate RDMA functionality

2. **Phase 3**: Add admission controllers for safety
   - Enable validation webhooks
   - Test NicClusterPolicy changes

3. **Phase 4**: Full operational parity
   - Configure resources and scheduling
   - Enable maintenance operator integration

### 12.2 For New Users

New users should:

1. Start with basic recipes (current template)
2. Add security configurations (Phase 1)
3. Enable SR-IOV if needed (Phase 1)
4. Progressively enable advanced features as needed

### 12.3 For GPU Operator Users

Users familiar with GPU Operator bundler should note:

- Similar structure and patterns
- Network Operator has more sub-components (SR-IOV, NIC Config, Maintenance)
- Security gaps are similar (imagePullSecrets)
- Resource management patterns identical
- Admission controller setup more complex (3 operators vs 1)

---

## 13. Conclusion

The current Network Operator bundler provides a **solid foundation for basic RDMA deployments** but has significant gaps for **production SR-IOV and multi-operator scenarios**. The identified gaps span:

- **24 critical parameters** needed for secure deployments and SR-IOV resource configuration
- **70 important operational parameters** for resources, admission controllers, and advanced features
- **32 optional parameters** for NIC configuration and maintenance operators

**Key Differences from GPU Operator**:
- **More sub-components**: 3 operators (Network, SR-IOV, Maintenance) vs 1 (GPU)
- **Less legacy usage**: Only 3 parameters in playbooks vs 40+ for GPU Operator
- **Simpler migration**: Fewer deployment scenarios to support
- **More critical gaps**: 24 vs 12 for GPU Operator due to SR-IOV complexity

**Recommendation**: Implement in phases, prioritizing P0 (security and SR-IOV) and P1 (resources, admission controllers) gaps to achieve production readiness within 4-5 months, slightly faster than GPU Operator due to simpler legacy requirements.

**Next Steps**:
1. Review and approve this gap analysis
2. Create GitHub issues for each phase
3. Design recipe measurement schema enhancements
4. Implement Phase 1 (security and SR-IOV) as highest priority
5. Coordinate with GPU Operator implementation for shared patterns (registry, resources, etc.)

---

## Appendix A: Key Helm Values Structure

### A.1 Top-Level Structure
```yaml
# Feature flags
nfd:
  enabled: true
  deployNodeFeatureRules: true
upgradeCRDs: true
sriovNetworkOperator:
  enabled: false
nicConfigurationOperator:
  enabled: false
maintenanceOperator:
  enabled: false

# Sub-charts
node-feature-discovery: { ... }
sriov-network-operator: { ... }
nic-configuration-operator-chart: { ... }
maintenance-operator-chart: { ... }

# General operator config
operator: { ... }
imagePullSecrets: []
```

### A.2 SR-IOV Network Operator Sub-Chart
```yaml
sriov-network-operator:
  operator:
    resourcePrefix: "nvidia.com"
    admissionControllers:
      enabled: false
      certificates: { ... }
  images:
    operator: "..."
    sriovConfigDaemon: "..."
    sriovCni: "..."
    ibSriovCni: "..."
    ovsCni: "..."
    sriovDevicePlugin: "..."
    resourcesInjector: "..."
    webhook: "..."
  sriovOperatorConfig:
    deploy: true
    configDaemonNodeSelector: { ... }
```

### A.3 Operator General Config
```yaml
operator:
  resources: { ... }
  tolerations: [ ... ]
  nodeSelector: { ... }
  affinity: { ... }
  repository: "nvcr.io/nvidia/cloud-native"
  image: "network-operator"
  cniBinDirectory: "/opt/cni/bin"
  maintenanceOperator:
    useRequestor: false
    requestorID: "nvidia.network.operator"
  useDTK: true
  admissionController:
    enabled: false
    useCertManager: true
  ofedDriver:
    initContainer:
      enable: true
      repository: "ghcr.io/mellanox"
      image: "network-operator-init-container"
      version: "v0.0.3"
```

---

## Appendix B: Current vs Target Template

### B.1 Current Template (13 Parameters)
```yaml
operator:
  tag: {{ .NetworkOperatorVersion }}
deployCR: true
ofedDriver:
  deploy: {{ .DeployOFED }}
  version: {{ .OFEDVersion }}
rdmaSharedDevicePlugin:
  deploy: {{ .EnableRDMA }}
  config:
    resources:
      - name: rdma_shared_device
        rdmaHcaMax: 1000
sriovDevicePlugin:
  deploy: {{ .EnableSRIOV }}
nvIPAM:
  deploy: {{ .EnableIPAM }}
  config:
    enableWebhook: false
secondaryNetwork:
  deploy: true
  multus:
    deploy: {{ .EnableMultus }}
  ipamPlugin:
    deploy: {{ .EnableWhereabouts }}
nicFeatureDiscovery:
  deploy: true
nfd:
  enabled: true
  deployNodeFeatureRules: true
containerRuntime:
  socket: {{ .ContainerRuntimeSocket }}
```

### B.2 Target Template After Phase 4 (80+ Parameters)
```yaml
# Security
{{- if .RegistryPullSecret }}
imagePullSecrets:
  - name: {{ .RegistryPullSecret }}
{{- end }}

# Operator Config
operator:
  tag: {{ .NetworkOperatorVersion }}
  {{- if .OperatorResources }}
  resources:
    limits:
      cpu: {{ .OperatorCPULimit }}
      memory: {{ .OperatorMemoryLimit }}
    requests:
      cpu: {{ .OperatorCPURequest }}
      memory: {{ .OperatorMemoryRequest }}
  {{- end }}
  {{- if .OperatorTolerations }}
  tolerations:
    {{ .OperatorTolerations | nindent 4 }}
  {{- end }}
  cniBinDirectory: {{ .CNIBinDirectory | default "/opt/cni/bin" }}
  maintenanceOperator:
    useRequestor: {{ .MaintenanceUseRequestor | default false }}
    requestorID: {{ .MaintenanceRequestorID | default "nvidia.network.operator" }}
  admissionController:
    enabled: {{ .NetworkOperatorAdmissionEnabled | default false }}
    useCertManager: {{ .NetworkOperatorUseCertManager | default true }}
  ofedDriver:
    initContainer:
      enable: {{ .OFEDInitEnabled | default true }}
      repository: {{ .OFEDInitRepository | default "ghcr.io/mellanox" }}
      image: {{ .OFEDInitImage | default "network-operator-init-container" }}
      version: {{ .OFEDInitVersion | default "v0.0.3" }}

# Upgrade Policy
upgradeCRDs: {{ .UpgradeCRDs | default true }}

# Deploy CR
deployCR: true

# OFED Driver
ofedDriver:
  deploy: {{ .DeployOFED }}
  version: {{ .OFEDVersion }}

# RDMA Device Plugin
rdmaSharedDevicePlugin:
  deploy: {{ .EnableRDMA }}
  {{- if .EnableRDMA }}
  config:
    resources:
      - name: rdma_shared_device
        rdmaHcaMax: {{ .RDMAHcaMax | default 1000 }}
  {{- end }}

# SR-IOV Device Plugin
sriovDevicePlugin:
  deploy: {{ .EnableSRIOV }}

# nvIPAM
nvIPAM:
  deploy: {{ .EnableIPAM }}
  {{- if .EnableIPAM }}
  config:
    enableWebhook: {{ .IPAMEnableWebhook | default false }}
  {{- end }}

# Secondary Network
secondaryNetwork:
  deploy: true
  multus:
    deploy: {{ .EnableMultus }}
  ipamPlugin:
    deploy: {{ .EnableWhereabouts }}

# NIC Feature Discovery
nicFeatureDiscovery:
  deploy: true

# Node Feature Discovery
nfd:
  enabled: {{ .NFDEnabled | default true }}
  deployNodeFeatureRules: {{ .NFDDeployNodeFeatureRules | default true }}

# Node Feature Discovery Sub-Chart
node-feature-discovery:
  enableNodeFeatureApi: {{ .NFDEnableNodeFeatureApi | default true }}
  featureGates:
    NodeFeatureAPI: {{ .NFDNodeFeatureAPIGate | default true }}
  postDeleteCleanup: {{ .NFDPostDeleteCleanup | default false }}

# SR-IOV Network Operator Sub-Chart
{{- if .SRIOVOperatorEnabled }}
sriovNetworkOperator:
  enabled: {{ .SRIOVOperatorEnabled }}
{{- end }}

sriov-network-operator:
  operator:
    resourcePrefix: {{ .SRIOVResourcePrefix | default "nvidia.com" }}
    admissionControllers:
      enabled: {{ .SRIOVAdmissionEnabled | default false }}
  {{- if .SRIOVImages }}
  images:
    operator: {{ .SRIOVOperatorImage }}
    sriovConfigDaemon: {{ .SRIOVConfigDaemonImage }}
    sriovCni: {{ .SRIOVCNIImage }}
    ibSriovCni: {{ .IBSRIOVCNIImage }}
    ovsCni: {{ .OVSCNIImage }}
    sriovDevicePlugin: {{ .SRIOVDevicePluginImage }}
    resourcesInjector: {{ .SRIOVResourcesInjectorImage }}
    webhook: {{ .SRIOVWebhookImage }}
  {{- end }}
  sriovOperatorConfig:
    deploy: {{ .SRIOVDeployConfig | default true }}
    {{- if .SRIOVConfigDaemonNodeSelector }}
    configDaemonNodeSelector:
      {{ .SRIOVConfigDaemonNodeSelector | nindent 6 }}
    {{- end }}

# Maintenance Operator
{{- if .MaintenanceOperatorEnabled }}
maintenanceOperator:
  enabled: {{ .MaintenanceOperatorEnabled }}
{{- end }}

# Container Runtime
containerRuntime:
  socket: {{ .ContainerRuntimeSocket }}

# Node Affinity
{{- if .CustomLabels }}
nodeAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
    nodeSelectorTerms:
    - matchExpressions:
      {{- range $key, $value := .CustomLabels }}
      - key: {{ $key }}
        operator: In
        values:
        - {{ $value }}
      {{- end }}
{{- end }}

# Namespace
{{- if .Namespace }}
namespace: {{ .Namespace }}
{{- end }}
```

---

**Document Version**: 1.0  
**Last Updated**: December 29, 2025  
**Author**: Cloud Native Stack Engineering  
**Review Status**: Draft - Pending Approval
