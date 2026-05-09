//go:build integration

package consumer_test

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
	"github.com/aescanero/dago/services/orchestrator/internal/consumer"
	"github.com/aescanero/dago/services/orchestrator/internal/statemachine"
	"github.com/aescanero/dago/tests/testutil/fakes"
)

var consumerIntegrationValkeyAddr string

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
	consumerIntegrationValkeyAddr = fmt.Sprintf("%s:%s", host, port.Port())
	code := m.Run()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

// TestNodeResultConsumer_Integration_NodeExecuted verifies that publishing a node.executed
// event is consumed, calls HandleNodeExecuted, and the state is updated (ACK on success).
func TestNodeResultConsumer_Integration_NodeExecuted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pub, err := valkeybus.NewPublisher(consumerIntegrationValkeyAddr)
	require.NoError(t, err)
	defer pub.Close()

	resultPub := fakes.NewInMemoryPublisher()
	execRepo := fakes.NewInMemoryExecutionRepository()
	graphRepo := fakes.NewInMemoryGraphRepository()

	// Create a graph with a single terminal node
	graphDef, _ := json.Marshal(domain.GraphDefinition{
		EntryNode: "node_a",
		Nodes:     map[string]domain.NodeDefinition{"node_a": {Pattern: "llm_call"}},
		Edges:     []domain.EdgeDefinition{},
	})
	g := &domain.Graph{
		ID:         uuid.New(),
		Name:       "test-graph",
		Version:    "1.0.0",
		EntryNode:  "node_a",
		Definition: graphDef,
		Status:     domain.GraphStatusActive,
	}
	createdGraph, err := graphRepo.Create(ctx, g)
	require.NoError(t, err)

	exec := &domain.Execution{
		ID:      uuid.New(),
		GraphID: createdGraph.ID,
		Status:  domain.ExecutionStatusRunning,
	}
	_, err = execRepo.Create(ctx, exec)
	require.NoError(t, err)

	sm := statemachine.NewExecutionStateMachine(execRepo, resultPub)
	c := consumer.NewNodeResultConsumer(execRepo, graphRepo, sm)

	// Publish a node.executed event to the real Valkey stream
	nodeExecutedData, _ := json.Marshal(map[string]any{
		"execution_id": exec.ID.String(),
		"graph_id":     createdGraph.ID.String(),
		"node_id":      "node-instance-1",
		"node_key":     "node_a",
		"duration_ms":  100,
		"output":       map[string]string{"result": "hello"},
	})
	evt := domain.Event{
		ID:              uuid.New().String(),
		Type:            domain.EventTypeNodeExecuted,
		Source:          "executor",
		SpecVersion:     "1.0",
		Time:            time.Now().UTC(),
		DataContentType: "application/json",
		Data:            nodeExecutedData,
	}
	require.NoError(t, pub.Publish(ctx, evt, ports.PublishOptions{Stream: domain.StreamNodeExecuted}))

	// Set up consumer and subscribe
	valkeyConsumer, err := valkeybus.NewConsumer(consumerIntegrationValkeyAddr)
	require.NoError(t, err)
	defer valkeyConsumer.Close()

	done := make(chan struct{})
	go func() {
		_ = valkeyConsumer.Subscribe(ctx, ports.ConsumeOptions{
			Stream:        domain.StreamNodeExecuted,
			Group:         "orchestrator-group",
			ConsumerName:  "consumer-integration-test-1",
			BlockDuration: 500 * time.Millisecond,
			MaxRetries:    3,
		}, func(ctx context.Context, e domain.Event) error {
			err := c.HandleNodeExecuted(ctx, e)
			if err == nil {
				select {
				case done <- struct{}{}:
				default:
				}
			}
			return err
		})
	}()

	select {
	case <-done:
		// Verify execution is now completed (terminal node)
		updated, err := execRepo.FindByID(ctx, exec.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ExecutionStatusCompleted, updated.Status)
	case <-ctx.Done():
		t.Fatal("timeout: node.executed event was not processed by consumer")
	}
}
