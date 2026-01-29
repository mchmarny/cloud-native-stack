# E2E Tests

End-to-end tests for CNS CLI and API.

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
| `api/health` | Health endpoint responds |
| `api/ready` | Readiness endpoint responds |
| `cli/recipe/*` | Recipe generation (query params, criteria file, overrides) |
| `cli/bundle/*` | Bundle generation (helm, argocd, node selectors) |
| `api/recipe/*` | GET/POST `/v1/recipe` endpoints |
| `api/bundle/*` | POST `/v1/bundle` endpoint |

## Prerequisites

- Docker
- Kind
- kubectl
- Tilt
- ctlptl

Install all tools:
```bash
brew install kind tilt-dev/tap/tilt tilt-dev/tap/ctlptl
```

## Manual Run

```bash
./tests/e2e/run.sh
```

Options:
- `CNSD_URL` - API URL (default: `http://localhost:8080`)
- `OUTPUT_DIR` - Test artifacts directory (default: temp dir)

## Cleanup

```bash
make dev-env-clean
```
