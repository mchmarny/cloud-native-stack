# Today

CNS Current Repo - https://github.com/NVIDIA/cloud-native-stack

* Reference architecture > ISV-focused 
* Installation Guide > Latest
* Long list of commands 

# Today's Demo 

Project Eidos == CNS v2
Proposal - https://docs.google.com/document/d/1oNHntk3dymohj4JGqJGBk8mTapY5ExeRqVf1y98Vq-0/edit?tab=t.96kogtny99fp#heading=h.agbo1o61yxf6

TLDR; K8s Cluster Configuration that's:
* Validated
* Use-case AND Environment Optimized 
* AND Reproducible

# Demo 

Most in terminal, so quick overview first: 

IMG: demo.png

Problem: 
* GPU Operator alone: 10+ min
* OS/Kernel/K8s/Use-case optimized: 20-30 or more
* Add other components (e.g. Net Operator, KATA, CNI,...), each with their own values: >100s

Unsustainable:
* Humans == bugs 
* Each component evolves at different rate (e.g. new driver versions)
* Config combo that was optimal or even valid yesterday is broken today

# Eidos Demo 

Initial iteration:
* SNAPSHOT: Agent into the cluster, capture measurements
* RECIPE: Use snapshot to generate optimal measurement for specific intent
* BUNDLE: Use recipe to generate component bundle with optimized configs

## Snapshot 

Also available as K8s deployment (RBAC+Manifest)

``` shell
eidos snapshot \
    --deploy-agent \
    --namespace gpu-operator \
    --image ghcr.io/mchmarny/eidos:latest \
    --node-selector nodeGroup=customer-gpu \
    --toleration dedicated=user-workload:NoSchedule \
    --toleration dedicated=user-workload:NoExecute \
    --cleanup-rbac
```

Key: 
* No Call home, never exported measurements outside of the cluster

In-cluster snapshot: 

```shell
kubectl -n gpu-operator get cm eidos-snapshot -o yaml | yq .
```

## Recipe

```shell
eidos recipe \
    --snapshot cm://gpu-operator/eidos-snapshot \
    --intent training \
    --output recipe.yaml
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

You can also skip the snapshot: 

```shell
eidos recipe \
    --service eks \
    --gpu gb200 \
    --intent training
```

OR, query the CNS APIs directly - https://cns.dgxc.io

```shell
curl -fsSL "https://cns.dgxc.io/v1/recipe?gpu=gb200&service=eks&intent=training" | jq .
```

Possible cause of the Core Component: Metadata

IMG: data.png

## Bundle

```shell
eidos bundle \
  --recipe recipe.yaml \
  --output ./bundles
```

List artifacts:

```shell
open ./bundles
```

Today: Only 3 bundlers (GPU, Net, Skyhook)

Bundles could be: 
* ArgoCD App of Apps (combining N bundles)
* Singe container image (self-extracting in cluster)

Review 1 Bundle - GPU Operator:

```shell
grip --browser --quiet ./bundles/gpu-operator/README.md
```

Review Values (context)

```shell
code ./bundles/gpu-operator/values.yaml
```

WARNING: Not yet optimized or validated!
Demo == automation framework

## Reproducibility

Same RECIPE/Eidos version == Exact Same Bundle (byte-level)

```shell
code ./bundles/gpu-operator/checksums.txt
```

IMG: s3c.png
