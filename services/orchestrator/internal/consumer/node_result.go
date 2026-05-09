package consumer

import (
	"context"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/aescanero/dago/services/orchestrator/internal/statemachine"
)

// NodeResultConsumer consumes node.executed and node.execute.failed events,
// delegating state transitions to ExecutionStateMachine.
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

// HandleNodeExecuted processes a node.executed event.
func (c *NodeResultConsumer) HandleNodeExecuted(ctx context.Context, evt domain.Event) error {
	panic("not implemented")
}

// HandleNodeExecuteFailed processes a node.execute.failed event.
func (c *NodeResultConsumer) HandleNodeExecuteFailed(ctx context.Context, evt domain.Event) error {
	panic("not implemented")
}
