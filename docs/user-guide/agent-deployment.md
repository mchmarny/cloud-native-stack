# Agent Deployment

Deploy Eidos as a Kubernetes Job to automatically capture cluster configuration snapshots.

## Overview

The Eidos agent runs as a Kubernetes Job that:
- Captures system configuration from GPU nodes
- Stores snapshot directly in Kubernetes ConfigMap (`eidos-snapshot` in `gpu-operator` namespace)
- No volumes required - writes via Kubernetes API
- Useful for auditing, troubleshooting, and multi-cluster management

**Agent capabilities:**
- ✅ Step 1: Snapshot capture with ConfigMap output
- ❌ Step 2: Recipe generation (use CLI or API)
- ❌ Step 3: Bundle generation (use CLI)

**ConfigMap Output:**
The agent uses `cm://namespace/name` URI scheme to write snapshots directly to Kubernetes ConfigMaps:
```bash
eidos snapshot --output cm://gpu-operator/eidos-snapshot
```

This creates:
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
  snapshot.yaml: |  # Complete snapshot YAML
    apiVersion: snapshot.dgxc.io/v1
    kind: Snapshot
    measurements: [...]
  format: yaml
  timestamp: "2026-01-03T10:30:00Z"
```

## Prerequisites

- Kubernetes cluster with GPU nodes
- `kubectl` configured with cluster access
- GPU Operator installed (agent runs in `gpu-operator` namespace)
- Cluster admin permissions (for RBAC setup)

## Quick Start

### 1. Deploy RBAC and ServiceAccount

The agent requires permissions to read Kubernetes resources and write to ConfigMaps:

```shell
kubectl apply -f https://raw.githubusercontent.com/nvidia/cloud-native-stack/main/deployments/eidos-agent/1-deps.yaml
```

**What this creates:**
- **Namespace**: `gpu-operator` (if not exists)
- **ServiceAccount**: `eidos` in `gpu-operator` namespace
- **Role**: Permissions to create/update ConfigMaps in `gpu-operator` namespace
- **RoleBinding**: Binds Role to ServiceAccount in `gpu-operator` namespace
- **ClusterRole**: Permissions to read nodes, pods, ClusterPolicy
- **ClusterRoleBinding**: Binds ClusterRole to ServiceAccount

### 2. Deploy the Agent Job

```shell
kubectl apply -f https://raw.githubusercontent.com/nvidia/cloud-native-stack/main/deployments/eidos-agent/2-job.yaml
```

**What this creates:**
- **Job**: `eidos` in the `gpu-operator` namespace
- Job runs `eidos snapshot --output cm://gpu-operator/eidos-snapshot`
- Snapshot is written directly to ConfigMap via Kubernetes API

### 3. View Snapshot Output

Check job status:
```shell
kubectl get jobs -n gpu-operator
```

Check job logs (for errors/debugging):
```shell
kubectl logs -n gpu-operator job/eidos
```

Retrieve snapshot from ConfigMap:
```shell
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}'
```

Save snapshot to file:
```shell
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > snapshot.yaml
```

## Customization

Before deploying, you may need to customize the Job manifest for your environment.

### Download and Edit Manifest

```shell
# Download job manifest
curl -O https://raw.githubusercontent.com/nvidia/cloud-native-stack/main/deployments/eidos-agent/2-job.yaml

# Edit with your preferred editor
vim 2-job.yaml
```

### Node Selection

Target specific GPU nodes using `nodeSelector`:

```yaml
spec:
  template:
    spec:
      nodeSelector:
        nvidia.com/gpu.present: "true"        # Any GPU node
        # nodeGroup: your-gpu-node-group      # Specific node group
        # instance-type: p4d.24xlarge         # Specific instance type
```

**Common node selectors:**

| Selector | Purpose |
|----------|---------|
| `nvidia.com/gpu.present: "true"` | Any node with GPU |
| `nodeGroup: gpu-nodes` | Specific node pool (EKS/GKE) |
| `node.kubernetes.io/instance-type: p4d.24xlarge` | AWS instance type |
| `cloud.google.com/gke-accelerator: nvidia-tesla-h100` | GKE GPU type |

### Tolerations

If your GPU nodes have taints, add tolerations:

```yaml
spec:
  template:
    spec:
      tolerations:
        - key: nvidia.com/gpu
          operator: Exists
          effect: NoSchedule
        # Add your custom taints:
        - key: dedicated
          operator: Equal
          value: gpu
          effect: NoSchedule
```

**Common tolerations:**

| Taint Key | Effect | Purpose |
|-----------|--------|---------|
| `nvidia.com/gpu` | NoSchedule | GPU Operator default |
| `dedicated` | NoSchedule | Dedicated GPU nodes |
| `workload` | NoSchedule | Workload-specific nodes |

### Image Version

Use a specific version instead of `latest`:

```yaml
spec:
  template:
    spec:
      containers:
        - name: eidos
          image: ghcr.io/nvidia/eidos:v0.8.0  # Pin to version
```

**Finding versions:**
- [GitHub Releases](https://github.com/nvidia/cloud-native-stack/releases)
- Container registry: [ghcr.io/nvidia/eidos](https://github.com/nvidia/cloud-native-stack/pkgs/container/eidos)

### Resource Limits

Adjust CPU and memory limits:

```yaml
spec:
  template:
    spec:
      containers:
        - name: eidos
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
```

### Custom Output Format

Change output format via command arguments:

```yaml
spec:
  template:
    spec:
      containers:
        - name: eidos
          args:
            - snapshot
            - --format
            - json  # Change to: yaml, json, table
```

## Deployment Examples

### Example 1: AWS EKS with GPU Node Group

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: eidos
  namespace: gpu-operator
spec:
  template:
    spec:
      serviceAccountName: eidos
      restartPolicy: Never
      nodeSelector:
        nodeGroup: gpu-nodes  # Your EKS node group
      tolerations:
        - key: nvidia.com/gpu
          operator: Exists
          effect: NoSchedule
      containers:
        - name: eidos
          image: ghcr.io/nvidia/eidos:latest
          command: ["eidos"]
          args: ["snapshot", "--output", "cm://gpu-operator/eidos-snapshot"]
```

### Example 2: GKE with H100 GPUs

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: eidos
  namespace: gpu-operator
spec:
  template:
    spec:
      serviceAccountName: eidos
      restartPolicy: Never
      nodeSelector:
        cloud.google.com/gke-accelerator: nvidia-tesla-h100
      containers:
        - name: eidos
          image: ghcr.io/nvidia/eidos:latest
          command: ["eidos"]
          args: ["snapshot", "--output", "cm://gpu-operator/eidos-snapshot"]
```

### Example 3: Periodic Snapshots (CronJob)

Automatic snapshots for drift detection:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: eidos-snapshot
  namespace: gpu-operator
spec:
  schedule: "0 */6 * * *"  # Every 6 hours
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: eidos
          restartPolicy: Never
          nodeSelector:
            nvidia.com/gpu.present: "true"
          containers:
            - name: eidos
              image: ghcr.io/nvidia/eidos:latest
              command: ["eidos"]
              args: ["snapshot", "--output", "cm://gpu-operator/eidos-snapshot"]
```

Retrieve historical snapshots:
```shell
# List completed jobs
kubectl get jobs -n gpu-operator -l job-name=eidos-snapshot

# Get latest snapshot from ConfigMap (updated by most recent job)
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > latest-snapshot.yaml

# Check ConfigMap update timestamp
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.metadata.creationTimestamp}'

# View job logs for debugging (if needed)
kubectl logs -n gpu-operator job/eidos-snapshot-28405680
```

**Note**: The ConfigMap `eidos-snapshot` is updated by each CronJob run. For historical tracking, save snapshots to external storage (S3, Git, etc.) using a post-job step.

## Post-Deployment

### Monitor Job Status

```shell
# Check job status
kubectl get jobs -n gpu-operator

# Describe job for events
kubectl describe job eidos -n gpu-operator

# Check pod status
kubectl get pods -n gpu-operator -l job-name=eidos
```

### Retrieve Snapshot

```shell
# View snapshot from ConfigMap
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}'

# Save to file
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > snapshot-$(date +%Y%m%d).yaml

# View job logs (for debugging)
kubectl logs -n gpu-operator job/eidos

# Check ConfigMap metadata
kubectl get configmap eidos-snapshot -n gpu-operator -o yaml
```

### Generate Recipe from Snapshot

```shell
# Option 1: Use ConfigMap directly (no file needed)
eidos recipe --snapshot cm://gpu-operator/eidos-snapshot --intent training --output recipe.yaml

# Option 2: Save snapshot to file first
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > snapshot.yaml
eidos recipe --snapshot snapshot.yaml --intent training --output recipe.yaml

# Generate bundle
eidos bundle --recipe recipe.yaml --output ./bundles
```

### Clean Up

```shell
# Delete job
kubectl delete job eidos -n gpu-operator

# Delete RBAC (if no longer needed)
kubectl delete -f https://raw.githubusercontent.com/mchmarny/cloud-native-stack/main/deployments/eidos-agent/1-deps.yaml
```

## Integration Patterns

### 1. CI/CD Pipeline

```yaml
# GitHub Actions example
- name: Deploy agent to capture snapshot
  run: |
    kubectl apply -f deployments/eidos-agent/1-deps.yaml
    kubectl apply -f deployments/eidos-agent/2-job.yaml
    kubectl wait --for=condition=complete --timeout=300s job/eidos -n gpu-operator
    
- name: Generate recipe from ConfigMap
  run: |
    # Option 1: Use ConfigMap directly (no file needed)
    eidos recipe -f cm://gpu-operator/eidos-snapshot -i training -o recipe.yaml
    
    # Option 2: Write recipe to ConfigMap as well
    eidos recipe -f cm://gpu-operator/eidos-snapshot -i training -o cm://gpu-operator/eidos-recipe
    
    # Option 3: Export snapshot to file for archival
    kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > snapshot.yaml
    
- name: Generate bundle
  run: |
    eidos bundle -f recipe.yaml -b gpu-operator -o ./bundles
    
- name: Upload artifacts
  uses: actions/upload-artifact@v3
  with:
    name: cluster-config
    path: |
      snapshot.yaml
      recipe.yaml
      bundles/
```

### 2. Multi-Cluster Auditing

```shell
#!/bin/bash
# Capture snapshots from multiple clusters

clusters=("prod-us-east" "prod-eu-west" "staging")

for cluster in "${clusters[@]}"; do
  echo "Capturing snapshot from $cluster..."
  
  # Switch context
  kubectl config use-context $cluster
  
  # Deploy agent
  kubectl apply -f deployments/eidos-agent/2-job.yaml
  
  # Wait for completion
  kubectl wait --for=condition=complete --timeout=300s job/eidos -n gpu-operator
  
  # Save snapshot from ConfigMap
  kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > snapshot-${cluster}.yaml
  
  # Clean up
  kubectl delete job eidos -n gpu-operator
done
```

### 3. Drift Detection

```shell
#!/bin/bash
# Compare current snapshot with baseline

# Baseline (first snapshot)
kubectl apply -f deployments/eidos-agent/2-job.yaml
kubectl wait --for=condition=complete job/eidos -n gpu-operator
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > baseline.yaml

# Current (later snapshot)
kubectl apply -f deployments/eidos-agent/2-job.yaml
kubectl wait --for=condition=complete job/eidos -n gpu-operator
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > current.yaml

# Compare
diff baseline.yaml current.yaml || echo "Configuration drift detected!"
```

## Troubleshooting

### Job Fails to Start

Check RBAC permissions:
```shell
kubectl auth can-i get nodes --as=system:serviceaccount:gpu-operator:eidos
kubectl auth can-i get pods --as=system:serviceaccount:gpu-operator:eidos
```

### Job Pending

Check node selectors and tolerations:
```shell
# View pod events
kubectl describe pod -n gpu-operator -l job-name=eidos

# Check node labels
kubectl get nodes --show-labels

# Check node taints
kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints
```

### Job Completes but No Output

Check ConfigMap and container logs:
```shell
# Check if ConfigMap was created
kubectl get configmap eidos-snapshot -n gpu-operator

# View ConfigMap contents
kubectl get configmap eidos-snapshot -n gpu-operator -o yaml

# View pod logs for errors
kubectl logs -n gpu-operator -l job-name=eidos

# Check for previous pod errors
kubectl logs -n gpu-operator -l job-name=eidos --previous
```

### Permission Denied

Ensure RBAC is correctly deployed:
```shell
# Verify ClusterRole
kubectl get clusterrole eidos

# Verify ClusterRoleBinding
kubectl get clusterrolebinding eidos

# Verify ServiceAccount
kubectl get serviceaccount eidos -n gpu-operator
```

### Image Pull Errors

Check image access:
```shell
# Describe pod
kubectl describe pod -n gpu-operator -l job-name=eidos

# For private registries, create image pull secret:
kubectl create secret docker-registry regcred \
  --docker-server=ghcr.io \
  --docker-username=<your-username> \
  --docker-password=<your-pat> \
  -n gpu-operator

# Add to job spec:
# imagePullSecrets:
#   - name: regcred
```

## Security Considerations

### Least Privilege

The agent only requires read permissions:
- Nodes
- Pods
- ClusterPolicy CRDs

### Network Policies

Restrict agent network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: eidos-agent
  namespace: gpu-operator
spec:
  podSelector:
    matchLabels:
      job-name: eidos
  policyTypes:
    - Egress
  egress:
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443  # Kubernetes API only
```

### Pod Security Standards

The agent runs as non-root:

```yaml
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
        fsGroup: 65532
      containers:
        - name: eidos
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop: ["ALL"]
```

## See Also

- [CLI Reference](cli-reference.md) - Eidos CLI commands
- [Installation Guide](installation.md) - Install CLI locally
- [API Reference](../integration/api-reference.md) - API server deployment
- [RBAC Manifest](../../deployments/eidos-agent/1-deps.yaml) - Full RBAC configuration
- [Job Manifest](../../deployments/eidos-agent/2-job.yaml) - Full Job configuration
