# Eidos End-to-End Demo

Three-stage workflow: **Snapshot → Recipe → Bundle**

![demo overview](images/e2e.png)

## Install

```shell
curl -sfL https://raw.githubusercontent.com/mchmarny/cloud-native-stack/main/install | bash -s --
```

Quick test: 

```shell
eidos -v
```

## 1. Snapshot

Deploy a Kubernetes Job to capture GPU node configuration:

```shell
eidos snapshot \
    --deploy-agent \
    --namespace gpu-operator \
    --image ghcr.io/nvidia/eidos:latest \
    --node-selector nodeGroup=customer-gpu \
    --cleanup
```

> **Note**: By default, all taints are tolerated. Use `--toleration key=value:effect` to override.

> **Fork Users**: If you are using a fork of this repository, specify the correct container image using the `--image` flag. The default image (`ghcr.io/nvidia/eidos:latest`) points to the upstream NVIDIA registry. For forks, use your own registry (e.g., `--image ghcr.io/YOUR-ORG/eidos:latest`).

Output:

```shell
deploying agent: namespace=gpu-operator
job completed successfully
snapshot saved to ConfigMap: uri=cm://gpu-operator/eidos-snapshot
```

View the snapshot:

```shell
kubectl -n gpu-operator get cm eidos-snapshot -o jsonpath='{.data.snapshot\.yaml}' | yq .
```

## 2. Recipe

Generate optimized configuration from the snapshot:

```shell
eidos recipe \
    --snapshot cm://gpu-operator/eidos-snapshot \
    --intent training \
    --output recipe.yaml
```

View component versions:

```shell
yq eval '.measurements[]
  | {"type": .type,
     "subtypes": (.subtypes | map({"subtype": .subtype, "count": (.data | length)}))
    }' recipe.yaml
```

**Alternative**: Generate recipe directly from parameters (no snapshot needed):

```shell
eidos recipe --service eks --accelerator h100 --intent training | yq
```

Or query the [CNS API](https://cns.dgxc.io) directly:

```shell
curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=h100&intent=training" | jq .
```

![data flow](images/data.png)

## 3. Bundle

Generate deployment artifacts with node scheduling:

```shell
eidos bundle \
    --recipe recipe.yaml \
    --output ./bundles \
    --system-node-selector nodeGroup=system-pool \
    --accelerated-node-selector nodeGroup=customer-gpu \
    --accelerated-node-toleration nvidia.com/gpu=present:NoSchedule
```

Output:

```shell
bundle generation completed: success=4 errors=0 summary=Generated 24 files (33.2 KB)
```

Similarly, bundles using API: 

```shell
# Query recipe API and pipe response to bundle API
curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=gb200&intent=training&os=ubuntu" | \
  curl -X POST "https://cns.dgxc.io/v1/bundle" -H "Content-Type: application/json" -d @- \
    -o bundles.zip

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
