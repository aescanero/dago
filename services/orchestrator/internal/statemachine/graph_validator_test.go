package statemachine_test

import (
	"errors"
	"testing"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/orchestrator/internal/statemachine"
)

func TestValidateGraph_ValidSequential(t *testing.T) {
	g := domain.GraphDefinition{
		EntryNode: "a",
		Nodes: map[string]domain.NodeDefinition{
			"a": {Pattern: "llm_call"},
			"b": {Pattern: "llm_call"},
			"c": {Pattern: "llm_call"},
		},
		Edges: []domain.EdgeDefinition{
			{Type: "sequential", From: "a", To: "b"},
			{Type: "sequential", From: "b", To: "c"},
		},
	}
	if err := statemachine.ValidateGraph(g); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateGraph_EntryNodeMissing(t *testing.T) {
	g := domain.GraphDefinition{
		EntryNode: "missing",
		Nodes: map[string]domain.NodeDefinition{
			"a": {Pattern: "llm_call"},
		},
		Edges: []domain.EdgeDefinition{},
	}
	err := statemachine.ValidateGraph(g)
	if !errors.Is(err, domain.ErrGraphValidation) {
		t.Errorf("expected ErrGraphValidation, got %v", err)
	}
}

func TestValidateGraph_UnsupportedEdgeType(t *testing.T) {
	g := domain.GraphDefinition{
		EntryNode: "a",
		Nodes: map[string]domain.NodeDefinition{
			"a": {Pattern: "llm_call"},
			"b": {Pattern: "llm_call"},
		},
		Edges: []domain.EdgeDefinition{
			{Type: "conditional", From: "a", To: "b"},
		},
	}
	err := statemachine.ValidateGraph(g)
	if !errors.Is(err, domain.ErrGraphValidation) {
		t.Errorf("expected ErrGraphValidation, got %v", err)
	}
	if err != nil && !containsString(err.Error(), "unsupported edge type: conditional") {
		t.Errorf("expected error message to contain 'unsupported edge type: conditional', got %q", err.Error())
	}
}

func TestValidateGraph_UnreachableNode(t *testing.T) {
	g := domain.GraphDefinition{
		EntryNode: "a",
		Nodes: map[string]domain.NodeDefinition{
			"a":        {Pattern: "llm_call"},
			"b":        {Pattern: "llm_call"},
			"orphaned": {Pattern: "llm_call"},
		},
		Edges: []domain.EdgeDefinition{
			{Type: "sequential", From: "a", To: "b"},
		},
	}
	err := statemachine.ValidateGraph(g)
	if !errors.Is(err, domain.ErrGraphValidation) {
		t.Errorf("expected ErrGraphValidation for unreachable node, got %v", err)
	}
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
