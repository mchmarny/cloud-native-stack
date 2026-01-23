# CNS End-to-End Demo

## Install

```shell
curl -sfL https://raw.githubusercontent.com/mchmarny/cloud-native-stack/main/install | bash -s --
```

Test CLI:

```shell
cnsctl -h
```

## Recipe

Basic:

```shell
cnsctl recipe \
  --service eks \
  --accelerator gb200 \
  --os ubuntu \
  --intent training
```

Metadata overlays: `components=5 overlays=5`

![data flow](images/recipe.png)

Recipe from API: 

```shell
curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=gb200&intent=training" | jq .
```

Make Snapshot: 

```shell
cnsctl snapshot \
    --deploy-agent \
    --namespace gpu-operator \
    --image ghcr.io/mchmarny/cns:latest \
    --node-selector nodeGroup=customer-gpu
```

Check Snapshot in ConfigMap:

```shell
kubectl -n gpu-operator get cm cns-snapshot -o jsonpath='{.data.snapshot\.yaml}' | yq .
```

Recipe from Snapshot: 

```shell
cnsctl recipe \
  --snapshot cm://gpu-operator/cns-snapshot \
  --intent training \
  --output recipe.yaml
```

Recipe Constraints:

```shell
yq .constraints recipe.yaml
```

Validate Recipe: 

```shell
cnsctl validate \
  --recipe recipe.yaml \
  --snapshot cm://gpu-operator/cns-snapshot | yq .
```

## Bundle

Bundle from Recipe:

```shell
cnsctl bundle \
  --recipe recipe.yaml \
  --output ./bundles \
  --system-node-selector nodeGroup=system-pool \
  --accelerated-node-selector nodeGroup=customer-gpu \
  --accelerated-node-toleration nvidia.com/gpu=present:NoSchedule
```

Check bundle content: 

```shell
tree ./bundles
```

Bundle from Recipe using API: 

```shell
curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=h100&intent=training" | \
  curl -X POST "https://cns.dgxc.io/v1/bundle?deployer=argocd" \
    -H "Content-Type: application/json" -d @- -o bundles.zip
```

View bundle README: 

```shell
grip --browser --quiet ./bundles/README.md
```

Review Bundle and validate chart dependencies:

```shell
cd ./bundles && tree .
```

Prep the deployment: 

```shell
helm dependency update
tree .
```

## Links

Top 3 for each audience

### [For Users](https://github.com/mchmarny/cloud-native-stack/tree/main/docs/user-guide)
* [Installation Guide](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/user-guide/installation.md)
* [CLI Reference](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/user-guide/cli-reference.md)
* [API Reference](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/user-guide/api-reference.md)

### [For Integrators](https://github.com/mchmarny/cloud-native-stack/tree/main/docs/integration) (use CNS in their own solution)
* [Recipe Development Guide](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/integration/recipe-development.md)
* [Automation and CI/CD Integration](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/integration/automation.md)
* [Data Flow Architecture](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/integration/data-flow.md)

### [For Contributors](https://github.com/mchmarny/cloud-native-stack/tree/main/docs/architecture) (develop CNS itself)
* [Code of Conduct](https://github.com/mchmarny/cloud-native-stack/blob/main/CODE_OF_CONDUCT.md)
* [How to Contribute](https://github.com/mchmarny/cloud-native-stack/blob/main/CONTRIBUTING.md)
* [Metadata Concepts](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/architecture/data.md)
