package testutil

import (
	"context"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

// noopPublisher is a no-op EventPublisher for tests that don't need to verify events.
type noopPublisher struct{}

func (noopPublisher) Publish(_ context.Context, _ domain.Event, _ ports.PublishOptions) error {
	return nil
}

func (noopPublisher) Close() error { return nil }

// compile-time interface check
var _ ports.EventPublisher = noopPublisher{}
