package ports

import (
	"context"
	"time"

	"github.com/aescanero/dago/libs/domain"
)

// PublishOptions configure a single Publish call.
type PublishOptions struct {
	Stream string
}

// ConsumeOptions configure a Subscribe or RecoverPending call.
type ConsumeOptions struct {
	Stream       string
	Group        string
	ConsumerName string
	// BlockDuration is how long XREADGROUP blocks waiting for new messages.
	BlockDuration time.Duration
	// MaxRetries is the number of delivery attempts before a message is moved to DLQ.
	// Defaults to 3.
	MaxRetries int
}

// EventHandler is called for every message received by a consumer.
// Returning nil triggers XACK. Returning an error leaves the message in the PEL.
type EventHandler func(ctx context.Context, event domain.Event) error

// EventPublisher is the output port for publishing events to Valkey Streams.
type EventPublisher interface {
	Publish(ctx context.Context, event domain.Event, opts PublishOptions) error
	Close() error
}

// EventConsumer is the output port for consuming events from Valkey Streams.
type EventConsumer interface {
	// Subscribe blocks until ctx is cancelled, calling handler for each message.
	// Automatic XACK on nil return; no ACK on error (message stays pending).
	// After MaxRetries deliveries the message is moved to dago.dlq and ACKed.
	Subscribe(ctx context.Context, opts ConsumeOptions, handler EventHandler) error
	// RecoverPending claims messages idle longer than idleThreshold from the PEL
	// and processes them with the same ACK/DLQ semantics as Subscribe.
	RecoverPending(ctx context.Context, opts ConsumeOptions, idleThreshold time.Duration, handler EventHandler) error
	Close() error
}
