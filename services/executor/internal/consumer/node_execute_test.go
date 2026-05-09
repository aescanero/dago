//go:build integration

package consumer_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/aescanero/dago/adapters/eventbus/valkey"
	fakellm "github.com/aescanero/dago/adapters/llm/fake"
	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/aescanero/dago/services/executor/internal/consumer"
	"github.com/aescanero/dago/services/executor/internal/handler"
	"github.com/google/uuid"
)

func TestExecutorConsumer_LLMCallSuccess(t *testing.T) {
	addr := os.Getenv("EXECUTOR_VALKEY_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	pub, err := valkey.NewPublisher(addr)
	if err != nil {
		t.Fatalf("create publisher: %v", err)
	}
	defer pub.Close()

	eventConsumer, err := valkey.NewConsumer(addr)
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}
	defer eventConsumer.Close()

	resultConsumer, err := valkey.NewConsumer(addr)
	if err != nil {
		t.Fatalf("create result consumer: %v", err)
	}
	defer resultConsumer.Close()

	llm := &fakellm.FakeLLMClient{
		Responses: []ports.LLMResponse{{Content: "integration response", StopReason: "end_turn"}},
	}

	resultPub, err := valkey.NewPublisher(addr)
	if err != nil {
		t.Fatalf("create result publisher: %v", err)
	}
	defer resultPub.Close()

	llmHandler := handler.NewLLMCallHandler(llm, resultPub)
	dispatcher := handler.NewDispatcher(map[string]handler.NodeHandler{
		"llm_call": llmHandler,
	})
	c := consumer.NewNodeExecuteConsumer(eventConsumer, dispatcher)

	executionID := uuid.NewString()
	reqData := handler.NodeExecuteRequestedData{
		ExecutionID: executionID,
		GraphID:     uuid.NewString(),
		NodeID:      uuid.NewString(),
		NodeKey:     "llm_node",
		Pattern:     "llm_call",
		Config:      json.RawMessage(`{"model":"fake-model"}`),
		Variables:   map[string]any{},
		Messages:    []ports.Message{{Role: "user", Content: "test"}},
	}
	dataBytes, err := json.Marshal(reqData)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	evt := domain.Event{
		ID:              uuid.NewString(),
		Type:            domain.EventTypeNodeExecuteRequested,
		Source:          "test",
		SpecVersion:     "1.0",
		Time:            time.Now(),
		DataContentType: "application/json",
		Data:            json.RawMessage(dataBytes),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pub.Publish(ctx, evt, ports.PublishOptions{Stream: domain.StreamNodeExecuteRequested}); err != nil {
		t.Fatalf("publish request: %v", err)
	}

	runErr := make(chan error, 1)
	go func() {
		runErr <- c.Run(ctx)
	}()

	// Poll node.executed stream for our result.
	found := make(chan handler.NodeExecutedData, 1)
	go func() {
		_ = resultConsumer.Subscribe(ctx, ports.ConsumeOptions{
			Stream:       domain.StreamNodeExecuted,
			Group:        "test-result-group",
			ConsumerName: "test-result-1",
		}, func(_ context.Context, e domain.Event) error {
			var payload handler.NodeExecutedData
			if err := json.Unmarshal(e.Data, &payload); err != nil {
				return err
			}
			if payload.ExecutionID == executionID {
				found <- payload
			}
			return nil
		})
	}()

	select {
	case payload := <-found:
		var update map[string]any
		if err := json.Unmarshal(payload.VariablesUpdate, &update); err != nil {
			t.Fatalf("unmarshal variables_update: %v", err)
		}
		if update["response"] != "integration response" {
			t.Errorf("variables_update.response = %v, want 'integration response'", update["response"])
		}
	case <-ctx.Done():
		t.Fatal("timeout: node.executed event not received within 5 seconds")
	}
}
