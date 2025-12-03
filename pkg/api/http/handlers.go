package http

import (
	"net/http"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GraphSubmitRequest represents a graph submission request
type GraphSubmitRequest struct {
	Graph  *domain.Graph          `json:"graph" binding:"required"`
	Inputs map[string]interface{} `json:"inputs"`
}

// GraphSubmitResponse represents a graph submission response
type GraphSubmitResponse struct {
	GraphID     string `json:"graph_id"`
	Status      string `json:"status"`
	SubmittedAt string `json:"submitted_at"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": gin.H{},
		"checks": gin.H{
			"orchestrator": "ok",
		},
	})
}

// handleSubmitGraph handles graph submission
func (s *Server) handleSubmitGraph(c *gin.Context) {
	var req GraphSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	// Submit graph
	graphID, err := s.orchestrator.SubmitGraph(c.Request.Context(), req.Graph, req.Inputs)
	if err != nil {
		s.logger.Error("failed to submit graph", zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorDetail{
				Code:    "SUBMISSION_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, GraphSubmitResponse{
		GraphID:     graphID,
		Status:      "submitted",
		SubmittedAt: "", // Add timestamp
	})
}

// handleListGraphs handles listing graphs
func (s *Server) handleListGraphs(c *gin.Context) {
	// For MVP, return empty list
	// Full implementation would query storage
	c.JSON(http.StatusOK, gin.H{
		"graphs": []interface{}{},
		"total":  0,
		"limit":  20,
		"offset": 0,
	})
}

// handleGetGraph handles getting graph details
func (s *Server) handleGetGraph(c *gin.Context) {
	graphID := c.Param("id")

	state, err := s.orchestrator.GetStatus(c.Request.Context(), graphID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "NOT_FOUND",
				Message: "Graph not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, state)
}

// handleGetStatus handles getting graph status
func (s *Server) handleGetStatus(c *gin.Context) {
	graphID := c.Param("id")

	state, err := s.orchestrator.GetStatus(c.Request.Context(), graphID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "NOT_FOUND",
				Message: "Graph not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"graph_id":     state.GraphID,
		"status":       state.Status,
		"submitted_at": state.SubmittedAt,
		"started_at":   state.StartedAt,
		"completed_at": state.CompletedAt,
	})
}

// handleGetResult handles getting graph result
func (s *Server) handleGetResult(c *gin.Context) {
	graphID := c.Param("id")

	state, err := s.orchestrator.GetStatus(c.Request.Context(), graphID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "NOT_FOUND",
				Message: "Graph not found",
			},
		})
		return
	}

	if state.Status != domain.ExecutionStatusCompleted && state.Status != domain.ExecutionStatusFailed {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error: ErrorDetail{
				Code:    "NOT_COMPLETED",
				Message: "Graph execution not yet completed",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"graph_id":     state.GraphID,
		"status":       state.Status,
		"result":       state.NodeStates,
		"completed_at": state.CompletedAt,
	})
}

// handleCancelGraph handles graph cancellation
func (s *Server) handleCancelGraph(c *gin.Context) {
	graphID := c.Param("id")

	if err := s.orchestrator.CancelExecution(c.Request.Context(), graphID); err != nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error: ErrorDetail{
				Code:    "CANCELLATION_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"graph_id":    graphID,
		"status":      "cancelled",
		"cancelled_at": "", // Add timestamp
	})
}
