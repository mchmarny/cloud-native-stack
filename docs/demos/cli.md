# eidos demo

> Demo from fork `mchmarny/cloud-native-stack` vs NV CNS repo

## Install

```shell
curl -sfL https://raw.githubusercontent.com/mchmarny/cloud-native-stack/refs/heads/main/install | bash -s --
```

## Snapshot 

Supports `kubectl` [deployment](../../deployments/eidos-agent) as well as CLI:

``` shell
eidos snapshot \
    --deploy-agent \
    --namespace gpu-operator \
    --image ghcr.io/mchmarny/eidos:latest \
    --toleration dedicated=user-workload:NoSchedule \
    --toleration dedicated=user-workload:NoExecute \
    --node-selector nodeGroup=customer-gpu \
    --cleanup-rbac
```

Outputs: 

```json
deploying agent: namespace=gpu-operator
agent deployed successfully
waiting for Job completion: job=eidos timeout=5m0s
job completed successfully
snapshot saved to ConfigMap: uri=cm://gpu-operator/eidos-snapshot
```

What it creates: 

```shell
kubectl -n gpu-operator get cm eidos-snapshot -o yaml 
```

## Recipe

```shell
eidos recipe \
    --snapshot cm://gpu-operator/eidos-snapshot \
    --intent training \
    --output recipe.yaml
```

Outputs: 

```json
loading snapshot from: uri=cm://gpu-operator/eidos-snapshot
recipe generation completed: output=recipe.yaml
```

Review the recipe:

```shell
cat recipe.yaml
```

## Bundle

```shell
eidos bundle \
  --recipe recipe.yaml \
  --bundlers gpu-operator \
  --output ./bundles
```

Outputs: 

```json
generating bundle: recipeFilePath=recipe.yaml outputDir=./bundles bundlerTypes=[gpu-operator]
bundle generation completed: success=1 errors=0 duration_sec=0.025941625 summary=Generated 6 files (8.4 KB) in 26ms. Success: 1/1 bundlers.
```

Review the created bundles:

```shell
open ./bundles
```
