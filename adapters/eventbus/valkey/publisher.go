package valkey

import (
	"context"
	"fmt"

	"github.com/valkey-io/valkey-go"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

// Publisher implements ports.EventPublisher using Valkey Streams.
type Publisher struct {
	client valkey.Client
}

// NewPublisher creates a Publisher connected to the given Valkey address.
func NewPublisher(addr string) (*Publisher, error) {
	c, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("eventbus new publisher: %w", err)
	}
	return &Publisher{client: c}, nil
}

// Publish serializes evt and appends it to the stream specified in opts.
func (p *Publisher) Publish(ctx context.Context, evt domain.Event, opts ports.PublishOptions) error {
	fields, err := marshalEnvelope(evt)
	if err != nil {
		return err
	}
	envelopeJSON := fmt.Sprintf("%v", fields[envelopeField])
	cmd := p.client.B().Xadd().Key(opts.Stream).Id("*").
		FieldValue().FieldValue(envelopeField, envelopeJSON).
		Build()
	if err := p.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("eventbus publish %s: %w", opts.Stream, err)
	}
	return nil
}

// Close releases the underlying Valkey connection.
func (p *Publisher) Close() error {
	p.client.Close()
	return nil
}

// ensureStream creates the stream and consumer group if they do not exist.
// Ignores BUSYGROUP error (group already exists).
func ensureStream(ctx context.Context, client valkey.Client, stream, group string) error {
	cmd := client.B().XgroupCreate().Key(stream).Group(group).Id("$").Mkstream().Build()
	if err := client.Do(ctx, cmd).Error(); err != nil && !valkey.IsValkeyBusyGroup(err) {
		return fmt.Errorf("eventbus ensure stream %s group %s: %w", stream, group, err)
	}
	return nil
}
