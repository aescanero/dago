package statemachine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

// nodeExecuteRequestedPayload is the data field of a node.execute.requested event.
type nodeExecuteRequestedPayload struct {
	ExecutionID string          `json:"execution_id"`
	GraphID     string          `json:"graph_id"`
	NodeID      string          `json:"node_id"`
	NodeKey     string          `json:"node_key"`
	Pattern     string          `json:"pattern"`
	Config      json.RawMessage `json:"config,omitempty"`
	Auth        string          `json:"auth,omitempty"`
}

// graphCompletedPayload is the data field of a graph.completed event.
type graphCompletedPayload struct {
	ExecutionID string `json:"execution_id"`
	GraphID     string `json:"graph_id"`
	DurationMs  int64  `json:"duration_ms"`
}

// graphFailedPayload is the data field of a graph.failed event.
type graphFailedPayload struct {
	ExecutionID string `json:"execution_id"`
	GraphID     string `json:"graph_id"`
	Error       string `json:"error"`
	ErrorCode   string `json:"error_code"`
}

// ExecutionStateMachine drives execution state transitions based on node outcomes.
// Idempotency is ensured via CanTransitionTo before every UpdateExecution call.
type ExecutionStateMachine struct {
	repo      ports.ExecutionRepository
	publisher ports.EventPublisher
}

// NewExecutionStateMachine builds an ExecutionStateMachine with its dependencies.
func NewExecutionStateMachine(repo ports.ExecutionRepository, publisher ports.EventPublisher) *ExecutionStateMachine {
	return &ExecutionStateMachine{repo: repo, publisher: publisher}
}

// HandleNodeExecuted processes a successful node execution result.
// If a sequential successor exists: publishes node.execute.requested and updates state.
// If the node is terminal: publishes graph.completed and marks the execution completed.
func (sm *ExecutionStateMachine) HandleNodeExecuted(
	ctx context.Context,
	exec *domain.Execution,
	graph domain.GraphDefinition,
	nodeKey string,
	output json.RawMessage,
	auth string,
) error {
	nextKey, err := NextNode(graph, nodeKey)
	if err != nil {
		return fmt.Errorf("statemachine.HandleNodeExecuted: NextNode: %w", err)
	}

	if nextKey != "" {
		// Intermediate node — advance to the next node.
		if !CanTransitionTo(exec.Status, domain.ExecutionStatusRunning) {
			return nil // idempotent — already transitioned
		}
		exec.CurrentNode = nextKey
		if err := sm.repo.UpdateExecution(ctx, exec); err != nil {
			return fmt.Errorf("statemachine.HandleNodeExecuted: UpdateExecution: %w", err)
		}

		nextNode := graph.Nodes[nextKey]
		data, err := json.Marshal(nodeExecuteRequestedPayload{
			ExecutionID: exec.ID.String(),
			GraphID:     exec.GraphID.String(),
			NodeID:      uuid.New().String(),
			NodeKey:     nextKey,
			Pattern:     nextNode.Pattern,
			Config:      nextNode.Config,
			Auth:        auth,
		})
		if err != nil {
			return fmt.Errorf("statemachine.HandleNodeExecuted: marshal payload: %w", err)
		}
		evt := newEvent(domain.EventTypeNodeExecuteRequested, "orchestrator", data)
		if err := sm.publisher.Publish(ctx, evt, ports.PublishOptions{Stream: domain.StreamNodeExecuteRequested}); err != nil {
			return fmt.Errorf("statemachine.HandleNodeExecuted: publish node.execute.requested: %w", err)
		}
		return nil
	}

	// Terminal node — mark the execution completed.
	if !CanTransitionTo(exec.Status, domain.ExecutionStatusCompleted) {
		return nil // idempotent
	}
	now := time.Now().UTC()
	exec.Status = domain.ExecutionStatusCompleted
	exec.CompletedAt = &now
	if err := sm.repo.UpdateExecution(ctx, exec); err != nil {
		return fmt.Errorf("statemachine.HandleNodeExecuted: UpdateExecution (completed): %w", err)
	}

	var durationMs int64
	if exec.StartedAt != nil {
		durationMs = now.Sub(*exec.StartedAt).Milliseconds()
	}
	data, err := json.Marshal(graphCompletedPayload{
		ExecutionID: exec.ID.String(),
		GraphID:     exec.GraphID.String(),
		DurationMs:  durationMs,
	})
	if err != nil {
		return fmt.Errorf("statemachine.HandleNodeExecuted: marshal completed payload: %w", err)
	}
	evt := newEvent(domain.EventTypeGraphCompleted, "orchestrator", data)
	if err := sm.publisher.Publish(ctx, evt, ports.PublishOptions{Stream: domain.StreamGraphCompleted}); err != nil {
		return fmt.Errorf("statemachine.HandleNodeExecuted: publish graph.completed: %w", err)
	}
	return nil
}

// HandleNodeExecuteFailed processes a failed node execution result.
// If !retryable: publishes graph.failed, marks execution failed, returns nil.
// If retryable: returns domain.ErrRetryable so the consumer NACKs.
func (sm *ExecutionStateMachine) HandleNodeExecuteFailed(
	ctx context.Context,
	exec *domain.Execution,
	retryable bool,
	errMsg, errCode string,
	auth string,
) error {
	if retryable {
		return domain.ErrRetryable
	}

	if !CanTransitionTo(exec.Status, domain.ExecutionStatusFailed) {
		return nil // idempotent
	}
	now := time.Now().UTC()
	exec.Status = domain.ExecutionStatusFailed
	exec.Error = errMsg
	exec.CompletedAt = &now
	if err := sm.repo.UpdateExecution(ctx, exec); err != nil {
		return fmt.Errorf("statemachine.HandleNodeExecuteFailed: UpdateExecution: %w", err)
	}

	data, err := json.Marshal(graphFailedPayload{
		ExecutionID: exec.ID.String(),
		GraphID:     exec.GraphID.String(),
		Error:       errMsg,
		ErrorCode:   errCode,
	})
	if err != nil {
		return fmt.Errorf("statemachine.HandleNodeExecuteFailed: marshal failed payload: %w", err)
	}
	evt := newEvent(domain.EventTypeGraphFailed, "orchestrator", data)
	if err := sm.publisher.Publish(ctx, evt, ports.PublishOptions{Stream: domain.StreamGraphFailed}); err != nil {
		return fmt.Errorf("statemachine.HandleNodeExecuteFailed: publish graph.failed: %w", err)
	}
	return nil
}

// CanTransitionTo reports whether a transition from current to next status is valid.
// Prevents double transitions if the consumer receives the same event twice.
func CanTransitionTo(current, next domain.ExecutionStatus) bool {
	switch next {
	case domain.ExecutionStatusRunning:
		return current == domain.ExecutionStatusPending || current == domain.ExecutionStatusRunning
	case domain.ExecutionStatusCompleted:
		return current == domain.ExecutionStatusRunning
	case domain.ExecutionStatusFailed:
		return current == domain.ExecutionStatusRunning
	case domain.ExecutionStatusPending, domain.ExecutionStatusCancelled, domain.ExecutionStatusInterrupted:
		return false
	default:
		return false
	}
}

func newEvent(eventType, source string, data json.RawMessage) domain.Event {
	return domain.Event{
		ID:              uuid.New().String(),
		Type:            eventType,
		Source:          source,
		SpecVersion:     "1.0",
		Time:            time.Now().UTC(),
		DataContentType: "application/json",
		Data:            data,
	}
}
