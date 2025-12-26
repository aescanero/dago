# DA Orchestrator (dago)

Deep Agent Orchestrator - Core orchestration engine for the Disaster Project.

## Overview

`dago` is the core orchestration engine that implements the Deep Agent architecture. It provides:

- **Orchestration Manager**: Coordinates graph execution via event-driven architecture
- **Event Coordination**: Publishes work events to separate worker services via Redis Streams
- **Adapters**: Integrations for event buses, storage, and metrics (NO LLM - handled by workers)
- **APIs**: HTTP, WebSocket, and gRPC interfaces for graph submission and monitoring

**Note**: This is a **pure orchestrator** - it does NOT execute nodes or call LLMs. Node execution is handled by separate worker services (`dago-node-executor` and `dago-node-router`).

## Architecture

```
┌─────────────────────────────────────────────────┐
│                 API Layer                        │
│  (HTTP, WebSocket, gRPC)                        │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│           Application Layer                      │
│  (Orchestrator Manager, Worker Pools)           │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│             Adapters Layer                       │
│  (LLM, Events, Storage, Metrics)                │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│           Infrastructure                         │
│  (Redis, Prometheus, Anthropic API)             │
└─────────────────────────────────────────────────┘
```

Built on [dago-libs](https://github.com/aescanero/dago-libs) domain models and ports.

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/aescanero/dago.git
cd dago

# Set your LLM API key
export LLM_API_KEY=your-anthropic-api-key

# Start the stack
make docker-compose-up

# The API will be available at:
# - HTTP: http://localhost:8080
# - gRPC: localhost:9090
```

### Using Pre-built Docker Image

```bash
docker run -p 8080:8080 -p 9090:9090 \
  -e REDIS_ADDR=redis:6379 \
  -e LLM_API_KEY=your-api-key \
  aescanero/dago:latest
```

### Building from Source

```bash
# Install dependencies
make deps

# Run tests
make test

# Build binary
make build

# Run
export REDIS_ADDR=localhost:6379
export LLM_API_KEY=your-api-key
./dago
```

## Configuration

Configuration is done via environment variables:

| Variable          | Default          | Description                    |
|-------------------|------------------|--------------------------------|
| `DAGO_HTTP_PORT`  | `8080`           | HTTP API port                  |
| `DAGO_GRPC_PORT`  | `9090`           | gRPC API port                  |
| `REDIS_ADDR`      | `localhost:6379` | Redis server address           |
| `REDIS_PASS`      | (empty)          | Redis password                 |
| `REDIS_DB`        | `0`              | Redis database number          |
| `LLM_PROVIDER`    | `anthropic`      | LLM provider (anthropic)       |
| `LLM_API_KEY`     | (required)       | LLM API key                    |
| `WORKER_POOL_SIZE`| `5`              | Number of worker goroutines    |
| `LOG_LEVEL`       | `info`           | Log level (debug,info,warn,error)|

## API Usage

### Submit a Graph

```bash
curl -X POST http://localhost:8080/api/v1/graphs \
  -H "Content-Type: application/json" \
  -d @graph.json
```

### Get Graph Status

```bash
curl http://localhost:8080/api/v1/graphs/{graph-id}/status
```

### WebSocket Real-time Updates

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/graphs/{graph-id}/ws');
ws.onmessage = (event) => {
  console.log('Status update:', JSON.parse(event.data));
};
```

See [docs/API.md](docs/API.md) for complete API documentation.

## Deployment

### Kubernetes with Helm

```bash
# Install
helm install dago deployments/helm/dago \
  --set redis.addr=redis:6379 \
  --set llm.apiKey=your-api-key

# Upgrade
helm upgrade dago deployments/helm/dago

# Uninstall
helm uninstall dago
```

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for detailed deployment instructions.

## Development

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Redis (for local development)
- Make

### Project Structure

```
dago/
├── cmd/dago/              # Main application entry point
├── internal/              # Private application code
│   ├── application/       # Business logic (orchestration, workers)
│   └── config/           # Configuration management
├── pkg/                  # Public importable packages
│   ├── adapters/         # Infrastructure adapters
│   └── api/              # API implementations
├── deployments/          # Deployment configurations
│   ├── docker/           # Docker files
│   └── helm/             # Helm chart
├── docs/                 # Documentation
└── tests/                # Integration and E2E tests
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests (requires Redis)
go test ./tests/integration/...

# E2E tests
go test ./tests/e2e/...
```

### Code Style

- Follow Go best practices and idioms
- Use dependency injection
- Implement interfaces from dago-libs
- Include error context
- Use structured logging
- Write tests for all new code

## Documentation

- [Architecture & Usage](docs/README.md)
- [API Documentation](docs/API.md)
- [Deployment Guide](docs/DEPLOYMENT.md)
- [Changelog](docs/CHANGELOG.md)

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Links

- **Domain**: [disasterproject.com](https://disasterproject.com)
- **GitHub**: [github.com/aescanero/dago](https://github.com/aescanero/dago)
- **Docker Hub**: [aescanero/dago](https://hub.docker.com/r/aescanero/dago)
- **Dependencies**: [dago-libs](https://github.com/aescanero/dago-libs)
