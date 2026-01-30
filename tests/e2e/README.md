# E2E Tests

End-to-end tests for CNS CLI and API, including snapshot, recipe, validate, and bundle workflows.

## Quick Start

```bash
# 1. Start the local dev environment (Kind cluster + cnsd)
make dev-env

# 2. In another terminal, set up port forwarding
kubectl port-forward -n cns svc/cnsd 8080:8080

# 3. Run tests
make e2e-tilt
```

## What's Tested

| Test | Description |
|------|-------------|
| `build/cnsctl` | Binary builds successfully |
| `cli/help` | CLI help and version commands |
| `api/health` | Health endpoint responds |
| `api/ready` | Readiness endpoint responds |
| `api/metrics` | GET `/metrics` Prometheus endpoint |
| `cli/recipe/*` | Recipe generation (query params, criteria file, overrides) |
| `cli/bundle/*` | Bundle generation (helm, argocd, node selectors) |
| `cli/external-data/*` | External data directory (`--data` flag) |
| `cli/format/*` | Output format variations (`--format json/table`) |
| `cli/deploy-agent/*` | Snapshot `--deploy-agent` CLI flag validation |
| `api/recipe/*` | GET/POST `/v1/recipe` endpoints |
| `api/bundle/*` | POST `/v1/bundle` endpoint |
| `snapshot/*` | Snapshot with deploy-agent (requires fake GPU setup) |
| `recipe/from-snapshot` | Recipe from ConfigMap snapshot (cm://...) |
| `validate/*` | Recipe validation against snapshot |
| `bundle/oci-push` | Bundle as OCI image to local registry |

## Fake GPU Testing

The e2e tests simulate GPU nodes using a fake nvidia-smi script that returns realistic output for **8x NVIDIA B200 192GB GPUs** (Blackwell architecture):

```
GPU 0: NVIDIA B200 (UUID: GPU-fake-0000-0000-0000-000000000000)
Driver: NVIDIA-SMI 560.35.03    CUDA Version: 12.6
Memory: 192GB HBM3e per GPU
```

Components:
1. **fake-nvidia-smi** - Script injected into Kind nodes (`tools/fake-nvidia-smi`)
2. **fake-gpu-operator** - Optional K8s-level GPU resource simulation

### Setting up Fake GPU locally

```bash
# Inject fake nvidia-smi into Kind worker nodes
for node in $(docker ps --filter "name=-worker" --format "{{.Names}}"); do
  docker cp tools/fake-nvidia-smi "${node}:/usr/local/bin/nvidia-smi"
  docker exec "$node" chmod +x /usr/local/bin/nvidia-smi
done

# Build and push cns image to local registry
KO_DOCKER_REPO=localhost:5001/cns ko build --bare --tags=local ./cmd/cnsctl

# Run tests with fake GPU enabled
FAKE_GPU_ENABLED=true CNS_IMAGE=localhost:5001/cns:local ./tests/e2e/run.sh
```

## Prerequisites

- Docker
- Kind
- kubectl
- Tilt
- ctlptl
- ko (for building cns image)

Install all tools:
```bash
brew install kind tilt-dev/tap/tilt tilt-dev/tap/ctlptl ko
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CNSD_URL` | `http://localhost:8080` | API URL |
| `OUTPUT_DIR` | temp dir | Test artifacts directory |
| `CNS_IMAGE` | `localhost:5001/cns:local` | CNS image for snapshot agent |
| `FAKE_GPU_ENABLED` | `false` | Enable fake GPU tests |
| `SNAPSHOT_NAMESPACE` | `gpu-operator` | Namespace for snapshot tests |
| `SNAPSHOT_CM` | `cns-e2e-snapshot` | ConfigMap name for snapshot |

## Manual Run

```bash
./tests/e2e/run.sh
```

### Example Output

```
[INFO] Setting up fake GPU environment
  $ docker cp fake-nvidia-smi cns-worker:/usr/local/bin/nvidia-smi
     → Simulated: GPU 0: NVIDIA B200 (UUID: GPU-fake-0000-0000-0000-000000000000)
     → Driver: NVIDIA-SMI 560.35.03    Driver Version: 560.35.03    CUDA Version: 12.6
[PASS] setup/fake-nvidia-smi

[INFO] --- Test: Recipe with query parameters ---
  $ cnsctl recipe --service eks --accelerator gb200 --os ubuntu --intent training -o basic.yaml
     → Generated recipe with 11 components
[PASS] cli/recipe/query-params

[INFO] --- Test: Recipe with external data ---
  $ cnsctl recipe --service eks --accelerator gb200 --os ubuntu --intent training --data ./examples/data
     → External component 'dgxc-teleport' included in recipe
[PASS] cli/external-data/recipe

[INFO] --- Test: Recipe with --format json ---
  $ cnsctl recipe --service eks --accelerator h100 --intent inference --format json
     → Valid JSON with 6 components
[PASS] cli/format/json

[INFO] --- Test: GET /metrics ---
  $ curl http://localhost:8080/metrics
     → HTTP 200 OK - Prometheus format (50 metrics)
     → CNS-specific metrics present
[PASS] api/metrics

[INFO] --- Test: Validate recipe ---
  $ cnsctl validate --recipe recipe.yaml --snapshot cm://gpu-operator/cns-e2e-snapshot
     → Validation: PASS (1 constraints checked)
[PASS] validate/recipe
```

## CI/CD

The e2e tests run automatically on:
- Push to `main` branch
- Pull requests to `main`

See `.github/workflows/e2e-test.yaml` for the CI configuration.

## Cleanup

```bash
make dev-env-clean
```
