package handler

import (
	"context"
	"encoding/json"

	"github.com/aescanero/dago/libs/ports"
)

// NodeExecuteRequestedData is the payload of a node.execute.requested event.
type NodeExecuteRequestedData struct {
	ExecutionID string          `json:"execution_id"`
	GraphID     string          `json:"graph_id"`
	NodeID      string          `json:"node_id"`
	NodeKey     string          `json:"node_key"`
	Pattern     string          `json:"pattern"`
	Config      json.RawMessage `json:"config"`
	Variables   map[string]any  `json:"variables"`
	Messages    []ports.Message `json:"messages"`
	Auth        string          `json:"auth"`
}

// NodeExecutedData is the payload of a node.executed event.
type NodeExecutedData struct {
	ExecutionID     string          `json:"execution_id"`
	GraphID         string          `json:"graph_id"`
	NodeID          string          `json:"node_id"`
	NodeKey         string          `json:"node_key"`
	Output          json.RawMessage `json:"output"`
	VariablesUpdate json.RawMessage `json:"variables_update"`
	DurationMs      int64           `json:"duration_ms"`
}

// NodeExecuteFailedData is the payload of a node.execute.failed event.
type NodeExecuteFailedData struct {
	ExecutionID string `json:"execution_id"`
	GraphID     string `json:"graph_id"`
	NodeID      string `json:"node_id"`
	NodeKey     string `json:"node_key"`
	Error       string `json:"error"`
	ErrorCode   string `json:"error_code"`
	Retryable   bool   `json:"retryable"`
}

// NodeHandler processes a single node execution request.
type NodeHandler interface {
	Handle(ctx context.Context, data NodeExecuteRequestedData) error
}
