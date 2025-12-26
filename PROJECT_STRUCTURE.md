# DA Orchestrator - Complete Project Structure

This document describes the complete structure of the DA Orchestrator core repository.

## Repository Statistics

- **Total Files Created**: 47
- **Go Source Files**: 29
- **YAML Configuration Files**: 11
- **Documentation Files**: 5
- **Shell Scripts**: 2
- **Directories**: 37

## Complete Directory Structure

```
dago/
├── .github/
│   └── workflows/              # GitHub Actions CI/CD
│       ├── ci.yml             # Tests + lint
│       ├── release.yml        # Build binary + create release
│       └── docker.yml         # Build Docker image, push with DOCKER_TOKEN
│
├── docs/                      # Documentation
│   ├── README.md             # Architecture and usage
│   ├── API.md                # API documentation
│   ├── DEPLOYMENT.md         # Deployment guide
│   └── CHANGELOG.md          # Version history
│
├── cmd/
│   └── dago/
│       └── main.go           # Main entry point
│
├── internal/                  # Private code
│   ├── application/           # Use cases / orchestration
│   │   ├── orchestrator/
│   │   │   ├── manager.go    # Orchestrator manager (publishes events)
│   │   │   ├── validator.go # Graph validator
│   │   │   └── doc.go
│   │   # Note: workers/ directory does NOT exist - workers are separate services
│   │
│   └── config/
│       ├── config.go         # Configuration from env
│       └── doc.go
│
├── pkg/                       # Public (importable)
│   ├── adapters/
│   │   # Note: llm/ directory does NOT exist in dago core
│   │   # LLM adapters are in dago-adapters repo, used by worker services
│   │   ├── events/
│   │   │   ├── redis/
│   │   │   │   └── streams.go # Redis Streams implementation
│   │   │   ├── memory/
│   │   │   │   └── memory.go  # In-memory for tests
│   │   │   └── doc.go
│   │   ├── storage/
│   │   │   ├── redis/
│   │   │   │   └── redis.go   # Redis state storage
│   │   │   ├── memory/
│   │   │   │   └── memory.go  # In-memory for tests
│   │   │   └── doc.go
│   │   └── metrics/
│   │       ├── prometheus/
│   │       │   └── collector.go # Prometheus metrics
│   │       └── doc.go
│   │
│   └── api/
│       ├── http/
│       │   ├── server.go      # HTTP server
│       │   ├── handlers.go    # Request handlers
│       │   ├── middleware.go  # Middleware
│       │   └── doc.go
│       ├── websocket/
│       │   ├── handler.go     # WebSocket handler
│       │   └── doc.go
│       └── grpc/
│           ├── server.go      # gRPC server
│           ├── service.go     # Service implementation
│           └── doc.go
│
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile         # Multi-stage build
│   │   └── .dockerignore
│   ├── docker-compose.yml     # Local development stack
│   └── helm/
│       └── dago/              # Helm chart
│           ├── Chart.yaml
│           ├── values.yaml
│           ├── templates/
│           │   ├── deployment.yaml
│           │   ├── service.yaml
│           │   ├── secret.yaml
│           │   ├── serviceaccount.yaml
│           │   ├── hpa.yaml
│           │   └── _helpers.tpl
│           └── README.md
│
├── scripts/
│   ├── build.sh               # Build script
│   └── deploy.sh              # Deployment helper
│
├── tests/
│   ├── integration/
│   │   └── README.md
│   └── e2e/
│       └── README.md
│
├── .gitignore
├── .dockerignore
├── go.mod                     # Depends on dago-libs v1.0.0
├── go.sum
├── Makefile
├── README.md
├── LICENSE
└── PROJECT_STRUCTURE.md       # This file
```

## Key Features Implemented

### 1. Application Layer (`internal/application/`)

#### Orchestrator Manager
- Graph submission and validation
- Execution lifecycle management
- Publishes events to Redis Streams (executor.work, router.work)
- Listens for completion events (node.completed)
- Timeout handling
- Status queries
- Cancellation support

**Note**: Worker pools do NOT exist in dago core - they run as separate services (dago-node-executor, dago-node-router).

### 2. Adapters Layer (`pkg/adapters/`)

**Note**: LLM adapters do NOT exist in dago core - they are in dago-adapters repo and used by worker services.

#### Event Bus
- Redis Streams with consumer groups
- In-memory implementation for testing
- Pub/sub pattern for event coordination

#### State Storage
- Redis with JSON serialization
- TTL support for old states
- In-memory implementation for testing

#### Metrics
- Prometheus metrics collection
- Graph submission and event publishing metrics
- Event queue depth metrics

### 3. API Layer (`pkg/api/`)

#### HTTP REST API
- Graph submission: `POST /api/v1/graphs`
- Status queries: `GET /api/v1/graphs/:id/status`
- Result retrieval: `GET /api/v1/graphs/:id/result`
- Cancellation: `POST /api/v1/graphs/:id/cancel`
- Health check: `GET /health`
- Metrics: `GET /metrics`

#### WebSocket API
- Real-time event streaming
- Graph execution updates

#### gRPC API
- Placeholder for future implementation
- High-performance RPC interface

### 4. Configuration (`internal/config/`)

Environment-based configuration:
- Server ports (HTTP, gRPC)
- Redis connection
- Timeouts and intervals
- Log level

**Note**: LLM provider settings and worker pool size are configured in worker services, not in dago core.

### 5. Deployment

#### Docker
- Multi-stage build for optimized image size
- Alpine-based runtime
- Non-root user
- Health checks

#### Docker Compose
- Complete local development stack
- Redis + DA Orchestrator
- Environment variable configuration

#### Helm Chart
- Kubernetes deployment
- ConfigMap and Secret management
- Horizontal Pod Autoscaler support
- Ingress support
- Service Account

### 6. CI/CD (GitHub Actions)

#### CI Workflow
- Go tests with coverage
- Linting with golangci-lint
- Binary build verification

#### Release Workflow
- Multi-platform binary builds
- GitHub release creation
- Checksums generation

#### Docker Workflow
- Multi-platform image builds (amd64, arm64)
- Push to Docker Hub
- Tag management

## Dependencies

### External Dependencies
- **dago-libs v1.0.0**: Domain models and port interfaces
- **Redis Go Client**: Event bus and storage
- **Gin**: HTTP framework
- **Gorilla WebSocket**: WebSocket support
- **gRPC**: RPC framework
- **Prometheus Client**: Metrics
- **Zap**: Structured logging

**Note**: Anthropic SDK is a dependency of worker services, not dago core.

### Infrastructure Requirements
- **Redis 7.0+**: Event bus and state storage

**Note**: LLM API access is required by worker services, not dago core.

## MVP Simplifications

For MVP, the following simplifications were made:

1. **Redis for Everything**: Events, storage, cache (single infrastructure)
2. **Basic gRPC**: Minimal implementation, HTTP is primary
3. **No Advanced Features**: No graph versioning, replay, or multi-region
4. **Separate Services**: Workers run as separate services, not embedded in dago core

**Note**: LLM provider selection and worker pool scaling are concerns of the worker services.

## Development Workflow

### Local Development
```bash
# Start Redis
docker-compose -f deployments/docker-compose.yml up redis -d

# Set environment variables
export REDIS_ADDR=localhost:6379
# Note: LLM_API_KEY not needed for dago core

# Build and run
make build
./dago
```

### Docker Development
```bash
# Build image
make docker-build

# Run with docker-compose (note: docker-compose.yml may have LLM_API_KEY but it's not used)
make docker-compose-up
```

### Kubernetes Deployment
```bash
# Using Helm
helm install dago deployments/helm/dago \
  --set redis.addr=redis:6379
# Note: llm.apiKey not needed - dago is pure orchestrator
```

## Testing

### Unit Tests
```bash
make test
```

### Integration Tests
```bash
# Requires Redis
go test ./tests/integration/...
```

### E2E Tests
```bash
# Requires full stack
make docker-compose-up
go test ./tests/e2e/...
```

## Next Steps

### Post-MVP Features
1. NATS event bus option
2. PostgreSQL for long-term storage
3. Graph versioning
4. Execution replay
5. Advanced monitoring dashboard
6. API authentication/authorization

**Note**: Additional LLM providers and auto-scaling worker pools are features of worker services.

### Scalability Improvements
1. Horizontal scaling with load balancer
2. Redis Cluster for high availability
3. Multi-region support
4. Independent scaling of orchestrator vs. worker services

## Architecture Principles

The repository follows Clean Architecture:

1. **Separation of Concerns**: Clear boundaries between layers
2. **Dependency Inversion**: Depend on interfaces, not implementations
3. **Testability**: In-memory implementations for testing
4. **Extensibility**: Easy to add new adapters
5. **MVP Focus**: Simple, working solution first

## Documentation

- **README.md**: Quick start and overview
- **docs/README.md**: Architecture and usage details
- **docs/API.md**: Complete API documentation
- **docs/DEPLOYMENT.md**: Deployment guide
- **docs/CHANGELOG.md**: Version history

## License

MIT License - see LICENSE file for details.

## Links

- **Domain**: https://disasterproject.com
- **GitHub**: https://github.com/aescanero/dago
- **Docker Hub**: https://hub.docker.com/r/aescanero/dago
- **Dependencies**: https://github.com/aescanero/dago-libs
