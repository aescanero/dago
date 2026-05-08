package valkey

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

const dlqStream = domain.EventTypeDLQ

// Consumer implements ports.EventConsumer using Valkey Streams consumer groups.
type Consumer struct {
	client    valkey.Client
	publisher *Publisher
}

// NewConsumer creates a Consumer connected to the given Valkey address.
func NewConsumer(addr string) (*Consumer, error) {
	c, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("eventbus new consumer: %w", err)
	}
	pub, err := NewPublisher(addr)
	if err != nil {
		c.Close()
		return nil, err
	}
	return &Consumer{client: c, publisher: pub}, nil
}

// Subscribe blocks until ctx is cancelled, calling handler for each delivered message.
// On nil return: XACK. On error return: leaves in PEL (retried next poll).
// After MaxRetries deliveries the message is moved to dago.dlq and ACKed.
func (c *Consumer) Subscribe(ctx context.Context, opts ports.ConsumeOptions, handler ports.EventHandler) error {
	maxRetries := normalizeMaxRetries(opts.MaxRetries)
	blockMs := normalizeBlockMs(opts.BlockDuration)

	if err := ensureStream(ctx, c.client, opts.Stream, opts.Group); err != nil {
		return err
	}

	for {
		if ctx.Err() != nil {
			return nil
		}
		if err := c.poll(ctx, opts, blockMs, maxRetries, handler); err != nil && ctx.Err() != nil {
			return nil
		}
	}
}

// poll reads one batch from XREADGROUP and processes each entry.
func (c *Consumer) poll(
	ctx context.Context,
	opts ports.ConsumeOptions,
	blockMs, maxRetries int64,
	handler ports.EventHandler,
) error {
	entries, err := c.readGroup(ctx, opts.Stream, opts.Group, opts.ConsumerName, blockMs, 10)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return nil //nolint:nilerr // cancellation is normal exit
		}
		_ = c.processEntry(ctx, entry, opts, int(maxRetries), handler)
	}
	return nil
}

// RecoverPending claims messages idle longer than idleThreshold and processes them.
func (c *Consumer) RecoverPending(
	ctx context.Context,
	opts ports.ConsumeOptions,
	idleThreshold time.Duration,
	handler ports.EventHandler,
) error {
	maxRetries := normalizeMaxRetries(opts.MaxRetries)

	if err := ensureStream(ctx, c.client, opts.Stream, opts.Group); err != nil {
		return err
	}

	idleMs := strconv.FormatInt(idleThreshold.Milliseconds(), 10)
	cmd := c.client.B().Xautoclaim().
		Key(opts.Stream).
		Group(opts.Group).
		Consumer(opts.ConsumerName).
		MinIdleTime(idleMs).
		Start("0-0").
		Count(100).
		Build()

	result, err := c.client.Do(ctx, cmd).ToArray()
	if err != nil {
		return fmt.Errorf("eventbus recover pending %s: %w", opts.Stream, err)
	}
	if len(result) < 2 {
		return nil
	}

	entries, err := result[1].AsXRange()
	if err != nil {
		return fmt.Errorf("eventbus recover pending parse %s: %w", opts.Stream, err)
	}

	for _, entry := range entries {
		_ = c.processEntry(ctx, entry, opts, int(maxRetries), handler)
	}
	return nil
}

// PendingCount returns the number of messages in the PEL for the given stream and group.
// This method is not part of the ports.EventConsumer interface; it is exposed for testing.
func (c *Consumer) PendingCount(ctx context.Context, stream, group string) (int64, error) {
	cmd := c.client.B().Xpending().Key(stream).Group(group).Build()
	arr, err := c.client.Do(ctx, cmd).ToArray()
	if err != nil {
		return 0, fmt.Errorf("eventbus pending count %s: %w", stream, err)
	}
	if len(arr) == 0 {
		return 0, nil
	}
	n, err := arr[0].ToInt64()
	if err != nil {
		return 0, fmt.Errorf("eventbus pending count parse %s: %w", stream, err)
	}
	return n, nil
}

// Close releases the underlying Valkey connections.
func (c *Consumer) Close() error {
	_ = c.publisher.Close()
	c.client.Close()
	return nil
}

func (c *Consumer) readGroup(
	ctx context.Context,
	stream, group, consumer string,
	blockMs, count int64,
) ([]valkey.XRangeEntry, error) {
	cmd := c.client.B().Xreadgroup().
		Group(group, consumer).
		Count(count).
		Block(blockMs).
		Streams().
		Key(stream).
		Id(">").
		Build()
	result, err := c.client.Do(ctx, cmd).AsXRead()
	if err != nil {
		return nil, fmt.Errorf("eventbus readgroup %s: %w", stream, err)
	}
	return result[stream], nil
}

func (c *Consumer) processEntry(
	ctx context.Context,
	entry valkey.XRangeEntry,
	opts ports.ConsumeOptions,
	maxRetries int,
	handler ports.EventHandler,
) error {
	evt, err := unmarshalEnvelope(entry.FieldValues)
	if err != nil {
		_ = c.ack(ctx, opts.Stream, opts.Group, entry.ID)
		return err
	}

	handlerErr := handler(ctx, evt)
	if handlerErr == nil {
		return c.ack(ctx, opts.Stream, opts.Group, entry.ID)
	}

	deliveries, countErr := c.deliveryCount(ctx, opts.Stream, opts.Group, entry.ID)
	if countErr != nil {
		return handlerErr
	}

	if deliveries >= int64(maxRetries) {
		return c.moveToDLQ(ctx, opts.Stream, opts.Group, entry.ID, evt, deliveries)
	}
	return handlerErr
}

func (c *Consumer) ack(ctx context.Context, stream, group, id string) error {
	cmd := c.client.B().Xack().Key(stream).Group(group).Id(id).Build()
	if err := c.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("eventbus ack %s %s: %w", stream, id, err)
	}
	return nil
}

func (c *Consumer) deliveryCount(ctx context.Context, stream, group, msgID string) (int64, error) {
	cmd := c.client.B().Xpending().Key(stream).Group(group).
		Start(msgID).End(msgID).Count(1).
		Build()
	arr, err := c.client.Do(ctx, cmd).ToArray()
	if err != nil {
		return 0, fmt.Errorf("eventbus delivery count %s: %w", stream, err)
	}
	if len(arr) == 0 {
		return 0, nil
	}
	entry, err := arr[0].ToArray()
	if err != nil {
		return 0, fmt.Errorf("eventbus delivery count parse: %w", err)
	}
	if len(entry) < 4 {
		return 0, fmt.Errorf("eventbus delivery count parse: unexpected response length %d", len(entry))
	}
	n, err := entry[3].ToInt64()
	if err != nil {
		return 0, fmt.Errorf("eventbus delivery count parse int64: %w", err)
	}
	return n, nil
}

func (c *Consumer) moveToDLQ(ctx context.Context, stream, group, msgID string, evt domain.Event, deliveries int64) error {
	dlqData, err := json.Marshal(map[string]any{
		"original_id":     evt.ID,
		"original_type":   evt.Type,
		"original_source": evt.Source,
		"original_stream": stream,
		"retry_count":     deliveries,
	})
	if err != nil {
		return fmt.Errorf("eventbus dlq marshal: %w", err)
	}

	dlqEvt := domain.Event{
		ID:              evt.ID,
		Type:            domain.EventTypeDLQ,
		Source:          stream,
		SpecVersion:     "1.0",
		Time:            evt.Time,
		DataContentType: "application/json",
		Auth:            evt.Auth,
		Data:            dlqData,
	}

	if pubErr := c.publisher.Publish(ctx, dlqEvt, ports.PublishOptions{Stream: dlqStream}); pubErr != nil {
		return fmt.Errorf("eventbus dlq publish: %w", pubErr)
	}

	return c.ack(ctx, stream, group, msgID)
}

func normalizeMaxRetries(v int) int64 {
	if v <= 0 {
		return 3
	}
	return int64(v)
}

func normalizeBlockMs(d time.Duration) int64 {
	if d <= 0 {
		return 500
	}
	return d.Milliseconds()
}
