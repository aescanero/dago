package statemachine

import (
	"context"
	"encoding/json"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

// ExecutionStateMachine drives execution state transitions based on node outcomes.
type ExecutionStateMachine struct {
	repo      ports.ExecutionRepository
	publisher ports.EventPublisher
}

// NewExecutionStateMachine builds an ExecutionStateMachine with its dependencies.
func NewExecutionStateMachine(repo ports.ExecutionRepository, publisher ports.EventPublisher) *ExecutionStateMachine {
	return &ExecutionStateMachine{repo: repo, publisher: publisher}
}

// HandleNodeExecuted processes a successful node execution event.
// If a sequential successor exists, publishes node.execute.requested and updates state.
// If the node is terminal, publishes graph.completed and marks execution completed.
func (sm *ExecutionStateMachine) HandleNodeExecuted(
	ctx context.Context,
	exec *domain.Execution,
	graph domain.GraphDefinition,
	nodeKey string,
	output json.RawMessage,
	auth string,
) error {
	panic("not implemented")
}

// HandleNodeExecuteFailed processes a failed node execution event.
// If !retryable: publishes graph.failed, marks execution failed, returns nil.
// If retryable: returns domain.ErrRetryable so the consumer NACKs.
func (sm *ExecutionStateMachine) HandleNodeExecuteFailed(
	ctx context.Context,
	exec *domain.Execution,
	retryable bool,
	errMsg, errCode string,
	auth string,
) error {
	panic("not implemented")
}
