package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/orchestrator/internal/handler"
	"github.com/aescanero/dago/services/orchestrator/internal/usecase"
)

type fakeExecUC struct {
	startFn func(context.Context, usecase.StartExecutionInput) (*domain.Execution, error)
}

func (f *fakeExecUC) StartExecution(ctx context.Context, in usecase.StartExecutionInput) (*domain.Execution, error) {
	return f.startFn(ctx, in)
}

func newExecRouter(uc handler.ExecutionUseCasePort) *gin.Engine {
	r := gin.New()
	h := handler.NewExecutionHandler(uc)
	r.POST("/api/v1/executions", h.Start)
	return r
}

func stubExecution(graphID uuid.UUID) *domain.Execution {
	return &domain.Execution{
		ID:          uuid.New(),
		GraphID:     graphID,
		Status:      domain.ExecutionStatusPending,
		Variables:   json.RawMessage("{}"),
		Messages:    json.RawMessage("[]"),
		NodeResults: json.RawMessage("{}"),
	}
}

func TestExecutionHandlerStart(t *testing.T) {
	graphID := uuid.New()
	expected := stubExecution(graphID)
	uc := &fakeExecUC{startFn: func(_ context.Context, _ usecase.StartExecutionInput) (*domain.Execution, error) {
		return expected, nil
	}}
	r := newExecRouter(uc)
	body, _ := json.Marshal(map[string]any{"graph_id": graphID.String()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, expected.ID.String(), resp["id"])
	assert.Equal(t, graphID.String(), resp["graph_id"])
	assert.Equal(t, "pending", resp["status"])
}

func TestExecutionHandlerStartGraphNotFound(t *testing.T) {
	uc := &fakeExecUC{startFn: func(_ context.Context, _ usecase.StartExecutionInput) (*domain.Execution, error) {
		return nil, domain.ErrNotFound
	}}
	r := newExecRouter(uc)
	body, _ := json.Marshal(map[string]any{"graph_id": uuid.New().String()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
