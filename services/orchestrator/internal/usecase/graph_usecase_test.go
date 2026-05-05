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

func newGraphUC() (*usecase.GraphUseCase, *fakes.InMemoryGraphRepository, *fakes.InMemoryExecutionRepository) {
	gr := fakes.NewInMemoryGraphRepository()
	er := fakes.NewInMemoryExecutionRepository()
	return usecase.NewGraphUseCase(gr, er), gr, er
}

func validCreateInput() usecase.CreateGraphInput {
	return usecase.CreateGraphInput{
		Name:       "my-graph",
		Version:    "1.0.0",
		EntryNode:  "start",
		Definition: json.RawMessage(`{"nodes":[],"edges":[]}`),
	}
}

func TestCreateGraphSuccess(t *testing.T) {
	uc, _, _ := newGraphUC()
	g, err := uc.CreateGraph(context.Background(), validCreateInput())
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, g.ID)
	assert.Equal(t, domain.GraphStatusDraft, g.Status)
	assert.Equal(t, "my-graph", g.Name)
	assert.Equal(t, "1.0.0", g.Version)
}

func TestCreateGraphInvalidVersion(t *testing.T) {
	uc, _, _ := newGraphUC()
	in := validCreateInput()
	in.Version = "v1"
	_, err := uc.CreateGraph(context.Background(), in)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestCreateGraphDuplicateVersion(t *testing.T) {
	uc, _, _ := newGraphUC()
	in := validCreateInput()
	_, err := uc.CreateGraph(context.Background(), in)
	require.NoError(t, err)
	_, err = uc.CreateGraph(context.Background(), in)
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestUpdateGraphOnlyDraft(t *testing.T) {
	uc, repo, _ := newGraphUC()
	g, err := uc.CreateGraph(context.Background(), validCreateInput())
	require.NoError(t, err)

	g.Status = domain.GraphStatusActive
	_, err = repo.Update(context.Background(), g)
	require.NoError(t, err)

	in := usecase.UpdateGraphInput{Name: "updated", Version: "1.0.1", EntryNode: "start", Definition: json.RawMessage(`{}`)}
	_, err = uc.UpdateGraph(context.Background(), g.ID, in)
	require.ErrorIs(t, err, domain.ErrInvalidGraphStatus)
}

func TestArchiveGraphSuccess(t *testing.T) {
	uc, _, _ := newGraphUC()
	g, err := uc.CreateGraph(context.Background(), validCreateInput())
	require.NoError(t, err)
	err = uc.ArchiveGraph(context.Background(), g.ID)
	require.NoError(t, err)
}

func TestArchiveGraphWithActiveExecution(t *testing.T) {
	uc, _, execRepo := newGraphUC()
	g, err := uc.CreateGraph(context.Background(), validCreateInput())
	require.NoError(t, err)

	_, err = execRepo.Create(context.Background(), &domain.Execution{
		ID:      uuid.New(),
		GraphID: g.ID,
		Status:  domain.ExecutionStatusRunning,
	})
	require.NoError(t, err)

	err = uc.ArchiveGraph(context.Background(), g.ID)
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestArchiveGraphNotFound(t *testing.T) {
	uc, _, _ := newGraphUC()
	err := uc.ArchiveGraph(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrNotFound)
}
