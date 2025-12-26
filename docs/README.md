# DA Orchestrator - Architecture & Usage

## Architecture Overview

The DA Orchestrator follows Clean Architecture principles with three main layers:

### 1. API Layer (`pkg/api/`)

Exposes interfaces for external communication:

- **HTTP API**: RESTful endpoints for graph submission and status queries
- **WebSocket**: Real-time event streaming for execution monitoring
- **gRPC**: High-performance RPC interface for service-to-service communication

### 2. Application Layer (`internal/application/`)

Contains the business logic:

- **Orchestrator Manager**: Coordinates graph execution, manages lifecycle
- **Validator**: Validates graph structure and dependencies

**Note**: Worker pools are NOT in dago core - they run as separate services (`dago-node-executor` and `dago-node-router`).

### 3. Adapters Layer (`pkg/adapters/`)

Implements infrastructure concerns:

- **Event Bus**: Redis Streams for event-driven communication
- **State Storage**: Redis for persistent state management
- **Metrics**: Prometheus metrics collection

**Note**: LLM adapters are in the separate `dago-adapters` repository and used by worker services, NOT by dago core.

## Component Interactions

```
┌────────────┐
│   Client   │
└──────┬─────┘
       │ HTTP/WS/gRPC
┌──────▼──────────────────────────────────────────┐
│                API Layer                         │
│  ┌──────────┐  ┌───────────┐  ┌─────────┐     │
│  │   HTTP   │  │ WebSocket │  │  gRPC   │     │
│  └──────────┘  └───────────┘  └─────────┘     │
└──────┬───────────────────────────────────────────┘
       │
┌──────▼───────────────────────────────────────────┐
│          Application Layer (dago core)            │
│  ┌─────────────────┐                             │
│  │   Orchestrator  │  Publishes events:          │
│  │     Manager     │  - executor.work            │
│  │                 │  - router.work              │
│  └─────────────────┘                             │
└──────┬──────────────────────────────────────────┘
       │
┌──────▼──────────────────────────────────────────┐
│             Adapters Layer                       │
│  ┌────────┐  ┌────────┐  ┌────────┐            │
│  │ Events │  │Storage │  │Metrics │            │
│  └────────┘  └────────┘  └────────┘            │
└──────┬────────────┬──────────┬──────────────────┘
       │            │          │
   ┌───▼────────────▼──────────▼───┐
   │         Redis Streams          │
   │    (Event Bus + Storage)       │
   └───┬────────────────────────┬───┘
       │                        │
┌──────▼────────┐      ┌────────▼────────┐
│ executor      │      │ router          │
│ workers       │      │ workers         │
│ (separate     │      │ (separate       │
│  service)     │      │  service)       │
│               │      │                 │
│ Uses LLM      │      │ Uses LLM        │
└───────────────┘      └─────────────────┘
```

**IMPORTANT**: dago core is a pure orchestrator - it does NOT execute nodes or call LLMs. Worker services subscribe to Redis Streams events and handle execution.

## Data Flow

### Graph Submission Flow

1. **Client** submits graph via HTTP POST `/api/v1/graphs`
2. **HTTP Handler** validates request format
3. **Validator** validates graph structure, cycles, dependencies
4. **Orchestrator Manager** accepts graph and assigns ID
5. **Event Bus** publishes `GraphSubmitted` event
6. **State Storage** persists initial graph state
7. **Response** returns graph ID and status to client

### Execution Flow

1. **Orchestrator Manager** publishes `executor.work` or `router.work` events to Redis Streams
2. **Worker services** (dago-node-executor/router) subscribe to execution events
3. **Workers** pick up events and execute nodes (using LLM when needed)
4. **Workers** publish `node.completed` events back to Redis Streams
5. **Orchestrator Manager** receives completion events and updates graph state
6. **WebSocket** streams updates to connected clients
7. **Metrics** records execution statistics

**Note**: Steps 2-4 happen in separate worker services, NOT in dago core.

## Configuration

### Environment Variables

```bash
# Server Configuration
DAGO_HTTP_PORT=8080           # HTTP API port
DAGO_GRPC_PORT=9090           # gRPC API port
LOG_LEVEL=info                # Log level

# Redis Configuration
REDIS_ADDR=localhost:6379     # Redis address
REDIS_PASS=                   # Redis password (optional)
REDIS_DB=0                    # Redis database number
```

**Note**: LLM_PROVIDER, LLM_API_KEY, and WORKER_POOL_SIZE are configured in the worker services (dago-node-executor and dago-node-router), NOT in dago core.

### Redis Usage

For MVP simplicity, all infrastructure uses Redis:

- **Event Bus**: Redis Streams with consumer groups
- **State Storage**: JSON-serialized state with TTL
- **Cache**: Short-lived data caching

## Security Considerations

### API Security

- Consider adding API authentication for production
- Rate limiting should be implemented at reverse proxy level

**Note**: LLM API keys are configured in worker services, not in dago core.

### Network Security

- Use TLS for external communication
- Internal services can use plain TCP in trusted networks
- Redis should be password-protected in production

### Data Security

- Graph state may contain sensitive information
- Implement appropriate retention policies
- Consider encryption at rest for Redis

## Performance Tuning

### Scaling

- Scale horizontally by adding more dago orchestrator instances
- Scale worker services independently based on workload
- Monitor CPU and memory usage

### Redis Optimization

- Single instance sufficient for MVP
- Consider Redis Cluster for high availability
- Monitor memory usage and set eviction policies

### Metrics to Monitor

- Graph submission rate
- Event publishing latency
- Redis connection pool
- Event processing backlog

**Note**: Worker utilization, node execution latency, and LLM API metrics are monitored in the worker services.

## Troubleshooting

### Common Issues

**Redis Connection Errors**
```
Error: dial tcp: connection refused
```
Solution: Ensure Redis is running and REDIS_ADDR is correct

**Event Backlog**
```
Warning: Event queue growing
```
Solution: Scale up worker services to process events faster

### Debug Mode

Enable debug logging:
```bash
LOG_LEVEL=debug ./dago
```

### Health Checks

- HTTP: `GET /health`
- Metrics: `GET /metrics`

## Best Practices

### Graph Design

- Keep graphs small and focused
- Avoid deep nesting (> 5 levels)
- Use parallelism where possible
- Set appropriate timeouts

### Error Handling

- Design for failure - worker services may fail
- Implement retry logic with exponential backoff
- Use circuit breakers for external services

**Note**: LLM call failures are handled in worker services.

### Monitoring

- Track success/failure rates
- Monitor execution times
- Set up alerts for anomalies
- Use distributed tracing for debugging

## Development Workflow

### Local Development

```bash
# Start Redis
docker-compose -f deployments/docker-compose.yml up redis -d

# Run tests
make test

# Build and run
make run
```

### Testing

- Unit tests: Test individual components
- Integration tests: Test with real Redis
- E2E tests: Test complete flows

### Debugging

Use delve for debugging:
```bash
dlv debug ./cmd/dago
```

## Future Enhancements

### Post-MVP Features

- NATS for event bus (in addition to Redis)
- PostgreSQL for long-term state storage
- Graph versioning
- Execution replay
- Advanced monitoring dashboard

**Note**: Auto-scaling worker pools and multi-provider LLM support are features of the worker services.

### Scalability

- Horizontal scaling with load balancer
- Redis Cluster for high availability
- Kubernetes-based deployment
- Multi-region support
