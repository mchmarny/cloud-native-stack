# CNS End-to-End Demo

Four-stage workflow: **Snapshot → Recipe → Validate → Bundle**

![demo overview](images/e2e.png)

## Install

```shell
curl -sfL https://raw.githubusercontent.com/mchmarny/cloud-native-stack/main/install | bash -s --
```

Quick test: 

```shell
cnsctl -v
```

## 1. Snapshot 

Deploy a Kubernetes Job to capture GPU node configuration:

```shell
cnsctl snapshot \
    --deploy-agent \
    --namespace gpu-operator \
    --image ghcr.io/mchmarny/cns:latest \
    --node-selector nodeGroup=customer-gpu \
    --cleanup
```

> **Note**: By default, all taints are tolerated. Use `--toleration key=value:effect` to override.

Output:

```shell
deploying agent: namespace=gpu-operator
job completed successfully
snapshot saved to ConfigMap: uri=cm://gpu-operator/cns-snapshot
```

View the snapshot:

```shell
kubectl -n gpu-operator get cm cns-snapshot -o jsonpath='{.data.snapshot\.yaml}' | yq .
```

## 2. Recipe

Generate optimized configuration from the snapshot:

```shell
cnsctl recipe \
    --snapshot cm://gpu-operator/cns-snapshot \
    --intent training \
    --output recipe.yaml
```

View component versions:

```shell
yq eval '.componentRefs[] | {"name": .name, "version": .version, "type": .type}' recipe.yaml
```

**Alternative**: Generate recipe directly from parameters (no snapshot needed):

```shell
cnsctl recipe \
    --service eks \
    --accelerator h100 \
    --os ubuntu \
    --intent training | yq
```

Or query the [CNS API](https://cns.dgxc.io) directly:

```shell
curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=h100&intent=training" | jq .
```

![data flow](images/data.png)

## 3. Validate 

Validate a target cluster against the recipe: 

```shell
cnsctl validate \
    --recipe recipe.yaml \
    --snapshot cm://gpu-operator/cns-snapshot
```

Outputs: 

```
loading recipe: uri=recipe.yaml
loading snapshot: uri=cm://gpu-operator/cns-snapshot
validating constraints: recipe=recipe.yaml snapshot=cm://gpu-operator/cns-snapshot constraints=4
```

Save results to a file:

```shell
cnsctl validate \
    --recipe recipe.yaml \
    --snapshot cm://gpu-operator/cns-snapshot \
    --output validation-results.yaml
```

Output: 

```yaml
kind: ValidationResult
apiVersion: cns.nvidia.com/v1alpha1
metadata:
  timestamp: "2026-01-12T21:53:03Z"
  version: 0.16.2
recipeSource: recipe.yaml
snapshotSource: cm://gpu-operator/cns-snapshot
summary:
  passed: 4
  failed: 0
  skipped: 0
  total: 4
  status: pass
  duration: 20.958µs
results:
  - name: K8s.server.version
    expected: '>= 1.32.4'
    actual: v1.33.5-eks-3025e55
    status: passed
  - name: OS.release.ID
    expected: ubuntu
    actual: ubuntu
    status: passed
  - name: OS.release.VERSION_ID
    expected: "24.04"
    actual: "24.04"
    status: passed
  - name: OS.sysctl./proc/sys/kernel/osrelease
    expected: '>= 6.8'
    actual: 6.8.0-1028-aws
    status: passed
```

## 4. Bundle

Generate deployment artifacts with node scheduling:

```shell
cnsctl bundle \
    --recipe recipe.yaml \
    --output ./bundles \
    --system-node-selector nodeGroup=system-pool \
    --accelerated-node-selector nodeGroup=customer-gpu \
    --accelerated-node-toleration nvidia.com/gpu=present:NoSchedule \
    --deployer argocd
```

Output:

```shell
bundle generation completed: success=4 errors=0 summary=Generated 24 files (33.2 KB)
```

Similarly, bundles using API with full options: 

```shell
# Query recipe API and pipe response to bundle API
curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=h100&intent=training" | \
  curl -X POST "https://cns.dgxc.io/v1/bundle?deployer=argocd" \
    -H "Content-Type: application/json" -d @- -o bundles.zip

# List generated bundles
unzip bundles.zip -d ./bundles
```

Bundles could be wrapped in: 
* ArgoCD App of Apps (combining N bundles)
* Signed container image (self-extracting in cluster)

GPU Operator README:

```shell
grip --browser --quiet ./bundles/gpu-operator/README.md
```
