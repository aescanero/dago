//go:build integration

package statemachine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	valkeybus "github.com/aescanero/dago/adapters/eventbus/valkey"
	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/aescanero/dago/services/orchestrator/internal/statemachine"
	"github.com/aescanero/dago/tests/testutil/fakes"
)

var integrationValkeyAddr string

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
	integrationValkeyAddr = fmt.Sprintf("%s:%s", host, port.Port())
	code := m.Run()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

// TestSM_Integration_SubmitValid_PublishesNodeExecuteRequested verifies that
// HandleNodeExecuted on an intermediate node publishes node.execute.requested to Valkey.
func TestSM_Integration_SubmitValid_PublishesNodeExecuteRequested(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pub, err := valkeybus.NewPublisher(integrationValkeyAddr)
	require.NoError(t, err)
	defer pub.Close()

	cons, err := valkeybus.NewConsumer(integrationValkeyAddr)
	require.NoError(t, err)
	defer cons.Close()

	repo := fakes.NewInMemoryExecutionRepository()
	sm := statemachine.NewExecutionStateMachine(repo, pub)

	graphID := uuid.New()
	exec := &domain.Execution{
		ID:      uuid.New(),
		GraphID: graphID,
		Status:  domain.ExecutionStatusRunning,
	}
	_, err = repo.Create(ctx, exec)
	require.NoError(t, err)

	g := domain.GraphDefinition{
		EntryNode: "node_a",
		Nodes: map[string]domain.NodeDefinition{
			"node_a": {Pattern: "llm_call"},
			"node_b": {Pattern: "llm_call"},
		},
		Edges: []domain.EdgeDefinition{
			{Type: "sequential", From: "node_a", To: "node_b"},
		},
	}

	// Trigger node_a completion (intermediate)
	err = sm.HandleNodeExecuted(ctx, exec, g, "node_a", json.RawMessage(`{}`), "test-token")
	require.NoError(t, err)

	// Verify execution status is still running
	updated, err := repo.FindByID(ctx, exec.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ExecutionStatusRunning, updated.Status)
	assert.Equal(t, "node_b", updated.CurrentNode)

	// Consume from the real Valkey stream to verify the event was published
	received := make(chan domain.Event, 1)
	go func() {
		_ = cons.Subscribe(ctx, ports.ConsumeOptions{
			Stream:        domain.StreamNodeExecuteRequested,
			Group:         "sm-integration-test-group",
			ConsumerName:  "sm-integration-test-1",
			BlockDuration: 500 * time.Millisecond,
			MaxRetries:    3,
		}, func(_ context.Context, e domain.Event) error {
			received <- e
			return nil
		})
	}()

	select {
	case evt := <-received:
		assert.Equal(t, domain.EventTypeNodeExecuteRequested, evt.Type)
		var data map[string]any
		require.NoError(t, json.Unmarshal(evt.Data, &data))
		assert.Equal(t, exec.ID.String(), data["execution_id"])
		assert.Equal(t, "node_b", data["node_key"])
	case <-ctx.Done():
		t.Fatal("timeout: node.execute.requested event not received from Valkey")
	}
}

// TestSM_Integration_TerminalNode_PublishesGraphCompleted verifies that
// HandleNodeExecuted on a terminal node publishes graph.completed.
func TestSM_Integration_TerminalNode_PublishesGraphCompleted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pub, err := valkeybus.NewPublisher(integrationValkeyAddr)
	require.NoError(t, err)
	defer pub.Close()

	cons, err := valkeybus.NewConsumer(integrationValkeyAddr)
	require.NoError(t, err)
	defer cons.Close()

	repo := fakes.NewInMemoryExecutionRepository()
	sm := statemachine.NewExecutionStateMachine(repo, pub)

	graphID := uuid.New()
	exec := &domain.Execution{
		ID:      uuid.New(),
		GraphID: graphID,
		Status:  domain.ExecutionStatusRunning,
	}
	_, err = repo.Create(ctx, exec)
	require.NoError(t, err)

	// Single-node graph — node_a is terminal
	g := domain.GraphDefinition{
		EntryNode: "node_a",
		Nodes: map[string]domain.NodeDefinition{
			"node_a": {Pattern: "llm_call"},
		},
		Edges: []domain.EdgeDefinition{},
	}

	err = sm.HandleNodeExecuted(ctx, exec, g, "node_a", json.RawMessage(`{"result":"ok"}`), "test-token")
	require.NoError(t, err)

	// Verify execution is completed in the repo
	updated, err := repo.FindByID(ctx, exec.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ExecutionStatusCompleted, updated.Status)

	// Consume graph.completed event from real Valkey
	received := make(chan domain.Event, 1)
	go func() {
		_ = cons.Subscribe(ctx, ports.ConsumeOptions{
			Stream:        domain.StreamGraphCompleted,
			Group:         "sm-integration-completed-group",
			ConsumerName:  "sm-integration-completed-1",
			BlockDuration: 500 * time.Millisecond,
			MaxRetries:    3,
		}, func(_ context.Context, e domain.Event) error {
			received <- e
			return nil
		})
	}()

	select {
	case evt := <-received:
		assert.Equal(t, domain.EventTypeGraphCompleted, evt.Type)
		var data map[string]any
		require.NoError(t, json.Unmarshal(evt.Data, &data))
		assert.Equal(t, exec.ID.String(), data["execution_id"])
	case <-ctx.Done():
		t.Fatal("timeout: graph.completed event not received from Valkey")
	}
}
