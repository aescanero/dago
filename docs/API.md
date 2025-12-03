# API Documentation

## Overview

The DA Orchestrator exposes three API interfaces:

1. **HTTP REST API**: Primary interface for graph management
2. **WebSocket API**: Real-time execution monitoring
3. **gRPC API**: High-performance service-to-service communication

## HTTP REST API

Base URL: `http://localhost:8080/api/v1`

### Endpoints

#### Submit Graph

Submit a new agent graph for execution.

```
POST /graphs
```

**Request Body:**
```json
{
  "graph": {
    "id": "example-graph",
    "version": "1.0.0",
    "nodes": [
      {
        "id": "node-1",
        "type": "agent",
        "config": {
          "model": "claude-3-5-sonnet-20241022",
          "system": "You are a helpful assistant",
          "temperature": 0.7
        },
        "dependencies": []
      }
    ]
  },
  "inputs": {
    "user_query": "Hello, world!"
  }
}
```

**Response:** `201 Created`
```json
{
  "graph_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "submitted",
  "submitted_at": "2025-12-02T10:30:00Z"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid graph structure
- `422 Unprocessable Entity`: Graph validation failed
- `500 Internal Server Error`: Server error

#### Get Graph Status

Retrieve the current status of a graph execution.

```
GET /graphs/{graph_id}
```

**Response:** `200 OK`
```json
{
  "graph_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "submitted_at": "2025-12-02T10:30:00Z",
  "started_at": "2025-12-02T10:30:01Z",
  "nodes": [
    {
      "id": "node-1",
      "status": "running",
      "started_at": "2025-12-02T10:30:01Z"
    }
  ]
}
```

**Status Values:**
- `submitted`: Graph accepted but not started
- `running`: Graph is executing
- `completed`: All nodes completed successfully
- `failed`: One or more nodes failed
- `cancelled`: Execution was cancelled

**Error Responses:**
- `404 Not Found`: Graph not found

#### Get Graph Result

Retrieve the execution results of a completed graph.

```
GET /graphs/{graph_id}/result
```

**Response:** `200 OK`
```json
{
  "graph_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "result": {
    "node-1": {
      "output": "Hello! How can I help you today?",
      "metadata": {
        "tokens": 42,
        "duration_ms": 1250
      }
    }
  },
  "completed_at": "2025-12-02T10:30:15Z"
}
```

**Error Responses:**
- `404 Not Found`: Graph not found
- `409 Conflict`: Graph not yet completed

#### Cancel Graph Execution

Cancel a running graph execution.

```
POST /graphs/{graph_id}/cancel
```

**Response:** `200 OK`
```json
{
  "graph_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "cancelled",
  "cancelled_at": "2025-12-02T10:30:10Z"
}
```

**Error Responses:**
- `404 Not Found`: Graph not found
- `409 Conflict`: Graph already completed or failed

#### List Graphs

List recent graph executions.

```
GET /graphs?limit=10&offset=0&status=running
```

**Query Parameters:**
- `limit`: Number of results (default: 20, max: 100)
- `offset`: Pagination offset (default: 0)
- `status`: Filter by status (optional)

**Response:** `200 OK`
```json
{
  "graphs": [
    {
      "graph_id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "running",
      "submitted_at": "2025-12-02T10:30:00Z"
    }
  ],
  "total": 1,
  "limit": 10,
  "offset": 0
}
```

#### Health Check

Check service health.

```
GET /health
```

**Response:** `200 OK`
```json
{
  "status": "healthy",
  "timestamp": "2025-12-02T10:30:00Z",
  "checks": {
    "redis": "ok",
    "workers": "ok"
  }
}
```

#### Metrics

Prometheus metrics endpoint.

```
GET /metrics
```

Returns Prometheus-formatted metrics.

## WebSocket API

Real-time updates for graph execution.

### Connect to Graph Stream

```
WS /graphs/{graph_id}/ws
```

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/graphs/550e8400/ws');

ws.onopen = () => {
  console.log('Connected');
};

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log('Update:', update);
};

ws.onerror = (error) => {
  console.error('Error:', error);
};

ws.onclose = () => {
  console.log('Disconnected');
};
```

**Message Format:**
```json
{
  "type": "node_started",
  "graph_id": "550e8400-e29b-41d4-a716-446655440000",
  "node_id": "node-1",
  "timestamp": "2025-12-02T10:30:01Z"
}
```

**Event Types:**
- `graph_started`: Graph execution began
- `node_started`: Node execution began
- `node_completed`: Node completed successfully
- `node_failed`: Node execution failed
- `graph_completed`: All nodes completed
- `graph_failed`: Graph execution failed

## gRPC API

High-performance API for service-to-service communication.

### Protocol Definition

```protobuf
syntax = "proto3";

package dago.v1;

service OrchestratorService {
  rpc SubmitGraph(SubmitGraphRequest) returns (SubmitGraphResponse);
  rpc GetGraphStatus(GetGraphStatusRequest) returns (GetGraphStatusResponse);
  rpc GetGraphResult(GetGraphResultRequest) returns (GetGraphResultResponse);
  rpc CancelGraph(CancelGraphRequest) returns (CancelGraphResponse);
  rpc StreamGraphEvents(StreamGraphEventsRequest) returns (stream GraphEvent);
}

message SubmitGraphRequest {
  string graph_json = 1;
  map<string, string> inputs = 2;
}

message SubmitGraphResponse {
  string graph_id = 1;
  string status = 2;
}

// ... additional messages
```

### Usage Example (Go)

```go
import (
    "context"
    pb "github.com/aescanero/dago/pkg/api/grpc/proto"
    "google.golang.org/grpc"
)

// Connect
conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewOrchestratorServiceClient(conn)

// Submit graph
resp, err := client.SubmitGraph(context.Background(), &pb.SubmitGraphRequest{
    GraphJson: graphJSON,
    Inputs: map[string]string{
        "user_query": "Hello",
    },
})
```

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Graph contains cycles",
    "details": {
      "cycle": ["node-1", "node-2", "node-1"]
    }
  }
}
```

### Error Codes

- `VALIDATION_ERROR`: Graph validation failed
- `NOT_FOUND`: Resource not found
- `ALREADY_EXISTS`: Resource already exists
- `INTERNAL_ERROR`: Internal server error
- `RATE_LIMIT_EXCEEDED`: Too many requests

## Rate Limiting

Currently not implemented in MVP. Recommend implementing at reverse proxy level:

```nginx
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

location /api/ {
    limit_req zone=api burst=20;
}
```

## Authentication

MVP does not include authentication. For production:

### Recommended: API Keys

```
GET /graphs
Authorization: Bearer <api-key>
```

### Recommended: OAuth 2.0

Use standard OAuth 2.0 Bearer tokens.

## Examples

### Submit and Monitor Graph

```bash
#!/bin/bash

# Submit graph
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/graphs \
  -H "Content-Type: application/json" \
  -d @graph.json)

GRAPH_ID=$(echo $RESPONSE | jq -r '.graph_id')
echo "Submitted graph: $GRAPH_ID"

# Poll for status
while true; do
  STATUS=$(curl -s http://localhost:8080/api/v1/graphs/$GRAPH_ID | jq -r '.status')
  echo "Status: $STATUS"

  if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then
    break
  fi

  sleep 2
done

# Get result
curl -s http://localhost:8080/api/v1/graphs/$GRAPH_ID/result | jq .
```

### WebSocket Monitoring

```html
<!DOCTYPE html>
<html>
<head>
    <title>Graph Monitor</title>
</head>
<body>
    <div id="status"></div>
    <div id="events"></div>

    <script>
        const graphId = '550e8400-e29b-41d4-a716-446655440000';
        const ws = new WebSocket(`ws://localhost:8080/api/v1/graphs/${graphId}/ws`);

        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            const div = document.getElementById('events');
            div.innerHTML += `<p>${data.type}: ${data.node_id}</p>`;
        };
    </script>
</body>
</html>
```

## Client Libraries

### Official Go Client

```go
import "github.com/aescanero/dago/pkg/client"

client := client.New("http://localhost:8080")
graphID, err := client.SubmitGraph(ctx, graph, inputs)
```

### Python Client (Community)

```python
from dago_client import DagoClient

client = DagoClient("http://localhost:8080")
graph_id = client.submit_graph(graph, inputs)
status = client.get_status(graph_id)
```

## Versioning

API version is included in the URL path: `/api/v1/`

Breaking changes will result in a new version: `/api/v2/`
