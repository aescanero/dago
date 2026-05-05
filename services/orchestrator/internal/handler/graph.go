package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/orchestrator/internal/usecase"
)

// GraphUseCasePort is the interface the handler depends on (for testability).
type GraphUseCasePort interface {
	CreateGraph(ctx context.Context, in usecase.CreateGraphInput) (*domain.Graph, error)
	GetGraph(ctx context.Context, id uuid.UUID) (*domain.Graph, error)
	ListGraphs(ctx context.Context, in usecase.ListGraphsInput) ([]*domain.Graph, int, error)
	UpdateGraph(ctx context.Context, id uuid.UUID, in usecase.UpdateGraphInput) (*domain.Graph, error)
	ArchiveGraph(ctx context.Context, id uuid.UUID) error
}

// GraphHandler handles graph REST endpoints.
type GraphHandler struct {
	uc GraphUseCasePort
}

// NewGraphHandler builds a GraphHandler.
func NewGraphHandler(uc GraphUseCasePort) *GraphHandler {
	return &GraphHandler{uc: uc}
}

type graphInput struct {
	Name         string          `json:"name" binding:"required"`
	Version      string          `json:"version" binding:"required"`
	Description  string          `json:"description"`
	EntryNode    string          `json:"entry_node" binding:"required"`
	Definition   json.RawMessage `json:"definition" binding:"required"`
	MemoryConfig json.RawMessage `json:"memory_config"`
}

func (in *graphInput) toDomain() usecase.CreateGraphInput {
	return usecase.CreateGraphInput{
		Name:         in.Name,
		Version:      in.Version,
		Description:  in.Description,
		EntryNode:    in.EntryNode,
		Definition:   in.Definition,
		MemoryConfig: in.MemoryConfig,
	}
}

type graphResponse struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	Description  string          `json:"description"`
	EntryNode    string          `json:"entry_node"`
	Definition   json.RawMessage `json:"definition"`
	MemoryConfig json.RawMessage `json:"memory_config"`
	Status       string          `json:"status"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

func graphResponseFrom(g *domain.Graph) graphResponse {
	return graphResponse{
		ID:           g.ID,
		Name:         g.Name,
		Version:      g.Version,
		Description:  g.Description,
		EntryNode:    g.EntryNode,
		Definition:   g.Definition,
		MemoryConfig: g.MemoryConfig,
		Status:       string(g.Status),
		CreatedAt:    g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    g.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Create handles POST /api/v1/graphs.
func (h *GraphHandler) Create(c *gin.Context) {
	var in graphInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("INVALID_JSON", err.Error()))
		return
	}
	g, err := h.uc.CreateGraph(c.Request.Context(), in.toDomain())
	if err != nil {
		mapDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, graphResponseFrom(g))
}

// GetByID handles GET /api/v1/graphs/:id.
func (h *GraphHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("INVALID_UUID", "id must be a valid UUID"))
		return
	}
	g, err := h.uc.GetGraph(c.Request.Context(), id)
	if err != nil {
		mapDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, graphResponseFrom(g))
}

type listResponse struct {
	Items      []graphResponse `json:"items"`
	Pagination pagination      `json:"pagination"`
}

type pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// List handles GET /api/v1/graphs.
func (h *GraphHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	graphs, total, err := h.uc.ListGraphs(c.Request.Context(), usecase.ListGraphsInput{
		Page:    page,
		PerPage: perPage,
		Status:  c.Query("status"),
	})
	if err != nil {
		mapDomainError(c, err)
		return
	}
	items := make([]graphResponse, len(graphs))
	for i, g := range graphs {
		items[i] = graphResponseFrom(g)
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	c.JSON(http.StatusOK, listResponse{
		Items: items,
		Pagination: pagination{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// Update handles PUT /api/v1/graphs/:id.
func (h *GraphHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("INVALID_UUID", "id must be a valid UUID"))
		return
	}
	var in graphInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("INVALID_JSON", err.Error()))
		return
	}
	g, err := h.uc.UpdateGraph(c.Request.Context(), id, in.toDomain())
	if err != nil {
		mapDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, graphResponseFrom(g))
}

// Archive handles DELETE /api/v1/graphs/:id.
func (h *GraphHandler) Archive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("INVALID_UUID", "id must be a valid UUID"))
		return
	}
	if err := h.uc.ArchiveGraph(c.Request.Context(), id); err != nil {
		mapDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
