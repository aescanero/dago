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

func init() { gin.SetMode(gin.TestMode) }

// fakeGraphUC is a controllable mock of GraphUseCasePort.
type fakeGraphUC struct {
	createFn  func(context.Context, usecase.CreateGraphInput) (*domain.Graph, error)
	getFn     func(context.Context, uuid.UUID) (*domain.Graph, error)
	listFn    func(context.Context, usecase.ListGraphsInput) ([]*domain.Graph, int, error)
	updateFn  func(context.Context, uuid.UUID, usecase.UpdateGraphInput) (*domain.Graph, error)
	archiveFn func(context.Context, uuid.UUID) error
}

func (f *fakeGraphUC) CreateGraph(ctx context.Context, in usecase.CreateGraphInput) (*domain.Graph, error) {
	return f.createFn(ctx, in)
}

func (f *fakeGraphUC) GetGraph(ctx context.Context, id uuid.UUID) (*domain.Graph, error) {
	return f.getFn(ctx, id)
}

func (f *fakeGraphUC) ListGraphs(ctx context.Context, in usecase.ListGraphsInput) ([]*domain.Graph, int, error) {
	return f.listFn(ctx, in)
}

func (f *fakeGraphUC) UpdateGraph(ctx context.Context, id uuid.UUID, in usecase.UpdateGraphInput) (*domain.Graph, error) {
	return f.updateFn(ctx, id, in)
}

func (f *fakeGraphUC) ArchiveGraph(ctx context.Context, id uuid.UUID) error {
	return f.archiveFn(ctx, id)
}

func newGraphRouter(uc handler.GraphUseCasePort) *gin.Engine {
	r := gin.New()
	h := handler.NewGraphHandler(uc)
	r.POST("/api/v1/graphs", h.Create)
	r.GET("/api/v1/graphs", h.List)
	r.GET("/api/v1/graphs/:id", h.GetByID)
	r.PUT("/api/v1/graphs/:id", h.Update)
	r.DELETE("/api/v1/graphs/:id", h.Archive)
	return r
}

func stubGraph() *domain.Graph {
	return &domain.Graph{
		ID:         uuid.New(),
		Name:       "test",
		Version:    "1.0.0",
		EntryNode:  "start",
		Definition: json.RawMessage(`{}`),
		Status:     domain.GraphStatusDraft,
	}
}

func TestGraphHandlerCreate(t *testing.T) {
	expected := stubGraph()
	uc := &fakeGraphUC{createFn: func(_ context.Context, _ usecase.CreateGraphInput) (*domain.Graph, error) {
		return expected, nil
	}}
	r := newGraphRouter(uc)
	body, _ := json.Marshal(map[string]any{
		"name": "test", "version": "1.0.0", "entry_node": "start", "definition": map[string]any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/graphs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, expected.ID.String(), resp["id"])
	assert.Equal(t, "draft", resp["status"])
}

func TestGraphHandlerCreateInvalidBody(t *testing.T) {
	r := newGraphRouter(&fakeGraphUC{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/graphs", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGraphHandlerGetNotFound(t *testing.T) {
	uc := &fakeGraphUC{getFn: func(_ context.Context, _ uuid.UUID) (*domain.Graph, error) {
		return nil, domain.ErrNotFound
	}}
	r := newGraphRouter(uc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graphs/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "GRAPH_NOT_FOUND", resp["code"])
}

func TestGraphHandlerDelete(t *testing.T) {
	uc := &fakeGraphUC{archiveFn: func(_ context.Context, _ uuid.UUID) error { return nil }}
	r := newGraphRouter(uc)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/graphs/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestGraphHandlerDeleteConflict(t *testing.T) {
	uc := &fakeGraphUC{archiveFn: func(_ context.Context, _ uuid.UUID) error { return domain.ErrConflict }}
	r := newGraphRouter(uc)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/graphs/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}
