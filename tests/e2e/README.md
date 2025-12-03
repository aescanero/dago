# End-to-End Tests

End-to-end tests for DA Orchestrator.

## Prerequisites

- Docker & Docker Compose
- Go 1.21+
- LLM API key

## Running Tests

```bash
# Start the stack
docker-compose -f ../../deployments/docker-compose.yml up -d

# Run E2E tests
export LLM_API_KEY=your-key
go test -v ./tests/e2e/...

# Stop the stack
docker-compose -f ../../deployments/docker-compose.yml down
```

## Test Structure

- `api_test.go`: HTTP API end-to-end tests
- `graph_execution_test.go`: Graph execution flow tests
- `websocket_test.go`: WebSocket streaming tests

## Writing Tests

E2E tests should:
- Test complete user workflows
- Use real HTTP/gRPC clients
- Test against running service
- Validate full execution flow
