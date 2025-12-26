package http

import (
	"net/http"
	"strings"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/ports"
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

// WorkerResponse represents the worker data format expected by the dashboard
type WorkerResponse struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	State         string                 `json:"state"`
	CurrentTask   string                 `json:"currentTask,omitempty"`
	PendingTasks  int                    `json:"pendingTasks"`
	HealthStatus  string                 `json:"healthStatus"`
	LastHeartbeat string                 `json:"lastHeartbeat"`
	StartedAt     string                 `json:"startedAt"`
	Version       string                 `json:"version,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// handleListWorkers handles listing workers
func (s *Server) handleListWorkers(c *gin.Context) {
	if s.registry == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: ErrorDetail{
				Code:    "REGISTRY_NOT_AVAILABLE",
				Message: "Worker registry is not configured",
			},
		})
		return
	}

	// Parse query parameters
	filter := ports.WorkerFilter{
		HealthyOnly: c.Query("healthy") == "true",
	}

	// Parse worker types filter
	if typeParam := c.Query("type"); typeParam != "" {
		types := strings.Split(typeParam, ",")
		for _, t := range types {
			filter.Types = append(filter.Types, ports.WorkerType(strings.TrimSpace(t)))
		}
	}

	// Parse status filter
	if statusParam := c.Query("status"); statusParam != "" {
		statuses := strings.Split(statusParam, ",")
		for _, s := range statuses {
			filter.Statuses = append(filter.Statuses, ports.WorkerStatus(strings.TrimSpace(s)))
		}
	}

	workers, err := s.registry.ListWorkers(c.Request.Context(), filter)
	if err != nil {
		s.logger.Error("failed to list workers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "REGISTRY_ERROR",
				Message: "Failed to retrieve workers",
				Details: err.Error(),
			},
		})
		return
	}

	// Convert to dashboard format
	workerResponses := make([]WorkerResponse, len(workers))
	for i, w := range workers {
		workerResponses[i] = workerInfoToResponse(w)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      workerResponses,
		"timestamp": gin.H{},
	})
}

// handleGetWorker handles getting a specific worker
func (s *Server) handleGetWorker(c *gin.Context) {
	if s.registry == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: ErrorDetail{
				Code:    "REGISTRY_NOT_AVAILABLE",
				Message: "Worker registry is not configured",
			},
		})
		return
	}

	workerID := c.Param("id")

	worker, err := s.registry.GetWorker(c.Request.Context(), workerID)
	if err != nil {
		s.logger.Error("failed to get worker", zap.String("worker_id", workerID), zap.Error(err))
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "WORKER_NOT_FOUND",
				Message: "Worker not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      workerInfoToResponse(*worker),
		"timestamp": gin.H{},
	})
}

// handleGetWorkerStats handles getting worker statistics
func (s *Server) handleGetWorkerStats(c *gin.Context) {
	if s.registry == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: ErrorDetail{
				Code:    "REGISTRY_NOT_AVAILABLE",
				Message: "Worker registry is not configured",
			},
		})
		return
	}

	// Get stats for both worker types
	executorStats, err := s.registry.GetWorkerStats(c.Request.Context(), ports.WorkerTypeExecutor)
	if err != nil {
		s.logger.Error("failed to get executor stats", zap.Error(err))
		executorStats = &ports.WorkerStats{
			Type:         ports.WorkerTypeExecutor,
			TotalWorkers: 0,
		}
	}

	routerStats, err := s.registry.GetWorkerStats(c.Request.Context(), ports.WorkerTypeRouter)
	if err != nil {
		s.logger.Error("failed to get router stats", zap.Error(err))
		routerStats = &ports.WorkerStats{
			Type:         ports.WorkerTypeRouter,
			TotalWorkers: 0,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"executor": executorStats,
			"router":   routerStats,
		},
		"timestamp": gin.H{},
	})
}

// handleGetWorkerPool handles getting worker pool status by type
func (s *Server) handleGetWorkerPool(c *gin.Context) {
	if s.registry == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: ErrorDetail{
				Code:    "REGISTRY_NOT_AVAILABLE",
				Message: "Worker registry is not configured",
			},
		})
		return
	}

	workerType := c.Param("type")
	if workerType != "executor" && workerType != "router" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_WORKER_TYPE",
				Message: "Worker type must be 'executor' or 'router'",
			},
		})
		return
	}

	var portWorkerType ports.WorkerType
	if workerType == "executor" {
		portWorkerType = ports.WorkerTypeExecutor
	} else {
		portWorkerType = ports.WorkerTypeRouter
	}

	stats, err := s.registry.GetWorkerStats(c.Request.Context(), portWorkerType)
	if err != nil {
		s.logger.Error("failed to get worker pool", zap.String("type", workerType), zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "REGISTRY_ERROR",
				Message: "Failed to retrieve worker pool",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"type":                workerType,
			"totalWorkers":        stats.TotalWorkers,
			"idleWorkers":         stats.IdleWorkers,
			"busyWorkers":         stats.BusyWorkers,
			"offlineWorkers":      stats.UnhealthyWorkers,
			"totalTasksCompleted": 0, // Not tracked yet
			"avgLatencyMs":        0, // Not tracked yet
			"totalErrors":         0, // Not tracked yet
		},
		"timestamp": gin.H{},
	})
}

// handleGetWorkerMetrics handles getting worker metrics
func (s *Server) handleGetWorkerMetrics(c *gin.Context) {
	// For MVP, return empty metrics
	// TODO: Implement metrics collection and storage
	c.JSON(http.StatusOK, gin.H{
		"data":      []interface{}{},
		"timestamp": gin.H{},
	})
}

// workerInfoToResponse converts ports.WorkerInfo to dashboard format
func workerInfoToResponse(w ports.WorkerInfo) WorkerResponse {
	// Map status to state
	state := "offline"
	switch w.Status {
	case ports.WorkerStatusIdle:
		state = "idle"
	case ports.WorkerStatusBusy:
		state = "busy"
	case ports.WorkerStatusUnhealthy:
		state = "offline"
	}

	// Map health status
	healthStatus := "healthy"
	if w.Status == ports.WorkerStatusUnhealthy {
		healthStatus = "unhealthy"
	}

	return WorkerResponse{
		ID:            w.ID,
		Type:          string(w.Type),
		State:         state,
		CurrentTask:   w.CurrentTask,
		PendingTasks:  w.PendingTasks,
		HealthStatus:  healthStatus,
		LastHeartbeat: w.LastHeartbeat.Format("2006-01-02T15:04:05Z07:00"),
		StartedAt:     w.RegisteredAt.Format("2006-01-02T15:04:05Z07:00"),
		Version:       w.Version,
		Metadata:      w.Metadata,
	}
}
