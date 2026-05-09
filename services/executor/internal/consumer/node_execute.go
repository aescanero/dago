package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/aescanero/dago/services/executor/internal/handler"
)

// NodeExecuteConsumer subscribes to node.execute.requested and dispatches to handlers.
type NodeExecuteConsumer struct {
	consumer   ports.EventConsumer
	dispatcher *handler.Dispatcher
}

// NewNodeExecuteConsumer creates a NodeExecuteConsumer.
func NewNodeExecuteConsumer(consumer ports.EventConsumer, d *handler.Dispatcher) *NodeExecuteConsumer {
	return &NodeExecuteConsumer{consumer: consumer, dispatcher: d}
}

// Run blocks until ctx is cancelled, processing node.execute.requested events.
// ACK on success or non-retryable error; NACK on retryable error.
func (c *NodeExecuteConsumer) Run(ctx context.Context) error {
	if err := c.consumer.Subscribe(ctx, ports.ConsumeOptions{
		Stream:       domain.StreamNodeExecuteRequested,
		Group:        "executor-group",
		ConsumerName: "executor-1",
	}, c.handle); err != nil {
		return fmt.Errorf("consumer: subscribe: %w", err)
	}
	return nil
}

func (c *NodeExecuteConsumer) handle(ctx context.Context, evt domain.Event) error {
	var data handler.NodeExecuteRequestedData
	if err := json.Unmarshal(evt.Data, &data); err != nil {
		log.Printf("consumer: unmarshal node.execute.requested: %v", err)
		return nil // ACK malformed events so they don't block the queue
	}

	err := c.dispatcher.Dispatch(ctx, data)
	if err == nil {
		return nil // ACK
	}

	if errors.Is(err, domain.ErrRateLimited) || errors.Is(err, domain.ErrProviderUnavailable) {
		return fmt.Errorf("consumer: retryable: %w", err) // NACK — leave in PEL for retry
	}
	return nil // ACK — failure event already published; no retry benefit
}
