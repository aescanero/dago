# Deployment Guide

This guide covers deploying DA Orchestrator in various environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Local Development](#local-development)
- [Docker](#docker)
- [Docker Compose](#docker-compose)
- [Kubernetes](#kubernetes)
- [Production Considerations](#production-considerations)

## Prerequisites

### Required

- Go 1.21+ (for building from source)
- Redis 7.0+
- Docker (for containerized deployment)
- Kubernetes 1.24+ (for K8s deployment)

### Optional

- Helm 3.0+ (for Helm deployment)
- kubectl (for K8s management)

## Local Development

### Build from Source

```bash
# Clone repository
git clone https://github.com/aescanero/dago.git
cd dago

# Install dependencies
make deps

# Build binary
make build

# Run tests
make test
```

### Run Locally

```bash
# Start Redis (using Docker)
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Set environment variables
export REDIS_ADDR=localhost:6379
export LOG_LEVEL=debug

# Run the application
./dago

# Note: LLM_API_KEY is only needed for worker services (dago-node-executor, dago-node-router)
```

The service will be available at:
- HTTP API: http://localhost:8080
- gRPC API: localhost:9090
- Metrics: http://localhost:8080/metrics

## Docker

### Using Pre-built Image

```bash
docker pull aescanero/dago:latest

docker run -d \
  --name dago \
  -p 8080:8080 \
  -p 9090:9090 \
  -e REDIS_ADDR=redis:6379 \
  aescanero/dago:latest

# Note: LLM_API_KEY not needed - dago is a pure orchestrator
```

### Building Custom Image

```bash
# Build image
make docker-build

# Run image
docker run -d \
  --name dago \
  -p 8080:8080 \
  -p 9090:9090 \
  -e REDIS_ADDR=redis:6379 \
  aescanero/dago:latest
```

### Multi-platform Build

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t aescanero/dago:latest \
  -f deployments/docker/Dockerfile \
  --push \
  .
```

## Docker Compose

### Quick Start

```bash
# Create .env file
cat > .env <<EOF
LOG_LEVEL=info
EOF

# Start services
make docker-compose-up

# Note: docker-compose.yml may have LLM_API_KEY and WORKER_POOL_SIZE
# but these are not used by dago core - only by worker services

# View logs
docker-compose -f deployments/docker-compose.yml logs -f dago

# Stop services
make docker-compose-down
```

### Custom Configuration

Edit `deployments/docker-compose.yml`:

```yaml
version: '3.8'

services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes

  dago:
    image: aescanero/dago:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - REDIS_ADDR=redis:6379
      - LOG_LEVEL=${LOG_LEVEL:-info}
    depends_on:
      - redis
    restart: unless-stopped
    # Note: LLM_API_KEY and WORKER_POOL_SIZE removed - not used by dago core

volumes:
  redis-data:
```

## Kubernetes

### Prerequisites

```bash
# Verify kubectl is configured
kubectl cluster-info

# Create namespace
kubectl create namespace dago
```

### Using Helm Chart

#### Install

```bash
# Install with default values
helm install dago deployments/helm/dago \
  --namespace dago

# Install with custom values
helm install dago deployments/helm/dago \
  --namespace dago \
  --values my-values.yaml

# Note: llm.apiKey not needed - dago is pure orchestrator
```

#### Custom Values File

Create `my-values.yaml`:

```yaml
replicaCount: 3

image:
  repository: aescanero/dago
  tag: "v1.0.0"
  pullPolicy: IfNotPresent

resources:
  limits:
    cpu: 1000m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

redis:
  addr: redis-master.redis.svc.cluster.local:6379
  password: "redis-password"

# Note: llm and workers sections removed - not used by dago core
# Configure these in worker services (dago-node-executor, dago-node-router)

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: dago.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: dago-tls
      hosts:
        - dago.example.com
```

#### Upgrade

```bash
helm upgrade dago deployments/helm/dago \
  --namespace dago \
  --values my-values.yaml
```

#### Uninstall

```bash
helm uninstall dago --namespace dago
```

### Using kubectl (Manual)

#### Create ConfigMap

```bash
kubectl create configmap dago-config \
  --namespace dago \
  --from-literal=REDIS_ADDR=redis:6379 \
  --from-literal=LOG_LEVEL=info
```

#### Create Secret (Optional)

```bash
# Note: Secrets not needed for dago core
# Only create secrets for worker services (dago-node-executor, dago-node-router)
# which need LLM_API_KEY
```

#### Deploy Redis

```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: dago
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: dago
spec:
  selector:
    app: redis
  ports:
  - port: 6379
    targetPort: 6379
EOF
```

#### Deploy DA Orchestrator

```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dago
  namespace: dago
spec:
  replicas: 3
  selector:
    matchLabels:
      app: dago
  template:
    metadata:
      labels:
        app: dago
    spec:
      containers:
      - name: dago
        image: aescanero/dago:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: grpc
        envFrom:
        - configMapRef:
            name: dago-config
        # Note: No LLM_API_KEY secret needed - dago is pure orchestrator
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            cpu: 250m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 512Mi
---
apiVersion: v1
kind: Service
metadata:
  name: dago
  namespace: dago
spec:
  selector:
    app: dago
  ports:
  - port: 80
    targetPort: 8080
    name: http
  - port: 9090
    targetPort: 9090
    name: grpc
  type: LoadBalancer
EOF
```

## Production Considerations

### High Availability

#### Redis

For production, use Redis Cluster or Redis Sentinel:

```yaml
# Using Redis Operator
apiVersion: redis.redis.opstreelabs.in/v1beta1
kind: RedisCluster
metadata:
  name: dago-redis
spec:
  clusterSize: 3
  persistenceEnabled: true
```

Or use managed Redis:
- AWS ElastiCache
- Google Cloud Memorystore
- Azure Cache for Redis

#### DA Orchestrator

- Run multiple replicas (minimum 3)
- Enable horizontal pod autoscaling
- Use pod disruption budgets
- Deploy across multiple availability zones

### Monitoring

#### Prometheus

```yaml
# ServiceMonitor for Prometheus Operator
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: dago
  namespace: dago
spec:
  selector:
    matchLabels:
      app: dago
  endpoints:
  - port: http
    path: /metrics
```

#### Grafana Dashboard

Import the provided Grafana dashboard from `deployments/monitoring/grafana-dashboard.json`

Key metrics to monitor:
- Graph submission rate
- Event publishing latency
- Redis connection pool
- Event queue depth

Note: Node execution latency, worker utilization, and LLM API errors are monitored in worker services.

### Logging

#### Centralized Logging

Use Fluentd/Fluent Bit to collect logs:

```yaml
# Fluent Bit DaemonSet
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
  namespace: logging
data:
  fluent-bit.conf: |
    [INPUT]
        Name              tail
        Path              /var/log/containers/dago-*.log
        Parser            docker
        Tag               kube.*
```

#### Log Levels

Production: `LOG_LEVEL=info`
Debug: `LOG_LEVEL=debug`

### Security

#### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: dago-network-policy
  namespace: dago
spec:
  podSelector:
    matchLabels:
      app: dago
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # HTTPS for external APIs (if needed)
```

#### Pod Security

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: dago
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 1000
  containers:
  - name: dago
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
        - ALL
```

### Backup and Recovery

#### Redis Backup

```bash
# Create backup
kubectl exec -it redis-0 -n dago -- redis-cli SAVE
kubectl cp dago/redis-0:/data/dump.rdb ./backup/dump.rdb

# Restore backup
kubectl cp ./backup/dump.rdb dago/redis-0:/data/dump.rdb
kubectl exec -it redis-0 -n dago -- redis-cli FLUSHALL
kubectl delete pod redis-0 -n dago
```

### Scaling

#### Horizontal Scaling

```bash
# Scale manually
kubectl scale deployment dago --replicas=10 -n dago

# Enable autoscaling
kubectl autoscale deployment dago \
  --min=3 --max=20 \
  --cpu-percent=70 \
  -n dago
```

#### Vertical Scaling

Adjust resource requests and limits based on monitoring data.

### Performance Tuning

#### Scaling Strategy

- Scale dago orchestrator horizontally for high graph submission rates
- Scale worker services independently based on event processing needs
- Monitor event queue depth to determine if more workers are needed

Note: WORKER_POOL_SIZE is configured in worker services, not dago core.

#### Redis Connection Pool

Default settings should be sufficient for most use cases. Monitor connection metrics.

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n dago
kubectl describe pod dago-xxx -n dago
kubectl logs dago-xxx -n dago
```

### Check Service

```bash
kubectl get svc -n dago
kubectl describe svc dago -n dago
```

### Check Redis Connection

```bash
kubectl exec -it dago-xxx -n dago -- sh
# Inside pod
redis-cli -h redis ping
```

### Common Issues

**Pod CrashLoopBackOff**
- Check logs: `kubectl logs dago-xxx -n dago`
- Verify configuration and secrets

**Service Unavailable**
- Check ingress: `kubectl get ingress -n dago`
- Verify service endpoints: `kubectl get endpoints -n dago`

**High Latency**
- Check worker utilization metrics
- Scale up workers or replicas
- Verify Redis performance

## Support

For issues and questions:
- GitHub Issues: https://github.com/aescanero/dago/issues
- Documentation: https://disasterproject.com/docs
