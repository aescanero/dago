package usecase_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/orchestrator/internal/usecase"
	"github.com/aescanero/dago/tests/testutil/fakes"
)

func newExecUC() (*usecase.ExecutionUseCase, *fakes.InMemoryGraphRepository, *fakes.InMemoryExecutionRepository, *fakes.InMemoryPublisher) {
	gr := fakes.NewInMemoryGraphRepository()
	er := fakes.NewInMemoryExecutionRepository()
	pub := fakes.NewInMemoryPublisher()
	return usecase.NewExecutionUseCase(gr, er, pub), gr, er, pub
}

func validGraphDef(entryNode string) json.RawMessage {
	def, _ := json.Marshal(domain.GraphDefinition{
		EntryNode: entryNode,
		Nodes:     map[string]domain.NodeDefinition{entryNode: {Pattern: "llm_call"}},
		Edges:     []domain.EdgeDefinition{},
	})
	return def
}

func seedGraph(t *testing.T, repo *fakes.InMemoryGraphRepository) *domain.Graph {
	t.Helper()
	g := &domain.Graph{
		ID:         uuid.New(),
		Name:       "test-graph",
		Version:    "1.0.0",
		EntryNode:  "start",
		Definition: validGraphDef("start"),
		Status:     domain.GraphStatusDraft,
	}
	created, err := repo.Create(context.Background(), g)
	require.NoError(t, err)
	return created
}

func TestStartExecutionSuccess(t *testing.T) {
	uc, gr, _, pub := newExecUC()
	g := seedGraph(t, gr)

	e, err := uc.StartExecution(context.Background(), usecase.StartExecutionInput{GraphID: g.ID})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, e.ID)
	assert.Equal(t, g.ID, e.GraphID)
	assert.Equal(t, domain.ExecutionStatusRunning, e.Status)
	assert.Equal(t, "start", e.CurrentNode)
	assert.Equal(t, json.RawMessage("{}"), e.Variables)
	assert.Equal(t, json.RawMessage("[]"), e.Messages)
	assert.Equal(t, json.RawMessage("{}"), e.NodeResults)

	// Verify node.execute.requested was published
	evts := pub.StreamEvents(domain.StreamNodeExecuteRequested)
	require.Len(t, evts, 1)
	assert.Equal(t, domain.EventTypeNodeExecuteRequested, evts[0].Type)
}

func TestStartExecutionGraphNotFound(t *testing.T) {
	uc, _, _, _ := newExecUC()
	_, err := uc.StartExecution(context.Background(), usecase.StartExecutionInput{GraphID: uuid.New()})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestStartExecutionGraphValidationError(t *testing.T) {
	uc, gr, _, _ := newExecUC()
	invalidDef, _ := json.Marshal(domain.GraphDefinition{
		EntryNode: "start",
		Nodes:     map[string]domain.NodeDefinition{"start": {Pattern: "llm_call"}, "end": {Pattern: "llm_call"}},
		Edges:     []domain.EdgeDefinition{{Type: "conditional", From: "start", To: "end"}},
	})
	g := &domain.Graph{
		ID:         uuid.New(),
		Name:       "bad-graph",
		Version:    "1.0.0",
		EntryNode:  "start",
		Definition: invalidDef,
		Status:     domain.GraphStatusDraft,
	}
	_, err := gr.Create(context.Background(), g)
	require.NoError(t, err)

	_, err = uc.StartExecution(context.Background(), usecase.StartExecutionInput{GraphID: g.ID})
	require.ErrorIs(t, err, domain.ErrGraphValidation)
}

func TestStartExecutionWithInitialVariables(t *testing.T) {
	uc, gr, _, _ := newExecUC()
	g := seedGraph(t, gr)
	vars := json.RawMessage(`{"input":"hello"}`)

	e, err := uc.StartExecution(context.Background(), usecase.StartExecutionInput{
		GraphID:   g.ID,
		Variables: vars,
	})
	require.NoError(t, err)
	assert.Equal(t, vars, e.Variables)
}
