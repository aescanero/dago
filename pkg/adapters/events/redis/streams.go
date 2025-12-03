package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aescanero/dago-libs/pkg/ports"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// StreamsEventBus implements EventBus using Redis Streams
type StreamsEventBus struct {
	client        *redis.Client
	logger        *zap.Logger
	consumerGroup string
	consumerName  string
}

// NewStreamsEventBus creates a new Redis Streams event bus
func NewStreamsEventBus(client *redis.Client, consumerGroup, consumerName string, logger *zap.Logger) (*StreamsEventBus, error) {
	return &StreamsEventBus{
		client:        client,
		logger:        logger,
		consumerGroup: consumerGroup,
		consumerName:  consumerName,
	}, nil
}

// Publish publishes an event to the appropriate stream topic
func (e *StreamsEventBus) Publish(ctx context.Context, topic string, event ports.Event) error {
	streamKey := getStreamKey(topic)

	// Serialize event
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Add to stream
	args := &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}

	if _, err := e.client.XAdd(ctx, args).Result(); err != nil {
		return fmt.Errorf("failed to add to stream: %w", err)
	}

	e.logger.Debug("event published",
		zap.String("event_id", event.ID),
		zap.String("type", string(event.Type)),
		zap.String("topic", topic),
		zap.String("stream", streamKey))

	return nil
}

// Subscribe subscribes to events on a specific topic
func (e *StreamsEventBus) Subscribe(ctx context.Context, topic string, handler ports.EventHandler) error {
	streamKey := getStreamKey(topic)

	// Create consumer group if it doesn't exist
	err := e.client.XGroupCreateMkStream(ctx, streamKey, e.consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	e.logger.Info("subscribed to event stream",
		zap.String("stream", streamKey),
		zap.String("topic", topic),
		zap.String("consumer_group", e.consumerGroup),
		zap.String("consumer", e.consumerName))

	// Start reading from stream
	go e.readStream(ctx, streamKey, handler)

	return nil
}

// readStream reads events from a stream
func (e *StreamsEventBus) readStream(ctx context.Context, streamKey string, handler ports.EventHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read from stream
			streams, err := e.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    e.consumerGroup,
				Consumer: e.consumerName,
				Streams:  []string{streamKey, ">"},
				Count:    10,
				Block:    time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					// No new messages
					continue
				}
				e.logger.Error("failed to read from stream",
					zap.String("stream", streamKey),
					zap.Error(err))
				time.Sleep(time.Second)
				continue
			}

			// Process messages
			for _, stream := range streams {
				for _, message := range stream.Messages {
					e.processMessage(ctx, streamKey, message, handler)
				}
			}
		}
	}
}

// processMessage processes a single message from the stream
func (e *StreamsEventBus) processMessage(ctx context.Context, streamKey string, message redis.XMessage, handler ports.EventHandler) {
	// Extract event data
	data, ok := message.Values["data"].(string)
	if !ok {
		e.logger.Error("invalid message format",
			zap.String("stream", streamKey),
			zap.String("message_id", message.ID))
		return
	}

	// Deserialize event
	var event ports.Event
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		e.logger.Error("failed to unmarshal event",
			zap.String("stream", streamKey),
			zap.String("message_id", message.ID),
			zap.Error(err))
		return
	}

	// Call handler
	if err := handler(ctx, event); err != nil {
		e.logger.Error("handler error",
			zap.String("stream", streamKey),
			zap.String("message_id", message.ID),
			zap.Error(err))
		return
	}

	// Acknowledge message
	if err := e.client.XAck(ctx, streamKey, e.consumerGroup, message.ID).Err(); err != nil {
		e.logger.Error("failed to acknowledge message",
			zap.String("stream", streamKey),
			zap.String("message_id", message.ID),
			zap.Error(err))
	}
}

// Unsubscribe removes subscriptions from a topic
func (e *StreamsEventBus) Unsubscribe(ctx context.Context, topic string) error {
	// For Redis streams, we don't actively remove consumers
	// They will timeout naturally or can be cleaned up separately
	// For MVP, just return nil
	return nil
}

// Close closes the event bus and cleans up resources
func (e *StreamsEventBus) Close() error {
	// For MVP, no additional cleanup needed
	// Redis client should be closed by the caller
	return nil
}

// getStreamKey returns the Redis stream key for a topic
func getStreamKey(topic string) string {
	return fmt.Sprintf("dago:events:%s", topic)
}
