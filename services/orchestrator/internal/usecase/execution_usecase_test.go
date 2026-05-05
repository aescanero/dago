package usecase_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/orchestrator/internal/usecase"
	"github.com/aescanero/dago/tests/testutil/fakes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newExecUC() (*usecase.ExecutionUseCase, *fakes.InMemoryGraphRepository, *fakes.InMemoryExecutionRepository) {
	gr := fakes.NewInMemoryGraphRepository()
	er := fakes.NewInMemoryExecutionRepository()
	return usecase.NewExecutionUseCase(gr, er), gr, er
}

func seedGraph(t *testing.T, repo *fakes.InMemoryGraphRepository) *domain.Graph {
	t.Helper()
	g := &domain.Graph{
		ID:         uuid.New(),
		Name:       "test-graph",
		Version:    "1.0.0",
		EntryNode:  "start",
		Definition: json.RawMessage(`{}`),
		Status:     domain.GraphStatusDraft,
	}
	created, err := repo.Create(context.Background(), g)
	require.NoError(t, err)
	return created
}

func TestStartExecutionSuccess(t *testing.T) {
	uc, gr, _ := newExecUC()
	g := seedGraph(t, gr)

	e, err := uc.StartExecution(context.Background(), usecase.StartExecutionInput{GraphID: g.ID})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, e.ID)
	assert.Equal(t, g.ID, e.GraphID)
	assert.Equal(t, domain.ExecutionStatusPending, e.Status)
	assert.Equal(t, json.RawMessage("{}"), e.Variables)
	assert.Equal(t, json.RawMessage("[]"), e.Messages)
	assert.Equal(t, json.RawMessage("{}"), e.NodeResults)
}

func TestStartExecutionGraphNotFound(t *testing.T) {
	uc, _, _ := newExecUC()
	_, err := uc.StartExecution(context.Background(), usecase.StartExecutionInput{GraphID: uuid.New()})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestStartExecutionWithInitialVariables(t *testing.T) {
	uc, gr, _ := newExecUC()
	g := seedGraph(t, gr)
	vars := json.RawMessage(`{"input":"hello"}`)

	e, err := uc.StartExecution(context.Background(), usecase.StartExecutionInput{
		GraphID:   g.ID,
		Variables: vars,
	})
	require.NoError(t, err)
	assert.Equal(t, vars, e.Variables)
}
