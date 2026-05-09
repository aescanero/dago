package statemachine_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/orchestrator/internal/statemachine"
	"github.com/aescanero/dago/tests/testutil/fakes"
)

func makeExecution(status domain.ExecutionStatus) *domain.Execution {
	return &domain.Execution{
		ID:      uuid.New(),
		GraphID: uuid.New(),
		Status:  status,
	}
}

func makeLinearGraph() domain.GraphDefinition {
	return domain.GraphDefinition{
		EntryNode: "a",
		Nodes: map[string]domain.NodeDefinition{
			"a": {Pattern: "llm_call"},
			"b": {Pattern: "llm_call"},
		},
		Edges: []domain.EdgeDefinition{
			{Type: "sequential", From: "a", To: "b"},
		},
	}
}

func makeSingleNodeGraph() domain.GraphDefinition {
	return domain.GraphDefinition{
		EntryNode: "a",
		Nodes: map[string]domain.NodeDefinition{
			"a": {Pattern: "llm_call"},
		},
		Edges: []domain.EdgeDefinition{},
	}
}

func TestHandleNodeExecuted_IntermediateNode(t *testing.T) {
	repo := fakes.NewInMemoryExecutionRepository()
	pub := fakes.NewInMemoryPublisher()
	sm := statemachine.NewExecutionStateMachine(repo, pub)

	exec := makeExecution(domain.ExecutionStatusRunning)
	if _, err := repo.Create(context.Background(), exec); err != nil {
		t.Fatalf("create: %v", err)
	}

	g := makeLinearGraph()
	err := sm.HandleNodeExecuted(context.Background(), exec, g, "a", json.RawMessage(`{}`), "tok")
	if err != nil {
		t.Fatalf("HandleNodeExecuted: %v", err)
	}

	// Should publish node.execute.requested for the next node
	evts := pub.StreamEvents(domain.StreamNodeExecuteRequested)
	if len(evts) != 1 {
		t.Errorf("expected 1 node.execute.requested event, got %d", len(evts))
	}

	// Execution status should remain running and current_node updated
	updated, _ := repo.FindByID(context.Background(), exec.ID)
	if updated.Status != domain.ExecutionStatusRunning {
		t.Errorf("expected status running, got %s", updated.Status)
	}
	if updated.CurrentNode != "b" {
		t.Errorf("expected current_node 'b', got %q", updated.CurrentNode)
	}
}

func TestHandleNodeExecuted_TerminalNode(t *testing.T) {
	repo := fakes.NewInMemoryExecutionRepository()
	pub := fakes.NewInMemoryPublisher()
	sm := statemachine.NewExecutionStateMachine(repo, pub)

	exec := makeExecution(domain.ExecutionStatusRunning)
	if _, err := repo.Create(context.Background(), exec); err != nil {
		t.Fatalf("create: %v", err)
	}

	g := makeSingleNodeGraph()
	err := sm.HandleNodeExecuted(context.Background(), exec, g, "a", json.RawMessage(`{}`), "tok")
	if err != nil {
		t.Fatalf("HandleNodeExecuted: %v", err)
	}

	// Should publish graph.completed
	evts := pub.StreamEvents(domain.StreamGraphCompleted)
	if len(evts) != 1 {
		t.Errorf("expected 1 graph.completed event, got %d", len(evts))
	}

	// Execution should be completed
	updated, _ := repo.FindByID(context.Background(), exec.ID)
	if updated.Status != domain.ExecutionStatusCompleted {
		t.Errorf("expected status completed, got %s", updated.Status)
	}
}

func TestHandleNodeExecuteFailed_NonRetryable(t *testing.T) {
	repo := fakes.NewInMemoryExecutionRepository()
	pub := fakes.NewInMemoryPublisher()
	sm := statemachine.NewExecutionStateMachine(repo, pub)

	exec := makeExecution(domain.ExecutionStatusRunning)
	if _, err := repo.Create(context.Background(), exec); err != nil {
		t.Fatalf("create: %v", err)
	}

	err := sm.HandleNodeExecuteFailed(context.Background(), exec, false, "some error", "execution_error", "tok")
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	// Should publish graph.failed
	evts := pub.StreamEvents(domain.StreamGraphFailed)
	if len(evts) != 1 {
		t.Errorf("expected 1 graph.failed event, got %d", len(evts))
	}

	// Execution should be failed
	updated, _ := repo.FindByID(context.Background(), exec.ID)
	if updated.Status != domain.ExecutionStatusFailed {
		t.Errorf("expected status failed, got %s", updated.Status)
	}
}

func TestHandleNodeExecuteFailed_Retryable(t *testing.T) {
	repo := fakes.NewInMemoryExecutionRepository()
	pub := fakes.NewInMemoryPublisher()
	sm := statemachine.NewExecutionStateMachine(repo, pub)

	exec := makeExecution(domain.ExecutionStatusRunning)
	if _, err := repo.Create(context.Background(), exec); err != nil {
		t.Fatalf("create: %v", err)
	}

	err := sm.HandleNodeExecuteFailed(context.Background(), exec, true, "rate limited", "rate_limited", "tok")
	if !errors.Is(err, domain.ErrRetryable) {
		t.Errorf("expected ErrRetryable, got %v", err)
	}

	// Should NOT publish any event
	if len(pub.Events()) != 0 {
		t.Errorf("expected no events published for retryable failure, got %d", len(pub.Events()))
	}
}
