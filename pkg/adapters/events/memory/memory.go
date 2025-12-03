package memory

import (
	"context"
	"sync"

	"github.com/aescanero/dago-libs/pkg/ports"
)

// InMemoryEventBus implements EventBus using in-memory handlers
// This is for testing purposes only
type InMemoryEventBus struct {
	subscribers map[string][]ports.EventHandler
	mu          sync.RWMutex
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		subscribers: make(map[string][]ports.EventHandler),
	}
}

// Publish publishes an event to all subscribers of a topic
func (e *InMemoryEventBus) Publish(ctx context.Context, topic string, event ports.Event) error {
	e.mu.RLock()
	handlers := make([]ports.EventHandler, len(e.subscribers[topic]))
	copy(handlers, e.subscribers[topic])
	e.mu.RUnlock()

	// Call all handlers asynchronously
	for _, handler := range handlers {
		go func(h ports.EventHandler) {
			if err := h(ctx, event); err != nil {
				// Silently ignore handler errors in MVP
			}
		}(handler)
	}

	return nil
}

// Subscribe subscribes to events on a specific topic
func (e *InMemoryEventBus) Subscribe(ctx context.Context, topic string, handler ports.EventHandler) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.subscribers[topic] = append(e.subscribers[topic], handler)

	// Start a goroutine to clean up subscription on context cancellation
	go func() {
		<-ctx.Done()
		e.unsubscribe(topic, handler)
	}()

	return nil
}

// Unsubscribe removes all subscriptions from a topic
func (e *InMemoryEventBus) Unsubscribe(ctx context.Context, topic string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.subscribers, topic)
	return nil
}

// Close closes the event bus and cleans up resources
func (e *InMemoryEventBus) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Clear all subscribers
	e.subscribers = make(map[string][]ports.EventHandler)
	return nil
}

// unsubscribe removes a handler from a topic
func (e *InMemoryEventBus) unsubscribe(topic string, handler ports.EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	handlers := e.subscribers[topic]
	for i, h := range handlers {
		// Compare function pointers (not perfect but works for MVP)
		if &h == &handler {
			e.subscribers[topic] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}
