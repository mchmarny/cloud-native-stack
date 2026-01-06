# eidos demo

> FYI: the latest CNS updates are all in a fork (https://github.com/mchmarny/cloud-native-stack/tree/main). Some of the links in there won’t work as is until merged upstream. Here are the key bits you can test today: 

## Install

```shell
curl -sfL https://raw.githubusercontent.com/mchmarny/cloud-native-stack/refs/heads/main/install | bash -s --
```

## Snapshot 

> Supports `kubectl` [deployment](https://github.com/mchmarny/cloud-native-stack/tree/main/deployments/eidos-agent) as well as CLI:

This will work on most of our clusters but you may need to update node selector and toleration flags if you are deploying somewhere else

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

```shell
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

To generate recipe (optimized config for given env/config combo) you can use above generated snapshot that was saved in a cluster:

```shell
eidos recipe \
    --snapshot cm://gpu-operator/eidos-snapshot \
    --intent training \
    --output recipe.yaml
```

Outputs: 

```shell
loading snapshot from: uri=cm://gpu-operator/eidos-snapshot
recipe generation completed: output=recipe.yaml
```

You can also skip the snapshot altogether and just generate the recipe from input parameters, for example: 

```shell
eidos recipe \
    --os ubuntu \
    --osv 24.04 \
    --kernel 6.8 \
    --service eks \
    --k8s 1.33 \
    --gpu gb200 \
    --intent training \
    --context \
    --format yaml
```

Whichever way you generated the recipe, its shape will be the same:

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

```shell
generating bundle: recipeFilePath=recipe.yaml outputDir=./bundles bundlerTypes=[gpu-operator]
bundle generation completed: success=1 errors=0 duration_sec=0.025941625 summary=Generated 6 files (8.4 KB) in 26ms. Success: 1/1 bundlers.
```

List all the artifacts created in the bundles:

```shell
open ./bundles
```

Review the Readme created for this bundle: 

```shell
cat ./bundles/gpu-operator/README.md
```

Follow the instructions in the doc to deploy that bundle.
