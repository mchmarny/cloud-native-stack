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
  --intent training | yq .
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
  --output ./bundle \
  --system-node-selector nodeGroup=system-pool \
  --accelerated-node-selector nodeGroup=customer-gpu \
  --accelerated-node-toleration nvidia.com/gpu=present:NoSchedule
```

Bundle from Recipe using API: 

```shell
curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=h100&intent=training" | \
  curl -X POST "https://cns.dgxc.io/v1/bundle?deployer=argocd" \
    -H "Content-Type: application/json" -d @- -o bundle.zip
```

Same content: 

```shell
unzip -l bundle.zip
```

Navigate into the bundle:

```shell
cd ./bundle
```

Check bundle content: 

```shell
tree .
```

Review the checksums: 

```shell
cat checksums.txt
```

Check the integrity of its content

```shell
shasum -a 256 -c checksums.txt
```

View bundle README: 

```shell
grip --browser --quiet ./bundle/README.md
```

Prep the deployment: 

```shell
helm dependency update
tree .
```

Bundle as an OCI image:

```shell
cnsctl bundle \
  --recipe recipe.yaml \
  --output oci://ghcr.io/mchmarny/cns-bundle \
  --deployer argocd \
  --image-refs .digest
```

Review manifest: 

```shell
crane manifest "ghcr.io/mchmarny/cns-bundle@$(cat .digest)" | jq .
```

Unpack the image: 

```shell
skopeo copy "docker://ghcr.io/mchmarny/cns-bundle@$(cat .digest)" oci:image-oci
mkdir -p ./cns-unpacked
oras pull --oci-layout "image-oci@$(cat .digest)" -o ./cns-unpacked
```

## Links

* [Installation Guide](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/user-guide/installation.md)
* [CLI Reference](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/user-guide/cli-reference.md)
* [API Reference](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/user-guide/api-reference.md)
