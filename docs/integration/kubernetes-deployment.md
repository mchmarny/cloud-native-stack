# Kubernetes Deployment

Deploy the Eidos API Server in your Kubernetes cluster for self-hosted recipe generation.

## Overview

**API Server Deployment** provides:
- **Private deployment** - Keep recipe data within your infrastructure
- **Custom configuration** - Modify recipe data for your environment
- **High availability** - Multiple replicas with load balancing
- **Monitoring integration** - Prometheus metrics and health checks
- **Production URL**: https://cns.dgxc.io (public instance)

**Agent Deployment** (separate from API server):
- **Automated snapshots** - Kubernetes Job captures cluster configuration
- **ConfigMap storage** - Snapshots written directly to ConfigMap (no volumes)
- **RBAC-secured** - ServiceAccount with ConfigMap read/write permissions
- **Pipeline-friendly** - Integrates with CI/CD for automated auditing
- See [agent-deployment.md](../user-guide/agent-deployment.md) for details

**Complete Workflow**:
1. Deploy Agent Job → Captures snapshot → Writes to ConfigMap
2. CLI reads from ConfigMap → Generates recipe → Creates bundle
3. Deploy bundle → Optimized GPU infrastructure

## Quick Start

### Deploy with Kustomize

```shell
# Create namespace
kubectl create namespace eidos

# Deploy API server
kubectl apply -k https://github.com/NVIDIA/cloud-native-stack/deployments/eidos-api-server

# Check deployment
kubectl get pods -n eidos
kubectl get svc -n eidos
```

### Deploy with Helm (Coming Soon)

```shell
helm repo add eidos https://nvidia.github.io/cloud-native-stack
helm install eidos-api-server eidos/eidos-api-server -n eidos --create-namespace
```

## Manual Deployment

### 1. Create Namespace

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: eidos
  labels:
    app: eidos-api-server
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
  name: eidos-api-server
  namespace: eidos
  labels:
    app: eidos-api-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: eidos-api-server
  template:
    metadata:
      labels:
        app: eidos-api-server
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
          image: ghcr.io/mchmarny/eidos-api-server:latest
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
  name: eidos-api-server
  namespace: eidos
  labels:
    app: eidos-api-server
spec:
  type: ClusterIP
  selector:
    app: eidos-api-server
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
  name: eidos-api-server
  namespace: eidos
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - eidos.yourdomain.com
      secretName: eidos-tls
  rules:
    - host: eidos.yourdomain.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: eidos-api-server
                port:
                  number: 80
```

```shell
kubectl apply -f ingress.yaml
```

## Agent Deployment

Deploy the Eidos Agent as a Kubernetes Job to automatically capture cluster configuration.

### 1. Create RBAC Resources

```yaml
# agent-rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: eidos
  namespace: gpu-operator
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: eidos
  namespace: gpu-operator
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "create", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: eidos
  namespace: gpu-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: eidos
subjects:
- kind: ServiceAccount
  name: eidos
  namespace: gpu-operator  # Must match ServiceAccount namespace
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: eidos
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
  name: eidos
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: eidos
subjects:
- kind: ServiceAccount
  name: eidos
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
  name: eidos
  namespace: gpu-operator
  labels:
    app: eidos-agent
spec:
  template:
    metadata:
      labels:
        app: eidos-agent
    spec:
      serviceAccountName: eidos
      restartPolicy: Never
      
      containers:
      - name: eidos
        image: ghcr.io/nvidia/eidos:latest
        imagePullPolicy: IfNotPresent
        
        command:
        - eidos
        - snapshot
        - --output
        - cm://gpu-operator/eidos-snapshot
        
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
kubectl wait --for=condition=complete job/eidos -n gpu-operator --timeout=5m

# Verify ConfigMap was created
kubectl get configmap eidos-snapshot -n gpu-operator

# View snapshot data
kubectl get configmap eidos-snapshot -n gpu-operator -o jsonpath='{.data.snapshot\.yaml}'
```

### 3. Generate Recipe from ConfigMap

```bash
# Using CLI (local or in another Job)
eidos recipe --snapshot cm://gpu-operator/eidos-snapshot \
             --intent training \
             --output recipe.yaml

# Or write recipe back to ConfigMap
eidos recipe --snapshot cm://gpu-operator/eidos-snapshot \
             --intent training \
             --output cm://gpu-operator/eidos-recipe
```

### 4. Generate Bundle

```bash
# From file
eidos bundle --recipe recipe.yaml --output ./bundles

# From ConfigMap
eidos bundle --recipe cm://gpu-operator/eidos-recipe --output ./bundles
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
| `LOG_FORMAT` | json | Log format: json, text |
| `RATE_LIMIT` | 100 | Requests per second |
| `RATE_BURST` | 200 | Burst capacity |
| `READ_TIMEOUT` | 30s | HTTP read timeout |
| `WRITE_TIMEOUT` | 30s | HTTP write timeout |
| `IDLE_TIMEOUT` | 60s | HTTP idle timeout |

### ConfigMap for Custom Recipe Data (Advanced)

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: eidos-recipe-data
  namespace: eidos
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
            name: eidos-recipe-data
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
  name: eidos-api-server
  namespace: eidos
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: eidos-api-server
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
  name: eidos-api-server
  namespace: eidos
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: eidos-api-server
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
  name: eidos-api-server
  namespace: eidos
  labels:
    app: eidos-api-server
spec:
  selector:
    matchLabels:
      app: eidos-api-server
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
  name: eidos-api-server
  namespace: eidos
spec:
  podSelector:
    matchLabels:
      app: eidos-api-server
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
  name: eidos
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
  name: eidos-api-server
  namespace: eidos

---
# role.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: eidos-api-server
rules:
  - apiGroups: [""]
    resources: ["nodes", "pods"]
    verbs: ["get", "list"]

---
# rolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: eidos-api-server
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: eidos-api-server
subjects:
  - kind: ServiceAccount
    name: eidos-api-server
    namespace: eidos
```

## Troubleshooting

### Check Pod Status

```shell
# Pod status
kubectl get pods -n eidos

# Describe pod
kubectl describe pod -n eidos -l app=eidos-api-server

# View logs
kubectl logs -n eidos -l app=eidos-api-server

# Follow logs
kubectl logs -n eidos -l app=eidos-api-server -f
```

### Check Service

```shell
# Service status
kubectl get svc -n eidos

# Endpoints
kubectl get endpoints -n eidos

# Test from within cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://eidos-api-server.eidos.svc.cluster.local/health
```

### Check Ingress

```shell
# Ingress status
kubectl get ingress -n eidos

# Describe ingress
kubectl describe ingress eidos-api-server -n eidos

# Check cert-manager certificate
kubectl get certificate -n eidos
```

### Performance Issues

```shell
# Check resource usage
kubectl top pods -n eidos

# Check HPA status
kubectl get hpa -n eidos

# Check metrics
kubectl exec -n eidos -it deploy/eidos-api-server -- \
  wget -qO- http://localhost:8080/metrics
```

### Connection Refused

1. Check service exists: `kubectl get svc -n eidos`
2. Check endpoints: `kubectl get endpoints -n eidos`
3. Check pod is ready: `kubectl get pods -n eidos`
4. Check readiness probe: `kubectl describe pod -n eidos <pod-name>`

### Rate Limiting

Check rate limit settings:
```shell
kubectl exec -n eidos deploy/eidos-api-server -- env | grep RATE
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
kubectl set image deployment/eidos-api-server \
  api-server=ghcr.io/mchmarny/eidos-api-server:v0.8.0 \
  -n eidos

# Watch rollout
kubectl rollout status deployment/eidos-api-server -n eidos

# Rollback if needed
kubectl rollout undo deployment/eidos-api-server -n eidos
```

### Blue-Green Deployment

```shell
# Deploy new version
kubectl apply -f deployment-v2.yaml

# Switch service
kubectl patch service eidos-api-server -n eidos \
  -p '{"spec":{"selector":{"version":"v2"}}}'

# Delete old deployment
kubectl delete deployment eidos-api-server-v1 -n eidos
```

## Backup and Disaster Recovery

### Export Configuration

```shell
# Export all resources
kubectl get all -n eidos -o yaml > eidos-backup.yaml

# Export specific resources
kubectl get deployment,service,ingress -n eidos -o yaml > eidos-config.yaml
```

### Restore from Backup

```shell
# Restore namespace and resources
kubectl apply -f eidos-backup.yaml
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
  name: eidos-api-server
  namespace: eidos
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: eidos-api-server
  updatePolicy:
    updateMode: "Auto"
```

## See Also

- [API Reference](api-reference.md) - API endpoint documentation
- [Automation](automation.md) - CI/CD integration
- [Data Flow](data-flow.md) - Understanding data architecture
- [Architecture: API Server](../architecture/api-server.md) - Internal architecture
