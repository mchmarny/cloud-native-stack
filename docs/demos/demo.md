# eidos demo

![demo overview](./demo.png)

> Note: latest CNS updates are all in a fork (https://github.com/mchmarny/cloud-native-stack/tree/main). Some of the links in there won’t work until PR is merged upstream. Here are the key bits you can use today.

## Install

```shell
curl -sfL https://raw.githubusercontent.com/mchmarny/cloud-native-stack/refs/heads/main/install | bash -s --
```

Validate install: 

```shell
eidos -v
```

Expected result: 

```shell
eidos version 0.9.0 (commit: 72930f3, date: 2026-01-06T18:49:40Z)
```

## Snapshot 

Supports `kubectl` [deployment](https://github.com/mchmarny/cloud-native-stack/tree/main/deployments/eidos-agent) as well as CLI-based agent deployment:

> Note: You may need to update node selector and toleration flags in your cluster.

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

Terminal output:

```shell
deploying agent: namespace=gpu-operator
agent deployed successfully
waiting for Job completion: job=eidos timeout=5m0s
job completed successfully
snapshot saved to ConfigMap: uri=cm://gpu-operator/eidos-snapshot
```

In-cluster snapshot: 

```shell
kubectl -n gpu-operator get cm eidos-snapshot -o yaml | yq .
```

Output: 

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  creationTimestamp: "2026-01-06T15:01:25Z"
  labels:
    app.kubernetes.io/component: snapshot
    app.kubernetes.io/name: eidos
  name: eidos-snapshot
  namespace: gpu-operator
  resourceVersion: "121025717"
  uid: 547f7c4f-0eb4-4e86-a2a2-7412c09a615b
data:
  format: yaml
  snapshot.yaml: |
    kind: Snapshot
    apiVersion: snapshot.dgxc.io/v1
    metadata:
        snapshot-timestamp: "2026-01-07T16:11:21Z"
        snapshot-version: 0.9.0
        source-node: ip-10-0-160-248.ec2.internal
    measurements:
    ...
  timestamp: "2026-01-07T16:11:24Z"
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

Review the recipe:

```shell
cat recipe.yaml
```

Or even cleaner: 

```shell
yq eval '.measurements[]
  | {"type": .type,
     "subtypes": (.subtypes | map({"subtype": .subtype, "count": (.data | length)}))
    }' recipe.yaml
```

You can also skip the snapshot altogether, and just generate the recipe from input parameters, for example: 

```shell
eidos recipe \
    --service eks \
    --gpu gb200 \
    --intent training \
    --context
```

Ypu can also query the CNS APIs directly (sans CLI):

```shell
curl -fsSL "https://cns.dgxc.io/v1/recipe?gpu=gb200&service=eks&intent=training" | jq .
```

## Bundle

Whichever way you got your recipe (these are versioned), you can generate a deployment bundle from that recipe:

```shell
eidos bundle \
  --recipe recipe.yaml \
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

Review the Readme created for this bundle (using [grip](https://github.com/joeyespo/grip) to render markdown): 

```shell
grip --browser --quiet ./bundles/gpu-operator/README.md
```

Follow the instructions in the doc to deploy that bundle.
