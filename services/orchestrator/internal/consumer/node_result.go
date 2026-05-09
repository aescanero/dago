package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/aescanero/dago/services/orchestrator/internal/statemachine"
)

// NodeResultConsumer consumes node.executed and node.execute.failed events,
// loading execution and graph state and delegating to ExecutionStateMachine.
type NodeResultConsumer struct {
	execRepo  ports.ExecutionRepository
	graphRepo ports.GraphRepository
	sm        *statemachine.ExecutionStateMachine
}

// NewNodeResultConsumer builds a NodeResultConsumer.
func NewNodeResultConsumer(
	execRepo ports.ExecutionRepository,
	graphRepo ports.GraphRepository,
	sm *statemachine.ExecutionStateMachine,
) *NodeResultConsumer {
	return &NodeResultConsumer{execRepo: execRepo, graphRepo: graphRepo, sm: sm}
}

// nodeExecutedData is the payload of a node.executed event.
type nodeExecutedData struct {
	ExecutionID string          `json:"execution_id"`
	GraphID     string          `json:"graph_id"`
	NodeKey     string          `json:"node_key"`
	Output      json.RawMessage `json:"output"`
}

// nodeExecuteFailedData is the payload of a node.execute.failed event.
type nodeExecuteFailedData struct {
	ExecutionID string `json:"execution_id"`
	NodeKey     string `json:"node_key"`
	Error       string `json:"error"`
	ErrorCode   string `json:"error_code"`
	Retryable   bool   `json:"retryable"`
}

// HandleNodeExecuted processes a node.executed event.
// Returns nil to ACK; returns error to NACK (retryable failure).
func (c *NodeResultConsumer) HandleNodeExecuted(ctx context.Context, evt domain.Event) error {
	var data nodeExecutedData
	if err := json.Unmarshal(evt.Data, &data); err != nil {
		log.Printf("consumer: unmarshal node.executed: %v", err)
		return nil // ACK malformed events
	}

	exec, graph, err := c.load(ctx, data.ExecutionID)
	if err != nil {
		return fmt.Errorf("consumer.HandleNodeExecuted: %w", err)
	}

	var auth string
	if evt.Auth != nil {
		auth = evt.Auth.Sub
	}
	if err := c.sm.HandleNodeExecuted(ctx, exec, *graph, data.NodeKey, data.Output, auth); err != nil {
		if errors.Is(err, domain.ErrRetryable) {
			return err // NACK
		}
		log.Printf("consumer: HandleNodeExecuted state machine error: %v", err)
		return nil // ACK — non-retryable internal error
	}
	return nil
}

// HandleNodeExecuteFailed processes a node.execute.failed event.
// Returns nil to ACK; returns domain.ErrRetryable to NACK.
func (c *NodeResultConsumer) HandleNodeExecuteFailed(ctx context.Context, evt domain.Event) error {
	var data nodeExecuteFailedData
	if err := json.Unmarshal(evt.Data, &data); err != nil {
		log.Printf("consumer: unmarshal node.execute.failed: %v", err)
		return nil // ACK malformed events
	}

	exec, _, err := c.load(ctx, data.ExecutionID)
	if err != nil {
		return fmt.Errorf("consumer.HandleNodeExecuteFailed: %w", err)
	}

	var auth string
	if evt.Auth != nil {
		auth = evt.Auth.Sub
	}
	if err := c.sm.HandleNodeExecuteFailed(ctx, exec, data.Retryable, data.Error, data.ErrorCode, auth); err != nil {
		if errors.Is(err, domain.ErrRetryable) {
			return err // NACK
		}
		log.Printf("consumer: HandleNodeExecuteFailed state machine error: %v", err)
	}
	return nil
}

// load fetches the Execution and its GraphDefinition by execution ID string.
func (c *NodeResultConsumer) load(ctx context.Context, executionIDStr string) (*domain.Execution, *domain.GraphDefinition, error) {
	execID, err := uuid.Parse(executionIDStr)
	if err != nil {
		return nil, nil, fmt.Errorf("parse execution_id %q: %w", executionIDStr, err)
	}
	exec, err := c.execRepo.FindByID(ctx, execID)
	if err != nil {
		return nil, nil, fmt.Errorf("find execution: %w", err)
	}
	g, err := c.graphRepo.FindByID(ctx, exec.GraphID)
	if err != nil {
		return nil, nil, fmt.Errorf("find graph: %w", err)
	}
	var graphDef domain.GraphDefinition
	if err := json.Unmarshal(g.Definition, &graphDef); err != nil {
		return nil, nil, fmt.Errorf("unmarshal graph definition: %w", err)
	}
	return exec, &graphDef, nil
}
