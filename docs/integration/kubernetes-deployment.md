# Kubernetes Deployment

Deploy the CNS API Server in your Kubernetes cluster for self-hosted recipe generation.

## Overview

**API Server deployment** enables self-hosted recipe generation:

- Isolated deployment: Recipe data stays within your infrastructure
- Custom recipes: Modify embedded recipe data (`data-v1.yaml`)
- High availability: Deploy multiple replicas with load balancing
- Observability: Prometheus `/metrics` endpoint and structured logging
- Public instance: https://cns.dgxc.io (managed deployment)

**API Server scope:**

- Recipe generation from query parameters (query mode)
- Does not capture snapshots (use agent Job or CLI)
- Does not generate bundles (use CLI)
- Does not analyze snapshots (query mode only)

**Agent deployment** (separate component):

- Kubernetes Job captures cluster configuration
- Writes snapshot to ConfigMap via Kubernetes API
- Requires RBAC: ServiceAccount with ConfigMap create/update permissions
- See [Agent Deployment](../user-guide/agent-deployment.md)

**Typical workflow:**

1. Deploy agent Job → Captures snapshot → Writes to ConfigMap
2. CLI reads ConfigMap → Generates recipe → Writes to file or ConfigMap
3. CLI reads recipe → Generates bundle → Writes to filesystem
4. Apply bundle to cluster (Helm install, kubectl apply)

## Quick Start

### Deploy with Kustomize

```shell
# Create namespace
kubectl create namespace cns

# Deploy API server
kubectl apply -k https://github.com/NVIDIA/cloud-native-stack/deployments/cnsd

# Check deployment
kubectl get pods -n cns
kubectl get svc -n cns
```

### Deploy with Helm

**Status**: Helm chart not yet available. Use Kustomize or manual deployment.

<!-- Uncomment when Helm chart is published
```shell
helm repo add cns https://nvidia.github.io/cloud-native-stack
helm install cnsd cns/cnsd -n cns --create-namespace
```
-->

## Manual Deployment

### 1. Create Namespace

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: cns
  labels:
    app: cnsd
```

```shell
kubectl apply -f namespace.yaml
```

### 2. Create Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cnsd
  namespace: cns
  labels:
    app: cnsd
spec:
  replicas: 3
  selector:
    matchLabels:
      app: cnsd
  template:
    metadata:
      labels:
        app: cnsd
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
        fsGroup: 65532
      
      containers:
        - name: api-server
          image: ghcr.io/nvidia/cnsd:latest
          imagePullPolicy: IfNotPresent
          
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          
          env:
            - name: PORT
              value: "8080"
            - name: LOG_LEVEL
              value: "info"
          
          livenessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 10
            periodSeconds: 30
            timeoutSeconds: 5
            failureThreshold: 3
          
          readinessProbe:
            httpGet:
              path: /ready
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
          
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop: ["ALL"]
```

```shell
kubectl apply -f deployment.yaml
```

### 3. Create Service

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: cnsd
  namespace: cns
  labels:
    app: cnsd
spec:
  type: ClusterIP
  selector:
    app: cnsd
  ports:
    - name: http
      port: 80
      targetPort: http
      protocol: TCP
```

```shell
kubectl apply -f service.yaml
```

### 4. Create Ingress (Optional)

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cnsd
  namespace: cns
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - cns.yourdomain.com
      secretName: cns-tls
  rules:
    - host: cns.yourdomain.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: cnsd
                port:
                  number: 80
```

```shell
kubectl apply -f ingress.yaml
```

## Agent Deployment

Deploy the CNS Agent as a Kubernetes Job to automatically capture cluster configuration.

### 1. Create RBAC Resources

```yaml
# agent-rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cns
  namespace: gpu-operator
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: cns
  namespace: gpu-operator
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "create", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cns
  namespace: gpu-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: cns
subjects:
- kind: ServiceAccount
  name: cns
  namespace: gpu-operator  # Must match ServiceAccount namespace
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cns
rules:
- apiGroups: [""]
  resources: ["nodes", "pods"]
  verbs: ["get", "list"]
- apiGroups: ["nvidia.com"]
  resources: ["clusterpolicies"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cns
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cns
subjects:
- kind: ServiceAccount
  name: cns
  namespace: gpu-operator
```

```shell
kubectl apply -f agent-rbac.yaml
```

### 2. Create Agent Job

```yaml
# agent-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: cns
  namespace: gpu-operator
  labels:
    app: cns-agent
spec:
  template:
    metadata:
      labels:
        app: cns-agent
    spec:
      serviceAccountName: cns
      restartPolicy: Never
      
      containers:
      - name: cns
        image: ghcr.io/nvidia/cns:latest
        imagePullPolicy: IfNotPresent
        
        command:
        - cns
        - snapshot
        - --output
        - cm://gpu-operator/cns-snapshot
        
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 65532
          capabilities:
            drop: ["ALL"]
```

```shell
kubectl apply -f agent-job.yaml

# Wait for completion
kubectl wait --for=condition=complete job/cns -n gpu-operator --timeout=5m

# Verify ConfigMap was created
kubectl get configmap cns-snapshot -n gpu-operator

# View snapshot data
kubectl get configmap cns-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}'
```

### 3. Generate Recipe from ConfigMap

```bash
# Using CLI (local or in another Job)
cnsctl recipe --snapshot cm://gpu-operator/cns-snapshot \
             --intent training \
             --output recipe.yaml

# Or write recipe back to ConfigMap
cnsctl recipe --snapshot cm://gpu-operator/cns-snapshot \
             --intent training \
             --output cm://gpu-operator/cns-recipe
```

### 4. Generate Bundle

```bash
# From file
cnsctl bundle --recipe recipe.yaml --output ./bundles

# From ConfigMap
cnsctl bundle --recipe cm://gpu-operator/cns-recipe --output ./bundles
```

### E2E Testing

Validate the complete workflow with the e2e script:

```bash
# Test full workflow: agent → snapshot → recipe → bundle
./tools/e2e -s examples/snapshots/h100.yaml \
           -r examples/recipes/h100-eks-ubuntu-training.yaml \
           -b examples/bundles/h100-eks-ubuntu-training

# Just test agent deployment and snapshot capture
./tools/e2e -s snapshot.yaml

# Test recipe and bundle generation from ConfigMap
./tools/e2e -r recipe.yaml -b ./bundles
```

The e2e script:
- Deploys agent Job with RBAC
- Waits for snapshot to be written to ConfigMap
- Optionally generates recipe and bundle
- Validates each step completes successfully
- Preserves resources on failure for debugging

## Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | HTTP server port |
| `LOG_LEVEL` | info | Logging level: debug, info, warn, error |
| `RATE_LIMIT` | 100 | Requests per second |
| `RATE_BURST` | 200 | Burst capacity |
| `READ_TIMEOUT` | 30s | HTTP read timeout |
| `WRITE_TIMEOUT` | 30s | HTTP write timeout |
| `IDLE_TIMEOUT` | 60s | HTTP idle timeout |

**Note:** The API server uses structured JSON logging to stderr. The CLI supports three logging modes (CLI/Text/JSON), but the API server always uses JSON for consistent log aggregation.

### ConfigMap for Custom Recipe Data (Advanced)

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cns-recipe-data
  namespace: cns
data:
  data-v1.yaml: |
    # Your custom recipe data
    base:
      - type: K8s
        subtypes: [...]
    overlays: [...]
```

Mount in deployment:
```yaml
spec:
  template:
    spec:
      volumes:
        - name: recipe-data
          configMap:
            name: cns-recipe-data
      containers:
        - name: api-server
          volumeMounts:
            - name: recipe-data
              mountPath: /data
          env:
            - name: RECIPE_DATA_PATH
              value: /data/data-v1.yaml
```

## High Availability

### Horizontal Pod Autoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: cnsd
  namespace: cns
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: cnsd
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 50
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 15
```

```shell
kubectl apply -f hpa.yaml
```

### Pod Disruption Budget

```yaml
# pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: cnsd
  namespace: cns
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: cnsd
```

```shell
kubectl apply -f pdb.yaml
```

## Monitoring

### Prometheus ServiceMonitor

```yaml
# servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: cnsd
  namespace: cns
  labels:
    app: cnsd
spec:
  selector:
    matchLabels:
      app: cnsd
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
      scrapeTimeout: 10s
```

```shell
kubectl apply -f servicemonitor.yaml
```

### Grafana Dashboard

Import dashboard JSON from `docs/monitoring/grafana-dashboard.json`:

**Key panels:**
- Request rate (by status code)
- Request duration (p50, p95, p99)
- Error rate
- Rate limit rejections
- Active connections

## Security

### Network Policies

```yaml
# networkpolicy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: cnsd
  namespace: cns
spec:
  podSelector:
    matchLabels:
      app: cnsd
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 53  # DNS
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
      ports:
        - protocol: TCP
          port: 443  # Kubernetes API
```

### Pod Security Standards

```yaml
# Add to namespace
apiVersion: v1
kind: Namespace
metadata:
  name: cns
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### RBAC (If API server needs K8s access)

```yaml
# serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cnsd
  namespace: cns

---
# role.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cnsd
rules:
  - apiGroups: [""]
    resources: ["nodes", "pods"]
    verbs: ["get", "list"]

---
# rolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cnsd
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cnsd
subjects:
  - kind: ServiceAccount
    name: cnsd
    namespace: cns
```

## Troubleshooting

### Check Pod Status

```shell
# Pod status
kubectl get pods -n cns

# Describe pod
kubectl describe pod -n cns -l app=cnsd

# View logs
kubectl logs -n cns -l app=cnsd

# Follow logs
kubectl logs -n cns -l app=cnsd -f
```

### Check Service

```shell
# Service status
kubectl get svc -n cns

# Endpoints
kubectl get endpoints -n cns

# Test from within cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://cnsd.cns.svc.cluster.local/health
```

### Check Ingress

```shell
# Ingress status
kubectl get ingress -n cns

# Describe ingress
kubectl describe ingress cnsd -n cns

# Check cert-manager certificate
kubectl get certificate -n cns
```

### Performance Issues

```shell
# Check resource usage
kubectl top pods -n cns

# Check HPA status
kubectl get hpa -n cns

# Check metrics
kubectl exec -n cns -it deploy/cnsd -- \
  wget -qO- http://localhost:8080/metrics
```

### Connection Refused

1. Check service exists: `kubectl get svc -n cns`
2. Check endpoints: `kubectl get endpoints -n cns`
3. Check pod is ready: `kubectl get pods -n cns`
4. Check readiness probe: `kubectl describe pod -n cns <pod-name>`

### Rate Limiting

Check rate limit settings:
```shell
kubectl exec -n cns deploy/cnsd -- env | grep RATE
```

Adjust via deployment:
```yaml
env:
  - name: RATE_LIMIT
    value: "200"  # Increase limit
  - name: RATE_BURST
    value: "400"
```

## Upgrading

### Rolling Update

```shell
# Update image
kubectl set image deployment/cnsd \
  api-server=ghcr.io/nvidia/cnsd:v0.8.0 \
  -n cns

# Watch rollout
kubectl rollout status deployment/cnsd -n cns

# Rollback if needed
kubectl rollout undo deployment/cnsd -n cns
```

### Blue-Green Deployment

```shell
# Deploy new version
kubectl apply -f deployment-v2.yaml

# Switch service
kubectl patch service cnsd -n cns \
  -p '{"spec":{"selector":{"version":"v2"}}}'

# Delete old deployment
kubectl delete deployment cnsd-v1 -n cns
```

## Backup and Disaster Recovery

### Export Configuration

```shell
# Export all resources
kubectl get all -n cns -o yaml > cns-backup.yaml

# Export specific resources
kubectl get deployment,service,ingress -n cns -o yaml > cns-config.yaml
```

### Restore from Backup

```shell
# Restore namespace and resources
kubectl apply -f cns-backup.yaml
```

## Cost Optimization

### Resource Limits

Start with minimal resources:
```yaml
resources:
  requests:
    cpu: 50m
    memory: 64Mi
  limits:
    cpu: 200m
    memory: 256Mi
```

Monitor and adjust based on usage.

### Vertical Pod Autoscaler (Optional)

```yaml
# vpa.yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: cnsd
  namespace: cns
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: cnsd
  updatePolicy:
    updateMode: "Auto"
```

## See Also

- [API Reference](api-reference.md) - API endpoint documentation
- [Automation](automation.md) - CI/CD integration
- [Data Flow](data-flow.md) - Understanding data architecture
- [Architecture: API Server](../architecture/api-server.md) - Internal architecture
