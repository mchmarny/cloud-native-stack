# API Reference

Complete reference for the Eidos API Server REST API.

## Overview

The Eidos API provides HTTP REST access to **recipe generation** (Step 2 of the Cloud Native Stack workflow). It enables programmatic integration with automation tools, CI/CD pipelines, and custom applications.

**Base URL:** `https://cns.dgxc.io`

**API Capabilities:**
- ✅ Recipe generation from query parameters
- ✅ Version negotiation via Accept header
- ✅ Rate limiting and request tracking
- ✅ Health and metrics endpoints
- ✅ SLSA Build Level 3 attestations
- ✅ Production deployment at https://cns.dgxc.io

**API Limitations:**
- ❌ No snapshot capture (use CLI or Agent)
- ❌ No bundle generation (use CLI)
- ❌ No snapshot analysis (query mode only)
- ❌ No ConfigMap integration (use CLI for Kubernetes-native storage)

**For complete workflow**, use the CLI:
- Snapshot: `eidos snapshot -o cm://namespace/name`
- Recipe: `eidos recipe -f cm://namespace/name -o recipe.yaml`
- Bundle: `eidos bundle -f recipe.yaml -o ./bundles`
- Agent: Kubernetes Job for automated snapshot capture
- E2E Testing: Validated with `tools/e2e` script

## Authentication

**Current:** No authentication required (public API)  
**Future:** API keys for production use

## Base URL

```
https://cns.dgxc.io
```

For local development:
```
http://localhost:8080
```

## Endpoints

### GET /

Service information and available routes.

**Response:**
```json
{
  "service": "eidos-api-server",
  "version": "v0.7.6",
  "routes": ["/v1/recipe"]
}
```

---

### GET /v1/recipe

Generate optimized configuration recipe based on environment parameters.

**Query Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `os` | string | No | any | OS family: ubuntu, cos, rhel, any |
| `osv` | string | No | any | OS version: 24.04, 22.04, etc. |
| `kernel` | string | No | any | Kernel version (supports vendor suffixes) |
| `service` | string | No | any | K8s service: eks, gke, aks, self-managed, any |
| `k8s` | string | No | any | Kubernetes version (supports vendor formats) |
| `gpu` | string | No | any | GPU type: h100, gb200, a100, l40, any |
| `intent` | string | No | any | Workload intent: training, inference, any |
| `context` | boolean | No | false | Include context metadata (true/false or 1/0) |

**Request Headers:**

| Header | Required | Description |
|--------|----------|-------------|
| `X-Request-Id` | No | Client-provided request ID for tracing |
| `Accept` | No | Content negotiation (future versioning) |

**Response Headers:**

| Header | Description |
|--------|-------------|
| `X-Request-Id` | Server-assigned or echoed request ID |
| `Cache-Control` | Cache directives (public, max-age=300) |
| `X-RateLimit-Limit` | Request quota (100/second) |
| `X-RateLimit-Remaining` | Remaining requests in window |
| `X-RateLimit-Reset` | Unix timestamp when quota resets |

**Success Response (200 OK):**

```json
{
  "apiVersion": "v1",
  "kind": "Recipe",
  "metadata": {
    "created": "2025-12-31T10:30:00Z",
    "recipe-version": "v1.0.0"
  },
  "request": {
    "os": "ubuntu",
    "gpu": "h100",
    "service": "eks",
    "intent": "training"
  },
  "matchedRules": [
    "OS: ubuntu, GPU: h100, Service: eks, Intent: training"
  ],
  "measurements": [
    {
      "type": "K8s",
      "subtypes": [
        {
          "subtype": "image",
          "data": {
            "gpu-operator": "v25.3.3",
            "driver": "580.82.07"
          }
        }
      ]
    },
    {
      "type": "GPU",
      "subtypes": [
        {
          "subtype": "driver",
          "data": {
            "version": "580.82.07",
            "cuda-version": "13.1"
          }
        }
      ]
    }
  ]
}
```

**Error Responses:**

**400 Bad Request** - Invalid parameters:
```json
{
  "code": "INVALID_PARAMETER",
  "message": "invalid gpu type: must be one of h100, gb200, a100, l40, any",
  "details": {
    "parameter": "gpu",
    "provided": "invalid-gpu"
  },
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-31T10:30:00Z",
  "retryable": false
}
```

**404 Not Found** - No matching configuration:
```json
{
  "code": "NO_MATCHING_RULE",
  "message": "no configuration recipe found for the specified parameters",
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-31T10:30:00Z",
  "retryable": false
}
```

**429 Too Many Requests** - Rate limit exceeded:
```json
{
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "rate limit exceeded, please retry after indicated time",
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-31T10:30:00Z",
  "retryable": true
}
```

Response includes `Retry-After` header.

**500 Internal Server Error** - Server error:
```json
{
  "code": "INTERNAL_ERROR",
  "message": "an internal error occurred processing your request",
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-31T10:30:00Z",
  "retryable": true
}
```

---

### GET /health

Liveness probe endpoint.

**Response (200 OK):**
```json
{
  "status": "healthy",
  "timestamp": "2025-12-31T10:30:00Z"
}
```

---

### GET /ready

Readiness probe endpoint.

**Response (200 OK):**
```json
{
  "status": "ready",
  "timestamp": "2025-12-31T10:30:00Z"
}
```

**Response (503 Service Unavailable):**
```json
{
  "status": "not_ready",
  "timestamp": "2025-12-31T10:30:00Z",
  "reason": "service is initializing"
}
```

---

### GET /metrics

Prometheus metrics endpoint.

**Response (200 OK):**
```
# HELP eidos_http_requests_total Total HTTP requests
# TYPE eidos_http_requests_total counter
eidos_http_requests_total{method="GET",path="/v1/recipe",status="200"} 42

# HELP eidos_http_request_duration_seconds HTTP request duration
# TYPE eidos_http_request_duration_seconds histogram
eidos_http_request_duration_seconds_bucket{method="GET",path="/v1/recipe",le="0.1"} 40
eidos_http_request_duration_seconds_bucket{method="GET",path="/v1/recipe",le="0.5"} 42

# HELP eidos_http_requests_in_flight Current HTTP requests in flight
# TYPE eidos_http_requests_in_flight gauge
eidos_http_requests_in_flight 3

# HELP eidos_rate_limit_rejects_total Total rate limit rejections
# TYPE eidos_rate_limit_rejects_total counter
eidos_rate_limit_rejects_total 5
```

## Usage Examples

### cURL

**Basic query:**
```shell
curl "https://cns.dgxc.io/v1/recipe?os=ubuntu&gpu=h100"
```

**Full specification:**
```shell
curl "https://cns.dgxc.io/v1/recipe?os=ubuntu&osv=24.04&kernel=6.8&service=eks&k8s=1.33&gpu=h100&intent=training&context=true"
```

**With request ID:**
```shell
curl -H "X-Request-Id: $(uuidgen)" \
  "https://cns.dgxc.io/v1/recipe?os=ubuntu&gpu=gb200"
```

**Save to file:**
```shell
curl "https://cns.dgxc.io/v1/recipe?os=ubuntu&gpu=h100" -o recipe.json
```

### Python (requests)

```python
import requests

# Basic request
params = {
    'os': 'ubuntu',
    'gpu': 'h100',
    'service': 'eks',
    'intent': 'training'
}

response = requests.get('https://cns.dgxc.io/v1/recipe', params=params)

if response.status_code == 200:
    recipe = response.json()
    print(f"Matched {len(recipe['matchedRules'])} rules")
    print(f"GPU Operator version: {recipe['measurements'][0]['subtypes'][0]['data']['gpu-operator']}")
else:
    error = response.json()
    print(f"Error: {error['message']}")
```

**With rate limiting:**
```python
import requests
import time

def get_recipe_with_retry(params, max_retries=3):
    for attempt in range(max_retries):
        response = requests.get('https://cns.dgxc.io/v1/recipe', params=params)
        
        if response.status_code == 200:
            return response.json()
        elif response.status_code == 429:
            retry_after = int(response.headers.get('Retry-After', 60))
            print(f"Rate limited. Retrying after {retry_after} seconds...")
            time.sleep(retry_after)
        else:
            raise Exception(f"API error: {response.json()['message']}")
    
    raise Exception("Max retries exceeded")

recipe = get_recipe_with_retry({'os': 'ubuntu', 'gpu': 'h100'})
```

### Go (net/http)

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
)

type Recipe struct {
    APIVersion    string        `json:"apiVersion"`
    Kind          string        `json:"kind"`
    MatchedRules  []string      `json:"matchedRules"`
    Measurements  []interface{} `json:"measurements"`
}

func main() {
    baseURL := "https://cns.dgxc.io/v1/recipe"
    
    // Build query
    params := url.Values{}
    params.Add("os", "ubuntu")
    params.Add("gpu", "h100")
    params.Add("service", "eks")
    
    // Make request
    resp, err := http.Get(baseURL + "?" + params.Encode())
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    
    // Parse response
    var recipe Recipe
    if err := json.NewDecoder(resp.Body).Decode(&recipe); err != nil {
        panic(err)
    }
    
    fmt.Printf("Matched %d rules\n", len(recipe.MatchedRules))
}
```

### JavaScript (fetch)

```javascript
// Basic request
async function getRecipe(params) {
  const query = new URLSearchParams(params);
  const response = await fetch(`https://cns.dgxc.io/v1/recipe?${query}`);
  
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message);
  }
  
  return response.json();
}

// Usage
const recipe = await getRecipe({
  os: 'ubuntu',
  gpu: 'h100',
  service: 'eks',
  intent: 'training'
});

console.log(`Matched ${recipe.matchedRules.length} rules`);
```

**With rate limit handling:**
```javascript
async function getRecipeWithRetry(params, maxRetries = 3) {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    const query = new URLSearchParams(params);
    const response = await fetch(`https://cns.dgxc.io/v1/recipe?${query}`);
    
    if (response.ok) {
      return response.json();
    }
    
    if (response.status === 429) {
      const retryAfter = parseInt(response.headers.get('Retry-After') || '60');
      console.log(`Rate limited. Retrying after ${retryAfter}s...`);
      await new Promise(resolve => setTimeout(resolve, retryAfter * 1000));
      continue;
    }
    
    const error = await response.json();
    throw new Error(error.message);
  }
  
  throw new Error('Max retries exceeded');
}
```

### Shell Script

```bash
#!/bin/bash
# Generate recipes for multiple environments

environments=(
  "os=ubuntu&gpu=h100&service=eks"
  "os=ubuntu&gpu=gb200&service=gke"
  "os=rhel&gpu=a100&service=aks"
)

for env in "${environments[@]}"; do
  echo "Fetching recipe for: $env"
  
  curl -s "https://cns.dgxc.io/v1/recipe?${env}" \
    | jq -r '.matchedRules[]'
  
  echo ""
done
```

## Rate Limiting

**Limits:**
- **Rate**: 100 requests per second per IP
- **Burst**: 200 requests

**Headers:**
- `X-RateLimit-Limit`: Maximum requests per window
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Unix timestamp when window resets

**Best practices:**
1. Respect `Retry-After` header when rate limited
2. Implement exponential backoff
3. Cache responses when possible (Cache-Control header)
4. Use request IDs for debugging

## Error Handling

**Error response structure:**
```json
{
  "code": "ERROR_CODE",
  "message": "Human-readable message",
  "details": { /* Optional context */ },
  "requestId": "uuid",
  "timestamp": "ISO-8601",
  "retryable": true/false
}
```

**Error codes:**

| Code | HTTP Status | Description | Retryable |
|------|-------------|-------------|-----------|
| `INVALID_REQUEST` | 400 | Invalid query parameters | No |
| `METHOD_NOT_ALLOWED` | 405 | Wrong HTTP method | No |
| `NO_MATCHING_RULE` | 404 | No configuration found | No |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests | Yes |
| `INTERNAL_ERROR` | 500 | Server error | Yes |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily down | Yes |

## OpenAPI Specification

Full OpenAPI 3.1 specification: [api/eidos/v1/api-server-v1.yaml](../../../api/eidos/v1/api-server-v1.yaml)

**Generate client SDKs:**
```shell
# Download spec
curl https://raw.githubusercontent.com/NVIDIA/cloud-native-stack/main/api/eidos/v1/api-server-v1.yaml -o spec.yaml

# Generate Python client
openapi-generator-cli generate -i spec.yaml -g python -o ./python-client

# Generate Go client
openapi-generator-cli generate -i spec.yaml -g go -o ./go-client
```

## Deployment

See [Kubernetes Deployment](kubernetes-deployment.md) for deploying your own API server instance.

## Monitoring

**Health checks:**
```shell
# Liveness
curl https://cns.dgxc.io/health

# Readiness
curl https://cns.dgxc.io/ready
```

**Metrics (Prometheus):**
```shell
curl https://cns.dgxc.io/metrics
```

**Key metrics:**
- `eidos_http_requests_total` - Total requests by method, path, status
- `eidos_http_request_duration_seconds` - Request latency histogram
- `eidos_http_requests_in_flight` - Current concurrent requests
- `eidos_rate_limit_rejects_total` - Rate limit rejections

## See Also

- [Data Flow](data-flow.md) - Understanding recipe data architecture
- [Automation](automation.md) - CI/CD integration patterns
- [Kubernetes Deployment](kubernetes-deployment.md) - Self-hosted deployment
- [CLI Reference](../user-guide/cli-reference.md) - CLI alternative
