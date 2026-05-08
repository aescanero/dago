//go:build integration

package valkey_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	valkeybus "github.com/aescanero/dago/adapters/eventbus/valkey"
	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

var valkeyAddr string

func TestMain(m *testing.M) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "valkey/valkey:8",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start valkey container: %v\n", err)
		os.Exit(1)
	}
	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get container host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get container port: %v\n", err)
		os.Exit(1)
	}
	valkeyAddr = fmt.Sprintf("%s:%s", host, port.Port())
	code := m.Run()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func newTestEvent(t *testing.T) domain.Event {
	t.Helper()
	data, err := json.Marshal(map[string]string{"execution_id": "test-exec-1"})
	require.NoError(t, err)
	return domain.Event{
		ID:              "550e8400-e29b-41d4-a716-446655440001",
		Type:            domain.EventTypeExecutionRequested,
		Source:          "orchestrator",
		SpecVersion:     "1.0",
		Time:            time.Now().UTC().Truncate(time.Millisecond),
		DataContentType: "application/json",
		Data:            data,
	}
}

// TestPublishAndConsume verifies that a published event is received by a consumer.
func TestPublishAndConsume(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pub, err := valkeybus.NewPublisher(valkeyAddr)
	require.NoError(t, err)
	defer pub.Close()

	cons, err := valkeybus.NewConsumer(valkeyAddr)
	require.NoError(t, err)
	defer cons.Close()

	stream := "test.publish.consume"
	group := "test-group"
	evt := newTestEvent(t)

	require.NoError(t, pub.Publish(ctx, evt, ports.PublishOptions{Stream: stream}))

	received := make(chan domain.Event, 1)
	opts := ports.ConsumeOptions{
		Stream:        stream,
		Group:         group,
		ConsumerName:  "consumer-1",
		BlockDuration: 500 * time.Millisecond,
		MaxRetries:    3,
	}
	go func() {
		_ = cons.Subscribe(ctx, opts, func(_ context.Context, e domain.Event) error {
			received <- e
			return nil
		})
	}()

	select {
	case got := <-received:
		assert.Equal(t, evt.ID, got.ID)
		assert.Equal(t, evt.Type, got.Type)
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}

// TestConsumerGroupAck verifies that a message is ACKed and removed from PEL on nil handler return.
func TestConsumerGroupAck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pub, err := valkeybus.NewPublisher(valkeyAddr)
	require.NoError(t, err)
	defer pub.Close()

	cons, err := valkeybus.NewConsumer(valkeyAddr)
	require.NoError(t, err)
	defer cons.Close()

	stream := "test.ack"
	group := "ack-group"
	evt := newTestEvent(t)

	require.NoError(t, pub.Publish(ctx, evt, ports.PublishOptions{Stream: stream}))

	done := make(chan struct{})
	opts := ports.ConsumeOptions{
		Stream:        stream,
		Group:         group,
		ConsumerName:  "consumer-1",
		BlockDuration: 500 * time.Millisecond,
		MaxRetries:    3,
	}
	go func() {
		_ = cons.Subscribe(ctx, opts, func(_ context.Context, _ domain.Event) error {
			close(done)
			return nil
		})
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout waiting for handler")
	}

	// After ACK the PEL should be empty.
	pending, err := cons.PendingCount(ctx, stream, group)
	require.NoError(t, err)
	assert.Equal(t, int64(0), pending, "PEL must be empty after ACK")
}

// TestConsumerGroupNoAck verifies that a message stays in PEL when the handler returns an error.
func TestConsumerGroupNoAck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pub, err := valkeybus.NewPublisher(valkeyAddr)
	require.NoError(t, err)
	defer pub.Close()

	cons, err := valkeybus.NewConsumer(valkeyAddr)
	require.NoError(t, err)
	defer cons.Close()

	stream := "test.noack"
	group := "noack-group"
	evt := newTestEvent(t)

	require.NoError(t, pub.Publish(ctx, evt, ports.PublishOptions{Stream: stream}))

	called := make(chan struct{}, 1)
	subCtx, subCancel := context.WithCancel(ctx)
	opts := ports.ConsumeOptions{
		Stream:        stream,
		Group:         group,
		ConsumerName:  "consumer-1",
		BlockDuration: 500 * time.Millisecond,
		MaxRetries:    10, // high to avoid DLQ in this test
	}
	go func() {
		_ = cons.Subscribe(subCtx, opts, func(_ context.Context, _ domain.Event) error {
			select {
			case called <- struct{}{}:
			default:
			}
			return fmt.Errorf("handler error")
		})
	}()

	select {
	case <-called:
		subCancel()
	case <-ctx.Done():
		subCancel()
		t.Fatal("timeout waiting for handler")
	}
	// Small pause to ensure Subscribe exited.
	time.Sleep(200 * time.Millisecond)

	pending, err := cons.PendingCount(ctx, stream, group)
	require.NoError(t, err)
	assert.Greater(t, pending, int64(0), "PEL must still contain the unACKed message")
}

// TestPendingRecovery verifies that RecoverPending reassigns idle PEL messages.
func TestPendingRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pub, err := valkeybus.NewPublisher(valkeyAddr)
	require.NoError(t, err)
	defer pub.Close()

	cons1, err := valkeybus.NewConsumer(valkeyAddr)
	require.NoError(t, err)
	defer cons1.Close()

	cons2, err := valkeybus.NewConsumer(valkeyAddr)
	require.NoError(t, err)
	defer cons2.Close()

	stream := "test.recovery"
	group := "recovery-group"
	evt := newTestEvent(t)

	require.NoError(t, pub.Publish(ctx, evt, ports.PublishOptions{Stream: stream}))

	// Consumer 1 reads the message but does NOT ACK it (returns error, MaxRetries high).
	delivered := make(chan struct{})
	subCtx, subCancel := context.WithCancel(ctx)
	opts1 := ports.ConsumeOptions{
		Stream:        stream,
		Group:         group,
		ConsumerName:  "consumer-1",
		BlockDuration: 500 * time.Millisecond,
		MaxRetries:    10,
	}
	go func() {
		_ = cons1.Subscribe(subCtx, opts1, func(_ context.Context, _ domain.Event) error {
			select {
			case delivered <- struct{}{}:
			default:
			}
			return fmt.Errorf("transient error")
		})
	}()
	select {
	case <-delivered:
		subCancel()
	case <-ctx.Done():
		subCancel()
		t.Fatal("timeout waiting for consumer-1 delivery")
	}
	time.Sleep(200 * time.Millisecond)

	// Consumer 2 recovers idle messages (idle >= 0 to pick it up immediately).
	recovered := make(chan domain.Event, 1)
	opts2 := ports.ConsumeOptions{
		Stream:       stream,
		Group:        group,
		ConsumerName: "consumer-2",
		MaxRetries:   3,
	}
	err = cons2.RecoverPending(ctx, opts2, 0, func(_ context.Context, e domain.Event) error {
		recovered <- e
		return nil
	})
	require.NoError(t, err)

	select {
	case got := <-recovered:
		assert.Equal(t, evt.ID, got.ID)
	case <-time.After(3 * time.Second):
		t.Fatal("RecoverPending did not deliver the message")
	}
}

// TestDLQAfterMaxRetries verifies that a message moves to dago.dlq after MaxRetries failures.
func TestDLQAfterMaxRetries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pub, err := valkeybus.NewPublisher(valkeyAddr)
	require.NoError(t, err)
	defer pub.Close()

	cons, err := valkeybus.NewConsumer(valkeyAddr)
	require.NoError(t, err)
	defer cons.Close()

	dlqCons, err := valkeybus.NewConsumer(valkeyAddr)
	require.NoError(t, err)
	defer dlqCons.Close()

	stream := "test.dlq"
	group := "dlq-group"
	evt := newTestEvent(t)

	require.NoError(t, pub.Publish(ctx, evt, ports.PublishOptions{Stream: stream}))

	dlqReceived := make(chan domain.Event, 1)
	go func() {
		_ = dlqCons.Subscribe(ctx, ports.ConsumeOptions{
			Stream:        domain.EventTypeDLQ,
			Group:         "dlq-reader",
			ConsumerName:  "dlq-consumer",
			BlockDuration: 500 * time.Millisecond,
			MaxRetries:    1,
		}, func(_ context.Context, e domain.Event) error {
			dlqReceived <- e
			return nil
		})
	}()

	opts := ports.ConsumeOptions{
		Stream:        stream,
		Group:         group,
		ConsumerName:  "consumer-1",
		BlockDuration: 100 * time.Millisecond,
		MaxRetries:    3,
	}
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()
	go func() {
		_ = cons.Subscribe(subCtx, opts, func(_ context.Context, _ domain.Event) error {
			return fmt.Errorf("persistent error")
		})
	}()

	select {
	case dlqEvt := <-dlqReceived:
		assert.Equal(t, domain.EventTypeDLQ, dlqEvt.Type)
		// The DLQ payload must contain the original execution_id.
		var dlqData map[string]any
		require.NoError(t, json.Unmarshal(dlqEvt.Data, &dlqData))
		assert.NotEmpty(t, dlqData["original_id"])
	case <-ctx.Done():
		t.Fatal("timeout waiting for DLQ message")
	}
}

// TestEnvelopeRoundtrip verifies that an event with all fields survives Publish→Subscribe intact.
func TestEnvelopeRoundtrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pub, err := valkeybus.NewPublisher(valkeyAddr)
	require.NoError(t, err)
	defer pub.Close()

	cons, err := valkeybus.NewConsumer(valkeyAddr)
	require.NoError(t, err)
	defer cons.Close()

	stream := "test.roundtrip"
	group := "rt-group"
	data, err := json.Marshal(map[string]string{"key": "value"})
	require.NoError(t, err)

	sent := domain.Event{
		ID:              "550e8400-e29b-41d4-a716-446655440002",
		Type:            domain.EventTypeNodeCompleted,
		Source:          "executor",
		SpecVersion:     "1.0",
		Time:            time.Now().UTC().Truncate(time.Millisecond),
		DataContentType: "application/json",
		Auth: &domain.EventAuth{
			Sub:     "user-uuid-1234",
			Scope:   "graphs:execute",
			Tags:    []string{"env:prod", "team:platform"},
			OrgUnit: "org-unit-uuid-5678",
		},
		Data: data,
	}

	require.NoError(t, pub.Publish(ctx, sent, ports.PublishOptions{Stream: stream}))

	received := make(chan domain.Event, 1)
	opts := ports.ConsumeOptions{
		Stream:        stream,
		Group:         group,
		ConsumerName:  "consumer-1",
		BlockDuration: 500 * time.Millisecond,
		MaxRetries:    3,
	}
	go func() {
		_ = cons.Subscribe(ctx, opts, func(_ context.Context, e domain.Event) error {
			received <- e
			return nil
		})
	}()

	select {
	case got := <-received:
		assert.Equal(t, sent.ID, got.ID)
		assert.Equal(t, sent.Type, got.Type)
		assert.Equal(t, sent.Source, got.Source)
		assert.Equal(t, sent.SpecVersion, got.SpecVersion)
		assert.True(t, sent.Time.Equal(got.Time), "time mismatch: %v vs %v", sent.Time, got.Time)
		assert.Equal(t, sent.DataContentType, got.DataContentType)
		require.NotNil(t, got.Auth)
		assert.Equal(t, sent.Auth.Sub, got.Auth.Sub)
		assert.Equal(t, sent.Auth.Scope, got.Auth.Scope)
		assert.Equal(t, sent.Auth.Tags, got.Auth.Tags)
		assert.Equal(t, sent.Auth.OrgUnit, got.Auth.OrgUnit)
		assert.JSONEq(t, string(sent.Data), string(got.Data))
	case <-ctx.Done():
		t.Fatal("timeout waiting for roundtrip event")
	}
}
