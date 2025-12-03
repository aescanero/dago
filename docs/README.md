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
- **Worker Pool**: Manages concurrent agent workers, health monitoring

### 3. Adapters Layer (`pkg/adapters/`)

Implements infrastructure concerns:

- **LLM Adapters**: Integration with LLM providers (Anthropic Claude)
- **Event Bus**: Redis Streams for event-driven communication
- **State Storage**: Redis for persistent state management
- **Metrics**: Prometheus metrics collection

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
│          Application Layer                        │
│  ┌─────────────────┐  ┌──────────────┐          │
│  │   Orchestrator  │  │ Worker Pool  │          │
│  │     Manager     │  │   Manager    │          │
│  └─────────────────┘  └──────────────┘          │
└──────┬──────────────────────┬────────────────────┘
       │                      │
┌──────▼──────────────────────▼────────────────────┐
│             Adapters Layer                        │
│  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐│
│  │  LLM   │  │ Events │  │Storage │  │Metrics ││
│  └────────┘  └────────┘  └────────┘  └────────┘│
└──────┬──────────┬────────────┬──────────┬───────┘
       │          │            │          │
       │      ┌───▼────────────▼──────────▼───┐
       │      │         Redis                  │
       │      └────────────────────────────────┘
       │
   ┌───▼──────┐
   │ Anthropic│
   └──────────┘
```

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

1. **Worker Pool** subscribes to execution events
2. **Worker** picks up ready node from event bus
3. **LLM Adapter** executes node with appropriate provider
4. **Orchestrator Manager** updates graph state based on results
5. **Event Bus** publishes node completion/failure events
6. **WebSocket** streams updates to connected clients
7. **Metrics** records execution statistics

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

# LLM Configuration
LLM_PROVIDER=anthropic        # LLM provider
LLM_API_KEY=sk-ant-...        # API key

# Worker Configuration
WORKER_POOL_SIZE=5            # Number of workers
```

### Redis Usage

For MVP simplicity, all infrastructure uses Redis:

- **Event Bus**: Redis Streams with consumer groups
- **State Storage**: JSON-serialized state with TTL
- **Cache**: Short-lived data caching

## Security Considerations

### API Security

- API keys for LLM providers stored in environment variables
- Consider adding API authentication for production
- Rate limiting should be implemented at reverse proxy level

### Network Security

- Use TLS for external communication
- Internal services can use plain TCP in trusted networks
- Redis should be password-protected in production

### Data Security

- Graph state may contain sensitive information
- Implement appropriate retention policies
- Consider encryption at rest for Redis

## Performance Tuning

### Worker Pool Sizing

- Start with 5 workers for MVP
- Monitor CPU and memory usage
- Scale horizontally by adding more instances

### Redis Optimization

- Single instance sufficient for MVP
- Consider Redis Cluster for high availability
- Monitor memory usage and set eviction policies

### Metrics to Monitor

- Graph submission rate
- Node execution latency
- Worker utilization
- Redis connection pool
- LLM API latency and errors

## Troubleshooting

### Common Issues

**Redis Connection Errors**
```
Error: dial tcp: connection refused
```
Solution: Ensure Redis is running and REDIS_ADDR is correct

**LLM API Errors**
```
Error: 401 Unauthorized
```
Solution: Check LLM_API_KEY is valid

**Worker Starvation**
```
Warning: All workers busy
```
Solution: Increase WORKER_POOL_SIZE or scale horizontally

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

- Design for failure - LLM calls can fail
- Implement retry logic with exponential backoff
- Use circuit breakers for external services

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

- Auto-scaling worker pools
- NATS for event bus (in addition to Redis)
- PostgreSQL for long-term state storage
- Multi-provider LLM support
- Graph versioning
- Execution replay
- Advanced monitoring dashboard

### Scalability

- Horizontal scaling with load balancer
- Redis Cluster for high availability
- Kubernetes-based deployment
- Multi-region support
