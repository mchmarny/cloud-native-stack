# Automation and CI/CD Integration

Integrate Cloud Native Stack into your automation pipelines for continuous GPU infrastructure optimization.

## Overview

Cloud Native Stack integrates with CI/CD pipelines to:
- **Capture** cluster configurations automatically
- **Generate** optimized recipes based on environment changes
- **Deploy** GPU operators with validated configurations
- **Monitor** configuration drift across clusters

## Integration Patterns

### Pattern 1: Configuration Snapshot + Drift Detection

Periodically capture snapshots and compare against baseline.

**Use case:** Detect unauthorized configuration changes

```yaml
# GitHub Actions
name: Configuration Drift Detection
on:
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours

jobs:
  snapshot:
    runs-on: ubuntu-latest
    steps:
      - name: Configure kubectl
        uses: azure/k8s-set-context@v1
        with:
          kubeconfig: ${{ secrets.KUBECONFIG }}
      
      - name: Deploy Eidos Agent
        run: |
          kubectl apply -f https://raw.githubusercontent.com/mchmarny/cloud-native-stack/main/deployments/eidos-agent/1-deps.yaml
          kubectl apply -f https://raw.githubusercontent.com/mchmarny/cloud-native-stack/main/deployments/eidos-agent/2-job.yaml
      
      - name: Wait for completion
        run: |
          kubectl wait --for=condition=complete --timeout=300s job/eidos -n gpu-operator
      
      - name: Capture snapshot from ConfigMap
        run: |
          kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > snapshot-$(date +%Y%m%d-%H%M%S).yaml
      
      - name: Compare with baseline
        run: |
          # Download baseline
          curl -O https://your-artifacts/baseline.yaml
          
          # Compare
          if ! diff -q baseline.yaml snapshot-*.yaml; then
            echo "::error::Configuration drift detected"
            diff baseline.yaml snapshot-*.yaml
            exit 1
          fi
      
      - name: Upload artifact
        uses: actions/upload-artifact@v3
        with:
          name: cluster-snapshots
          path: snapshot-*.yaml
```

### Pattern 2: Recipe-Based Deployment

Generate optimized configuration and deploy operators.

**Use case:** Deploy GPU Operator with environment-specific settings

```yaml
# GitLab CI
stages:
  - snapshot
  - recipe
  - bundle
  - deploy

capture_snapshot:
  stage: snapshot
  image: bitnami/kubectl:latest
  script:
    - kubectl apply -f deployments/eidos-agent/2-job.yaml
    - kubectl wait --for=condition=complete job/eidos -n gpu-operator
    - kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > snapshot.yaml
  artifacts:
    paths:
      - snapshot.yaml

generate_recipe:
  stage: recipe
  image: ghcr.io/mchmarny/eidos:latest
  script:
    # Option 1: Use ConfigMap directly (no artifact needed)
    - eidos recipe -f cm://gpu-operator/eidos-snapshot --intent training -o recipe.yaml
    # Option 2: Use snapshot file from previous stage
    # - eidos recipe --snapshot snapshot.yaml --intent training --output recipe.yaml
  artifacts:
    paths:
      - recipe.yaml
  dependencies:
    - capture_snapshot

create_bundle:
  stage: bundle
  image: ghcr.io/mchmarny/eidos:latest
  script:
    - eidos bundle --recipe recipe.yaml --bundlers gpu-operator --output ./bundles
  artifacts:
    paths:
      - bundles/
  dependencies:
    - generate_recipe

deploy_operators:
  stage: deploy
  image: bitnami/kubectl:latest
  script:
    - cd bundles/gpu-operator
    - sha256sum -c checksums.txt
    - chmod +x scripts/install.sh
    - ./scripts/install.sh
  dependencies:
    - create_bundle
  when: manual
```

### Pattern 3: API-Driven Recipe Generation

Use API for recipe generation without installing CLI.

**Use case:** Lightweight recipe generation in containers

```yaml
# CircleCI
version: 2.1

jobs:
  generate_recipe:
    docker:
      - image: cimg/base:2025.01
    steps:
      - run:
          name: Generate recipe via API
          command: |
            # Detect environment
            OS="ubuntu"
            GPU="h100"
            SERVICE="eks"
            
            # Generate recipe
            curl -s "https://cns.dgxc.io/v1/recipe?os=${OS}&gpu=${GPU}&service=${SERVICE}&intent=training" \
              -o recipe.json
            
            # Validate
            jq -e '.measurements | length > 0' recipe.json
      
      - persist_to_workspace:
          root: .
          paths:
            - recipe.json
  
  extract_versions:
    docker:
      - image: cimg/base:2025.01
    steps:
      - attach_workspace:
          at: .
      
      - run:
          name: Extract component versions
          command: |
            # GPU Operator version
            GPU_OP_VERSION=$(jq -r '.measurements[] | 
              select(.type=="K8s") | 
              .subtypes[] | 
              select(.subtype=="image") | 
              .data["gpu-operator"]' recipe.json)
            
            echo "GPU Operator: $GPU_OP_VERSION"
            
            # Save for deployment
            echo "export GPU_OP_VERSION=$GPU_OP_VERSION" >> $BASH_ENV

workflows:
  deploy:
    jobs:
      - generate_recipe
      - extract_versions:
          requires:
            - generate_recipe
```

### Pattern 4: Multi-Cluster Management

Deploy consistent configurations across multiple clusters.

**Use case:** Multi-region GPU clusters with unified configuration

```bash
#!/bin/bash
# multi-cluster-deploy.sh

# Define clusters
CLUSTERS=(
  "prod-us-east-1:eks:h100"
  "prod-eu-west-1:eks:h100"
  "staging-us-west-2:eks:gb200"
)

# Iterate clusters
for cluster_config in "${CLUSTERS[@]}"; do
  IFS=":" read -r CLUSTER SERVICE GPU <<< "$cluster_config"
  
  echo "Processing cluster: $CLUSTER"
  
  # Switch context
  kubectl config use-context "$CLUSTER"
  
  # Capture snapshot
  kubectl apply -f deployments/eidos-agent/2-job.yaml
  kubectl wait --for=condition=complete --timeout=300s job/eidos -n gpu-operator
  kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > "snapshot-${CLUSTER}.yaml"
  
  # Generate recipe (can use ConfigMap directly or file)
  # Option 1: Use ConfigMap
  eidos recipe -f "cm://gpu-operator/eidos-snapshot" --intent training -o "recipe-${CLUSTER}.yaml"
  # Option 2: Use saved file
  # eidos recipe --snapshot "snapshot-${CLUSTER}.yaml" --intent training --output "recipe-${CLUSTER}.yaml"
  
  # Create bundle
  eidos bundle \
    --recipe "recipe-${CLUSTER}.yaml" \
    --bundlers gpu-operator \
    --output "./bundles/${CLUSTER}"
  
  # Deploy (with approval)
  echo "Deploy to $CLUSTER? [y/N]"
  read -r response
  if [[ "$response" =~ ^[Yy]$ ]]; then
    cd "bundles/${CLUSTER}/gpu-operator"
    ./scripts/install.sh
    cd -
  fi
  
  # Clean up
  kubectl delete job eidos -n gpu-operator
done
```

## Terraform Integration

### Module: Eidos Agent Deployment

```hcl
# modules/eidos-agent/main.tf

resource "kubectl_manifest" "eidos_deps" {
  yaml_body = file("${path.module}/manifests/1-deps.yaml")
}

resource "kubectl_manifest" "eidos_job" {
  yaml_body = templatefile("${path.module}/manifests/2-job.yaml", {
    node_selector = var.node_selector
    tolerations   = var.tolerations
    image_version = var.image_version
  })
  
  depends_on = [kubectl_manifest.eidos_deps]
}

# Wait for job completion and get snapshot from ConfigMap
resource "null_resource" "wait_for_snapshot" {
  provisioner "local-exec" {
    command = <<-EOT
      kubectl wait --for=condition=complete \
        --timeout=300s job/eidos -n gpu-operator
      kubectl get configmap eidos-snapshot -n gpu-operator \
        -o jsonpath='{.data.snapshot\.yaml}' > ${var.snapshot_output}
    EOT
  }
  
  depends_on = [kubectl_manifest.eidos_job]
}

# Generate recipe (can use ConfigMap directly)
resource "null_resource" "generate_recipe" {
  provisioner "local-exec" {
    command = <<-EOT
      eidos recipe \
        -f cm://gpu-operator/eidos-snapshot \
        --intent ${var.workload_intent} \
        -o ${var.recipe_output}
    EOT
  }
  
  depends_on = [null_resource.wait_for_snapshot]
}

# variables.tf
variable "node_selector" {
  description = "Node selector for agent pod"
  type        = map(string)
  default     = { "nvidia.com/gpu.present" = "true" }
}

variable "tolerations" {
  description = "Tolerations for agent pod"
  type        = list(object({
    key    = string
    value  = string
    effect = string
  }))
  default = []
}

variable "image_version" {
  description = "Eidos image version"
  type        = string
  default     = "latest"
}

variable "snapshot_output" {
  description = "Path to save snapshot"
  type        = string
  default     = "snapshot.yaml"
}

variable "recipe_output" {
  description = "Path to save recipe"
  type        = string
  default     = "recipe.yaml"
}

variable "workload_intent" {
  description = "Workload intent: training or inference"
  type        = string
  default     = "training"
}

# outputs.tf
output "snapshot_file" {
  value = var.snapshot_output
}

output "recipe_file" {
  value = var.recipe_output
}
```

**Usage:**
```hcl
# main.tf
module "eidos_agent" {
  source = "./modules/eidos-agent"
  
  node_selector = {
    "nodeGroup" = "gpu-nodes"
  }
  
  tolerations = [{
    key    = "nvidia.com/gpu"
    value  = ""
    effect = "NoSchedule"
  }]
  
  workload_intent = "training"
  snapshot_output = "cluster-${var.environment}-snapshot.yaml"
  recipe_output   = "cluster-${var.environment}-recipe.yaml"
}
```

## Kubernetes Operators

### Custom Operator: Configuration Drift Watcher

```go
// Watch for configuration changes and reconcile
package main

import (
    "context"
    "fmt"
    "time"
    
    "k8s.io/client-go/kubernetes"
    ctrl "sigs.k8s.io/controller-runtime"
)

type ConfigReconciler struct {
    Client    kubernetes.Interface
    Namespace string
}

func (r *ConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Deploy Eidos agent
    if err := r.deployAgent(ctx); err != nil {
        return ctrl.Result{}, err
    }
    
    // 2. Wait for completion
    if err := r.waitForJob(ctx); err != nil {
        return ctrl.Result{}, err
    }
    
    // 3. Retrieve snapshot
    snapshot, err := r.getSnapshot(ctx)
    if err != nil {
        return ctrl.Result{}, err
    }
    
    // 4. Compare with baseline
    if r.hasConfigDrift(snapshot) {
        // Alert or auto-remediate
        fmt.Println("Configuration drift detected!")
    }
    
    // 5. Clean up
    if err := r.cleanupAgent(ctx); err != nil {
        return ctrl.Result{}, err
    }
    
    // Requeue after 6 hours
    return ctrl.Result{RequeueAfter: 6 * time.Hour}, nil
}

func (r *ConfigReconciler) deployAgent(ctx context.Context) error {
    // Apply RBAC and Job manifests
    return nil
}

func (r *ConfigReconciler) waitForJob(ctx context.Context) error {
    // Wait for job completion with timeout
    return nil
}

func (r *ConfigReconciler) getSnapshot(ctx context.Context) (string, error) {
    // Retrieve snapshot from ConfigMap
    return "", nil
}

func (r *ConfigReconciler) hasConfigDrift(snapshot string) bool {
    // Compare with baseline
    return false
}

func (r *ConfigReconciler) cleanupAgent(ctx context.Context) error {
    // Delete job
    return nil
}
```

## Monitoring and Alerting

### Prometheus Metrics

**Scrape Eidos API Server:**
```yaml
# prometheus-config.yaml
scrape_configs:
  - job_name: 'eidos-api-server'
    static_configs:
      - targets: ['eidos-api-server.default.svc.cluster.local:8080']
    metrics_path: /metrics
```

**Key metrics:**
```promql
# Request rate
rate(eidos_http_requests_total[5m])

# Error rate
rate(eidos_http_requests_total{status=~"5.."}[5m])

# Latency (p95)
histogram_quantile(0.95, 
  rate(eidos_http_request_duration_seconds_bucket[5m])
)

# Rate limit rejections
rate(eidos_rate_limit_rejects_total[5m])
```

### Alerting Rules

```yaml
# prometheus-rules.yaml
groups:
  - name: eidos_alerts
    interval: 30s
    rules:
      - alert: EidosHighErrorRate
        expr: |
          rate(eidos_http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Eidos API high error rate"
          description: "Error rate is {{ $value | humanizePercentage }}"
      
      - alert: EidosHighLatency
        expr: |
          histogram_quantile(0.95,
            rate(eidos_http_request_duration_seconds_bucket[5m])
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Eidos API high latency"
          description: "P95 latency is {{ $value }}s"
      
      - alert: EidosRateLimitHit
        expr: |
          rate(eidos_rate_limit_rejects_total[5m]) > 1
        for: 5m
        labels:
          severity: info
        annotations:
          summary: "Eidos API rate limit reached"
          description: "Rate limit rejections: {{ $value }}/s"
```

## Best Practices

### 1. Caching Recipes

API responses are cacheable (Cache-Control: max-age=300):

```python
import requests
from cachetools import TTLCache

# Cache recipes for 5 minutes
recipe_cache = TTLCache(maxsize=100, ttl=300)

def get_recipe_cached(params):
    cache_key = frozenset(params.items())
    
    if cache_key not in recipe_cache:
        response = requests.get('https://cns.dgxc.io/v1/recipe', params=params)
        recipe_cache[cache_key] = response.json()
    
    return recipe_cache[cache_key]
```

### 2. Error Handling and Retries

```python
import requests
from tenacity import retry, stop_after_attempt, wait_exponential

@retry(
    stop=stop_after_attempt(3),
    wait=wait_exponential(multiplier=1, min=4, max=10)
)
def get_recipe_with_retry(params):
    response = requests.get('https://cns.dgxc.io/v1/recipe', params=params)
    response.raise_for_status()
    return response.json()
```

### 3. Parallel Recipe Generation

```python
from concurrent.futures import ThreadPoolExecutor
import requests

def get_recipe(params):
    response = requests.get('https://cns.dgxc.io/v1/recipe', params=params)
    return response.json()

# Generate recipes for multiple environments in parallel
environments = [
    {'os': 'ubuntu', 'gpu': 'h100', 'service': 'eks'},
    {'os': 'ubuntu', 'gpu': 'gb200', 'service': 'gke'},
    {'os': 'rhel', 'gpu': 'a100', 'service': 'aks'},
]

with ThreadPoolExecutor(max_workers=3) as executor:
    recipes = list(executor.map(get_recipe, environments))
```

### 4. Structured Logging

```python
import logging
import json

# Configure structured logging
logging.basicConfig(
    level=logging.INFO,
    format='%(message)s'
)

def log_recipe_request(params, recipe, duration):
    logging.info(json.dumps({
        'event': 'recipe_generated',
        'params': params,
        'matched_rules': len(recipe.get('matchedRules', [])),
        'measurements': len(recipe.get('measurements', [])),
        'duration_ms': duration * 1000
    }))
```

### 5. Snapshot Versioning

```bash
#!/bin/bash
# Save snapshots with metadata

CLUSTER="prod-us-east-1"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
OUTPUT="snapshot-${CLUSTER}-${TIMESTAMP}.yaml"

# Capture snapshot from ConfigMap
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}' > "$OUTPUT"

# Add metadata
cat << EOF > "${OUTPUT}.meta"
cluster: $CLUSTER
timestamp: $TIMESTAMP
git_commit: $(git rev-parse HEAD)
k8s_version: $(kubectl version -o json | jq -r '.serverVersion.gitVersion')
EOF

# Upload to artifact storage
aws s3 cp "$OUTPUT" "s3://my-bucket/snapshots/"
aws s3 cp "${OUTPUT}.meta" "s3://my-bucket/snapshots/"
```

## Security Considerations

### API Key Management (Future)

```python
import os
import requests

API_KEY = os.environ.get('EIDOS_API_KEY')

headers = {
    'Authorization': f'Bearer {API_KEY}',
    'X-Request-Id': generate_uuid()
}

response = requests.get(
    'https://cns.dgxc.io/v1/recipe',
    params={'os': 'ubuntu', 'gpu': 'h100'},
    headers=headers
)
```

### Network Policies

Restrict Eidos agent network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: eidos-agent
  namespace: gpu-operator
spec:
  podSelector:
    matchLabels:
      job-name: eidos
  policyTypes:
    - Egress
  egress:
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443  # Kubernetes API
```

### Secrets Management

```yaml
# kubernetes-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: eidos-credentials
  namespace: gpu-operator
type: Opaque
stringData:
  api-key: your-api-key-here
```

```yaml
# Reference in pod
env:
  - name: EIDOS_API_KEY
    valueFrom:
      secretKeyRef:
        name: eidos-credentials
        key: api-key
```

## Troubleshooting

### Debug API Calls

```bash
# Verbose curl
curl -v "https://cns.dgxc.io/v1/recipe?os=ubuntu&gpu=h100"

# With timing
curl -w "\nTime: %{time_total}s\n" \
  "https://cns.dgxc.io/v1/recipe?os=ubuntu&gpu=h100"

# Check headers
curl -I "https://cns.dgxc.io/v1/recipe?os=ubuntu&gpu=h100"
```

### Validate Snapshots

```bash
# Check YAML syntax
yamllint snapshot.yaml

# Validate structure
yq eval '.measurements | length' snapshot.yaml

# Check for required measurements
yq eval '.measurements[] | .type' snapshot.yaml | sort -u
```

### Test Recipe Generation

```bash
# Generate and validate
eidos recipe --os ubuntu --gpu h100 --output recipe.yaml
yamllint recipe.yaml

# Check matched rules
yq eval '.matchedRules' recipe.yaml

# Extract GPU Operator version
yq eval '.measurements[] | 
  select(.type=="K8s") | 
  .subtypes[] | 
  select(.subtype=="image") | 
  .data["gpu-operator"]' recipe.yaml
```

## See Also

- [API Reference](api-reference.md) - API endpoint documentation
- [Data Flow](data-flow.md) - Understanding data architecture
- [Kubernetes Deployment](kubernetes-deployment.md) - Self-hosted API server
- [CLI Reference](../user-guide/cli-reference.md) - CLI commands
- [Agent Deployment](../user-guide/agent-deployment.md) - Kubernetes agent
