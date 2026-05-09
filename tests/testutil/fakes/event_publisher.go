package fakes

import (
	"context"
	"sync"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

// InMemoryPublisher records all published events in memory for assertions in tests.
type InMemoryPublisher struct {
	mu     sync.RWMutex
	events []publishedEvent
}

type publishedEvent struct {
	Event  domain.Event
	Stream string
}

// NewInMemoryPublisher creates an InMemoryPublisher.
func NewInMemoryPublisher() *InMemoryPublisher {
	return &InMemoryPublisher{}
}

// Publish records the event and its target stream.
func (p *InMemoryPublisher) Publish(_ context.Context, event domain.Event, opts ports.PublishOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, publishedEvent{Event: event, Stream: opts.Stream})
	return nil
}

// Close is a no-op for the in-memory publisher.
func (p *InMemoryPublisher) Close() error { return nil }

// Events returns a snapshot of all published events.
func (p *InMemoryPublisher) Events() []domain.Event {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]domain.Event, len(p.events))
	for i, e := range p.events {
		out[i] = e.Event
	}
	return out
}

// StreamEvents returns all events published to a specific stream.
func (p *InMemoryPublisher) StreamEvents(stream string) []domain.Event {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []domain.Event
	for _, e := range p.events {
		if e.Stream == stream {
			out = append(out, e.Event)
		}
	}
	return out
}

// compile-time interface check
var _ ports.EventPublisher = (*InMemoryPublisher)(nil)
