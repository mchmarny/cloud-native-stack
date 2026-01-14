# API Reference

Complete reference for using the CNS API Server.

## Overview

The CNS API Server provides HTTP REST access to recipe generation and bundle creation for GPU-accelerated infrastructure. Use the API for programmatic access to configuration recommendations and deployment artifacts.

```
┌──────────────┐      ┌──────────────┐
│ GET /recipe  │─────▶│   Recipe     │
└──────────────┘      └──────────────┘
        │
        ▼
┌──────────────┐      ┌──────────────┐
│ POST /bundle │─────▶│  bundles.zip │
└──────────────┘      └──────────────┘
```

**API vs CLI:**
- Use the **API** for remote recipe generation and bundle creation
- Use the **CLI** for local operations, snapshot capture, and ConfigMap integration

| Feature | API | CLI |
|---------|-----|-----|
| Recipe generation | ✅ GET /v1/recipe | ✅ `cnsctl recipe` |
| Bundle creation | ✅ POST /v1/bundle | ✅ `cnsctl bundle` |
| Snapshot capture | ❌ Use CLI | ✅ `cnsctl snapshot` |
| ConfigMap I/O | ❌ Use CLI | ✅ `cm://` URIs |
| Agent deployment | ❌ Use CLI | ✅ `--deploy-agent` |

## Base URL

**Production:**
```
https://cns.dgxc.io
```

**Local development:**
```
http://localhost:8080
```

Start the local server:
```shell
make server
```

## Quick Start

### Get a Recipe

Generate an optimized configuration recipe for your environment:

```shell
# Basic recipe for H100 on EKS
curl "https://cns.dgxc.io/v1/recipe?accelerator=h100&service=eks"

# Training workload on Ubuntu
curl "https://cns.dgxc.io/v1/recipe?accelerator=h100&service=eks&intent=training&os=ubuntu"

# Save recipe to file
curl -s "https://cns.dgxc.io/v1/recipe?accelerator=h100&service=eks" -o recipe.json
```

### Generate Bundles

Create deployment bundles from a recipe:

```shell
# Pipe recipe directly to bundle endpoint
curl -s "https://cns.dgxc.io/v1/recipe?accelerator=h100&service=eks" | \
  curl -X POST "https://cns.dgxc.io/v1/bundle?bundlers=gpu-operator" \
    -H "Content-Type: application/json" -d @- -o bundles.zip

# Extract the bundles
unzip bundles.zip -d ./bundles
```

## Endpoints

### GET /v1/recipe

Generate an optimized configuration recipe based on environment parameters.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `service` | string | any | K8s service: `eks`, `gke`, `aks`, `oke`, `any` |
| `accelerator` | string | any | GPU type: `h100`, `gb200`, `a100`, `l40`, `any` |
| `gpu` | string | any | Alias for `accelerator` |
| `intent` | string | any | Workload: `training`, `inference`, `any` |
| `os` | string | any | Node OS: `ubuntu`, `rhel`, `cos`, `amazonlinux`, `any` |
| `nodes` | integer | 0 | GPU node count (0 = any) |

**Examples:**

```shell
# Minimal request
curl "https://cns.dgxc.io/v1/recipe"

# Specify accelerator
curl "https://cns.dgxc.io/v1/recipe?accelerator=h100"

# Full specification
curl "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=h100&intent=training&os=ubuntu&nodes=8"

# Using gpu alias
curl "https://cns.dgxc.io/v1/recipe?gpu=gb200&service=gke"

# Pretty print with jq
curl -s "https://cns.dgxc.io/v1/recipe?accelerator=h100" | jq '.'
```

**Response:**

```json
{
  "apiVersion": "cns.nvidia.com/v1alpha1",
  "kind": "Recipe",
  "metadata": {
    "version": "v1.0.0",
    "created": "2026-01-11T10:30:00Z",
    "appliedOverlays": [
      "service=eks, accelerator=h100, intent=training"
    ]
  },
  "criteria": {
    "service": "eks",
    "accelerator": "h100",
    "intent": "training",
    "os": "ubuntu"
  },
  "componentRefs": [
    {
      "name": "gpu-operator",
      "version": "v25.3.3",
      "order": 1,
      "repository": "https://helm.ngc.nvidia.com/nvidia"
    },
    {
      "name": "network-operator",
      "version": "v25.4.0",
      "order": 2,
      "repository": "https://helm.ngc.nvidia.com/nvidia"
    }
  ],
  "constraints": {
    "driver": {
      "version": "580.82.07",
      "cudaVersion": "13.1"
    }
  }
}
```

---

### POST /v1/bundle

Generate deployment bundles from a recipe.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `bundlers` | string | (all) | Comma-delimited list of bundler types to execute |

**Request Body:**

The request body is the recipe (RecipeResult) directly. No wrapper object needed.

**Supported Bundlers:**

| Bundler | Description |
|---------|-------------|
| `gpu-operator` | NVIDIA GPU Operator |
| `network-operator` | NVIDIA Network Operator |
| `cert-manager` | Certificate Manager |
| `nvsentinel` | NVSentinel monitoring |
| `skyhook` | Skyhook node optimization |

**Examples:**

```shell
# Simplest workflow: pipe recipe to bundle
curl -s "https://cns.dgxc.io/v1/recipe?accelerator=h100&service=eks" | \
  curl -X POST "https://cns.dgxc.io/v1/bundle?bundlers=gpu-operator" \
    -H "Content-Type: application/json" -d @- -o bundles.zip

# Generate GPU Operator bundle from saved recipe
curl -X POST "https://cns.dgxc.io/v1/bundle?bundlers=gpu-operator" \
  -H "Content-Type: application/json" \
  -d @recipe.json \
  -o bundles.zip

# Generate all available bundles (no bundlers param)
curl -X POST "https://cns.dgxc.io/v1/bundle" \
  -H "Content-Type: application/json" \
  -d '{
    "apiVersion": "cns.nvidia.com/v1alpha1",
    "kind": "Recipe",
    "componentRefs": [
      {"name": "gpu-operator", "version": "v25.3.3", "type": "helm"},
      {"name": "network-operator", "version": "v25.4.0", "type": "helm"}
    ]
  }' \
  -o bundles.zip

# Generate multiple specific bundles
curl -X POST "https://cns.dgxc.io/v1/bundle?bundlers=gpu-operator,network-operator" \
  -H "Content-Type: application/json" \
  -d '{
    "apiVersion": "cns.nvidia.com/v1alpha1",
    "kind": "Recipe",
    "componentRefs": [
      {"name": "gpu-operator", "version": "v25.3.3", "type": "helm"},
      {"name": "network-operator", "version": "v25.4.0", "type": "helm"}
    ]
  }' \
  -o bundles.zip
```

**Response Headers:**

| Header | Description | Example |
|--------|-------------|---------|
| `Content-Type` | Always `application/zip` | `application/zip` |
| `Content-Disposition` | Download filename | `attachment; filename="bundles.zip"` |
| `X-Bundle-Files` | Total files in archive | `10` |
| `X-Bundle-Size` | Uncompressed size (bytes) | `45678` |
| `X-Bundle-Duration` | Generation time | `1.234s` |

**Bundle Structure:**

```
bundles.zip
├── gpu-operator/
│   ├── values.yaml              # Helm chart values
│   ├── manifests/
│   │   ├── clusterpolicy.yaml   # ClusterPolicy CR
│   │   └── dcgm-exporter.yaml   # DCGM Exporter config
│   ├── scripts/
│   │   ├── install.sh           # Installation script
│   │   └── uninstall.sh         # Cleanup script
│   ├── README.md                # Deployment instructions
│   └── checksums.txt            # SHA256 checksums
└── network-operator/
    └── ...
```

---

### GET /health

Service health check (liveness probe).

```shell
curl "https://cns.dgxc.io/health"
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-11T10:30:00Z"
}
```

---

### GET /ready

Service readiness check (readiness probe).

```shell
curl "https://cns.dgxc.io/ready"
```

**Response:**
```json
{
  "status": "ready",
  "timestamp": "2026-01-11T10:30:00Z"
}
```

---

### GET /metrics

Prometheus metrics endpoint.

```shell
curl "https://cns.dgxc.io/metrics"
```

**Key Metrics:**

| Metric | Type | Description |
|--------|------|-------------|
| `cns_http_requests_total` | counter | Total HTTP requests by method, path, status |
| `cns_http_request_duration_seconds` | histogram | Request latency distribution |
| `cns_http_requests_in_flight` | gauge | Current concurrent requests |
| `cns_rate_limit_rejects_total` | counter | Rate limit rejections |

## Complete Workflow Example

Fetch a recipe and generate bundles in one workflow:

```shell
#!/bin/bash

# Step 1: Get recipe for H100 on EKS for training
echo "Fetching recipe..."
curl -s "https://cns.dgxc.io/v1/recipe?accelerator=h100&service=eks&intent=training" \
  -o recipe.json

# Display recipe summary
echo "Recipe components:"
jq -r '.componentRefs[] | "  - \(.name): \(.version)"' recipe.json

# Step 2: Generate bundles from recipe (pipe directly)
echo "Generating bundles..."
curl -s -X POST "https://cns.dgxc.io/v1/bundle?bundlers=gpu-operator" \
  -H "Content-Type: application/json" \
  -d @recipe.json \
  -o bundles.zip

# Alternative: one-liner without intermediate file
# curl -s "https://cns.dgxc.io/v1/recipe?accelerator=h100&service=eks" | \
#   curl -X POST "https://cns.dgxc.io/v1/bundle?bundlers=gpu-operator" \
#     -H "Content-Type: application/json" -d @- -o bundles.zip

# Step 3: Extract and verify
echo "Extracting bundles..."
unzip -q bundles.zip -d ./deployment

# Verify checksums
echo "Verifying checksums..."
cd deployment/gpu-operator
sha256sum -c checksums.txt

# Step 4: Deploy (example)
echo "Bundle ready for deployment:"
ls -la
```

## Error Handling

**Error Response Format:**

```json
{
  "code": "ERROR_CODE",
  "message": "Human-readable error description",
  "details": { ... },
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-01-11T10:30:00Z",
  "retryable": true
}
```

**Error Codes:**

| Code | HTTP Status | Description | Retryable |
|------|-------------|-------------|-----------|
| `INVALID_REQUEST` | 400 | Invalid query parameters or request body | No |
| `METHOD_NOT_ALLOWED` | 405 | Wrong HTTP method | No |
| `NO_MATCHING_RULE` | 404 | No configuration found | No |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests | Yes |
| `INTERNAL_ERROR` | 500 | Server error | Yes |

**Handling Rate Limits:**

```shell
# Check rate limit headers
curl -I "https://cns.dgxc.io/v1/recipe?accelerator=h100"

# Response headers:
# X-RateLimit-Limit: 100
# X-RateLimit-Remaining: 95
# X-RateLimit-Reset: 1736589000
```

When rate limited (HTTP 429), use the `Retry-After` header:

```shell
# Retry with backoff
response=$(curl -s -w "%{http_code}" "https://cns.dgxc.io/v1/recipe?accelerator=h100")
if [ "${response: -3}" = "429" ]; then
  retry_after=$(curl -sI "https://cns.dgxc.io/v1/recipe" | grep -i "Retry-After" | awk '{print $2}')
  echo "Rate limited. Retrying after ${retry_after}s..."
  sleep "$retry_after"
fi
```

## Rate Limiting

- **Limit**: 100 requests per second per IP
- **Burst**: 200 requests
- **Headers**: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- **429 Response**: Includes `Retry-After` header

## Programming Language Examples

### Python

```python
import requests
import zipfile
import io

BASE_URL = "https://cns.dgxc.io"

# Get recipe
params = {
    "accelerator": "h100",
    "service": "eks",
    "intent": "training",
    "os": "ubuntu"
}

resp = requests.get(f"{BASE_URL}/v1/recipe", params=params)
resp.raise_for_status()
recipe = resp.json()

print(f"Recipe has {len(recipe['componentRefs'])} components")

# Generate bundles
bundle_req = {
    "recipe": recipe,
    "bundlers": ["gpu-operator"]
}

resp = requests.post(f"{BASE_URL}/v1/bundle", json=bundle_req)
resp.raise_for_status()

# Extract zip
with zipfile.ZipFile(io.BytesIO(resp.content)) as zf:
    zf.extractall("./deployment")
    print(f"Extracted {len(zf.namelist())} files")
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
)

func main() {
    baseURL := "https://cns.dgxc.io"

    // Get recipe
    params := url.Values{}
    params.Add("accelerator", "h100")
    params.Add("service", "eks")
    
    resp, err := http.Get(baseURL + "/v1/recipe?" + params.Encode())
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    var recipe map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&recipe)
    
    fmt.Printf("Got recipe with %d components\n", 
        len(recipe["componentRefs"].([]interface{})))
}
```

### JavaScript/Node.js

```javascript
const BASE_URL = "https://cns.dgxc.io";

async function main() {
    // Get recipe
    const params = new URLSearchParams({
        accelerator: "h100",
        service: "eks",
        intent: "training"
    });
    
    const recipeResp = await fetch(`${BASE_URL}/v1/recipe?${params}`);
    const recipe = await recipeResp.json();
    
    console.log(`Recipe has ${recipe.componentRefs.length} components`);
    
    // Generate bundles
    const bundleResp = await fetch(`${BASE_URL}/v1/bundle`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            recipe: recipe,
            bundlers: ["gpu-operator"]
        })
    });
    
    // Save zip
    const buffer = await bundleResp.arrayBuffer();
    require("fs").writeFileSync("bundles.zip", Buffer.from(buffer));
    console.log("Bundles saved to bundles.zip");
}

main();
```

## OpenAPI Specification

The full OpenAPI 3.1 specification is available at:
[api/cns/v1/server.yaml](../../api/cns/v1/server.yaml)

Generate client SDKs:

```shell
# Download spec
curl https://raw.githubusercontent.com/NVIDIA/cloud-native-stack/main/api/cns/v1/server.yaml \
  -o openapi.yaml

# Generate Python client
openapi-generator-cli generate -i openapi.yaml -g python -o ./python-client

# Generate Go client
openapi-generator-cli generate -i openapi.yaml -g go -o ./go-client

# Generate TypeScript client
openapi-generator-cli generate -i openapi.yaml -g typescript-fetch -o ./ts-client
```

## Troubleshooting

### Common Issues

**"Invalid accelerator type" error:**
```shell
# Use valid values: h100, gb200, a100, l40, any
curl "https://cns.dgxc.io/v1/recipe?accelerator=h100"
```

**"Recipe is required" error:**
```shell
# Ensure recipe is in request body
curl -X POST "https://cns.dgxc.io/v1/bundle" \
  -H "Content-Type: application/json" \
  -d '{"recipe": {...}}'  # recipe must not be null
```

**Empty zip file:**
```shell
# Check recipe has componentRefs
curl -s "https://cns.dgxc.io/v1/recipe?accelerator=h100" | jq '.componentRefs'
```

**Connection refused (local):**
```shell
# Start local server first
make server
```

## See Also

- [CLI Reference](cli-reference.md) - Command-line interface
- [Agent Deployment](agent-deployment.md) - Kubernetes agent for snapshot capture
- [Installation Guide](installation.md) - Setup instructions
- [Integration API Reference](../integration/api-reference.md) - Detailed API specification
