package statemachine_test

import (
	"testing"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/orchestrator/internal/statemachine"
)

func TestNextNode_SequentialSuccessor(t *testing.T) {
	g := domain.GraphDefinition{
		EntryNode: "a",
		Nodes: map[string]domain.NodeDefinition{
			"a": {Pattern: "llm_call"},
			"b": {Pattern: "llm_call"},
		},
		Edges: []domain.EdgeDefinition{
			{Type: "sequential", From: "a", To: "b"},
		},
	}
	next, err := statemachine.NextNode(g, "a")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if next != "b" {
		t.Errorf("expected next node 'b', got %q", next)
	}
}

func TestNextNode_Terminal(t *testing.T) {
	g := domain.GraphDefinition{
		EntryNode: "a",
		Nodes: map[string]domain.NodeDefinition{
			"a": {Pattern: "llm_call"},
			"b": {Pattern: "llm_call"},
		},
		Edges: []domain.EdgeDefinition{
			{Type: "sequential", From: "a", To: "b"},
		},
	}
	next, err := statemachine.NextNode(g, "b")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if next != "" {
		t.Errorf("expected empty next node (terminal), got %q", next)
	}
}
