package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/orchestrator/internal/usecase"
)

// ExecutionUseCasePort is the interface the handler depends on.
type ExecutionUseCasePort interface {
	StartExecution(ctx context.Context, in usecase.StartExecutionInput) (*domain.Execution, error)
}

// ExecutionHandler handles execution REST endpoints.
type ExecutionHandler struct {
	uc ExecutionUseCasePort
}

// NewExecutionHandler builds an ExecutionHandler.
func NewExecutionHandler(uc ExecutionUseCasePort) *ExecutionHandler {
	return &ExecutionHandler{uc: uc}
}

type executionInput struct {
	GraphID   uuid.UUID       `json:"graph_id" binding:"required"`
	Variables json.RawMessage `json:"variables"`
}

type executionResponse struct {
	ID          uuid.UUID       `json:"id"`
	GraphID     uuid.UUID       `json:"graph_id"`
	Status      string          `json:"status"`
	CurrentNode string          `json:"current_node,omitempty"`
	Variables   json.RawMessage `json:"variables"`
	Messages    json.RawMessage `json:"messages"`
	NodeResults json.RawMessage `json:"node_results"`
	Error       string          `json:"error,omitempty"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

func execResponseFrom(e *domain.Execution) executionResponse {
	return executionResponse{
		ID:          e.ID,
		GraphID:     e.GraphID,
		Status:      string(e.Status),
		CurrentNode: e.CurrentNode,
		Variables:   e.Variables,
		Messages:    e.Messages,
		NodeResults: e.NodeResults,
		Error:       e.Error,
		CreatedAt:   e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   e.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Start handles POST /api/v1/executions.
func (h *ExecutionHandler) Start(c *gin.Context) {
	var in executionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("INVALID_JSON", err.Error()))
		return
	}
	e, err := h.uc.StartExecution(c.Request.Context(), usecase.StartExecutionInput{
		GraphID:   in.GraphID,
		Variables: in.Variables,
	})
	if err != nil {
		mapDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, execResponseFrom(e))
}
