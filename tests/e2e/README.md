# End-to-End Tests

End-to-end tests for DA Orchestrator.

## Prerequisites

- Docker & Docker Compose
- Go 1.21+
- Worker services running (dago-node-executor, dago-node-router)

## Running Tests

```bash
# Start the stack (dago orchestrator + Redis)
docker-compose -f ../../deployments/docker-compose.yml up -d

# Note: You'll also need to start worker services separately
# Worker services need LLM_API_KEY configured

# Run E2E tests
go test -v ./tests/e2e/...

# Stop the stack
docker-compose -f ../../deployments/docker-compose.yml down
```

**Note**: dago core doesn't need LLM_API_KEY - only worker services do.

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
