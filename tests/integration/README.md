# Integration Tests

Integration tests for DA Orchestrator.

## Prerequisites

- Go 1.21+
- Redis running on localhost:6379
- LLM API key (for full tests)

## Running Tests

```bash
# Run all integration tests
go test -v ./tests/integration/...

# Run with Redis
docker run -d --name redis-test -p 6379:6379 redis:7-alpine
go test -v ./tests/integration/...
docker stop redis-test && docker rm redis-test
```

## Test Structure

- `redis_test.go`: Redis adapter integration tests
- `storage_test.go`: State storage integration tests
- `events_test.go`: Event bus integration tests

## Writing Tests

Integration tests should:
- Test real components (Redis, etc.)
- Clean up resources after tests
- Use test-specific namespaces/prefixes
- Be idempotent
