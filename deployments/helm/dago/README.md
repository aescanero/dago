# DA Orchestrator Helm Chart

This Helm chart deploys the DA Orchestrator on Kubernetes.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.0+
- Redis instance (can be deployed separately or use external service)

## Installation

### Basic Installation

```bash
helm install dago . \
  --set llm.apiKey=your-api-key
```

### With Custom Values

Create a `my-values.yaml` file:

```yaml
replicaCount: 3

redis:
  addr: "redis-master.redis.svc.cluster.local:6379"
  password: "your-redis-password"

llm:
  provider: "anthropic"
  apiKey: "your-api-key"

workers:
  poolSize: 10

resources:
  limits:
    cpu: 2000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 20

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: dago.example.com
      paths:
        - path: /
          pathType: Prefix
```

Install with custom values:

```bash
helm install dago . -f my-values.yaml
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `2` |
| `image.repository` | Image repository | `aescanero/dago` |
| `image.tag` | Image tag | `latest` |
| `redis.addr` | Redis address | `redis:6379` |
| `redis.password` | Redis password | `""` |
| `llm.provider` | LLM provider | `anthropic` |
| `llm.apiKey` | LLM API key | `""` (required) |
| `workers.poolSize` | Worker pool size | `5` |
| `autoscaling.enabled` | Enable HPA | `false` |
| `ingress.enabled` | Enable ingress | `false` |

## Upgrading

```bash
helm upgrade dago . -f my-values.yaml
```

## Uninstallation

```bash
helm uninstall dago
```
