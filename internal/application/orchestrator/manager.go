package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/ports"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Manager coordinates graph execution
type Manager struct {
	eventBus   ports.EventBus
	storage    ports.StateStorage
	metrics    ports.MetricsCollector
	validator  *Validator
	logger     *zap.Logger

	// Track active executions
	executions sync.Map // map[string]*executionContext

	// Configuration
	graphTimeout time.Duration
	nodeTimeout  time.Duration
}

// executionContext holds state for a single graph execution
type executionContext struct {
	graphID    string
	status     domain.ExecutionStatus
	startedAt  time.Time
	cancelFunc context.CancelFunc
	mu         sync.RWMutex
}

// NewManager creates a new orchestrator manager
func NewManager(
	eventBus ports.EventBus,
	storage ports.StateStorage,
	metrics ports.MetricsCollector,
	validator *Validator,
	logger *zap.Logger,
	graphTimeout, nodeTimeout time.Duration,
) *Manager {
	return &Manager{
		eventBus:     eventBus,
		storage:      storage,
		metrics:      metrics,
		validator:    validator,
		logger:       logger,
		graphTimeout: graphTimeout,
		nodeTimeout:  nodeTimeout,
	}
}

// SubmitGraph validates and submits a graph for execution
func (m *Manager) SubmitGraph(ctx context.Context, graph *domain.Graph, inputs map[string]interface{}) (string, error) {
	// Validate graph structure
	if err := m.validator.Validate(graph); err != nil {
		m.logger.Error("graph validation failed",
			zap.String("graph_id", graph.ID),
			zap.Error(err))
		m.metrics.RecordGraphSubmitted(string(domain.ExecutionStatusFailed))
		return "", fmt.Errorf("validation failed: %w", err)
	}

	// Generate execution ID
	graphID := uuid.New().String()

	// Create initial state
	state := &domain.GraphState{
		GraphID:     graphID,
		Graph:       graph,
		Status:      domain.ExecutionStatusSubmitted,
		Inputs:      inputs,
		NodeStates:  make(map[string]*domain.NodeState),
		SubmittedAt: time.Now(),
	}

	// Initialize node states
	for nodeID := range graph.Nodes {
		state.NodeStates[nodeID] = &domain.NodeState{
			NodeID: nodeID,
			Status: domain.ExecutionStatusPending,
		}
	}

	// Store initial state
	if err := m.storage.SaveState(ctx, state); err != nil {
		m.logger.Error("failed to save initial state",
			zap.String("graph_id", graphID),
			zap.Error(err))
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	// Publish graph submitted event
	event := &domain.Event{
		ID:        uuid.New().String(),
		Type:      domain.EventTypeGraphSubmitted,
		GraphID:   graphID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"graph":  graph,
			"inputs": inputs,
		},
	}

	// Convert domain.Event to ports.Event
	portsEvent := ports.Event{
		ID:          event.ID,
		Type:        ports.EventType(event.Type),
		Timestamp:   event.Timestamp,
		ExecutionID: event.GraphID,
		Data:        event.Data,
	}

	if err := m.eventBus.Publish(ctx, "graph.events", portsEvent); err != nil {
		m.logger.Error("failed to publish graph submitted event",
			zap.String("graph_id", graphID),
			zap.Error(err))
		return "", fmt.Errorf("failed to publish event: %w", err)
	}

	// Track execution
	execCtx, cancel := context.WithTimeout(context.Background(), m.graphTimeout)
	m.executions.Store(graphID, &executionContext{
		graphID:    graphID,
		status:     domain.ExecutionStatusSubmitted,
		startedAt:  time.Now(),
		cancelFunc: cancel,
	})

	m.metrics.RecordGraphSubmitted(string(domain.ExecutionStatusSubmitted))
	m.logger.Info("graph submitted",
		zap.String("graph_id", graphID),
		zap.String("original_graph_id", graph.ID))

	// Start execution monitoring in background
	go m.monitorExecution(execCtx, graphID)

	return graphID, nil
}

// GetStatus retrieves the current status of a graph execution
func (m *Manager) GetStatus(ctx context.Context, graphID string) (*domain.GraphState, error) {
	stateInterface, err := m.storage.GetState(ctx, graphID)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	// Type assert to GraphState
	state, ok := stateInterface.(*domain.GraphState)
	if !ok {
		return nil, fmt.Errorf("invalid state type")
	}

	return state, nil
}

// CancelExecution cancels a running graph execution
func (m *Manager) CancelExecution(ctx context.Context, graphID string) error {
	// Get execution context
	val, ok := m.executions.Load(graphID)
	if !ok {
		return fmt.Errorf("execution not found: %s", graphID)
	}

	execCtx := val.(*executionContext)
	execCtx.mu.Lock()
	defer execCtx.mu.Unlock()

	// Check if already terminal state
	if execCtx.status == domain.ExecutionStatusCompleted ||
		execCtx.status == domain.ExecutionStatusFailed ||
		execCtx.status == domain.ExecutionStatusCancelled {
		return fmt.Errorf("execution already in terminal state: %s", execCtx.status)
	}

	// Cancel context
	execCtx.cancelFunc()
	execCtx.status = domain.ExecutionStatusCancelled

	// Update state in storage
	stateInterface, err := m.storage.GetState(ctx, graphID)
	if err != nil {
		return fmt.Errorf("failed to get state: %w", err)
	}

	// Type assert to GraphState
	state, ok := stateInterface.(*domain.GraphState)
	if !ok {
		return fmt.Errorf("invalid state type")
	}

	now := time.Now()
	state.Status = domain.ExecutionStatusCancelled
	state.CompletedAt = &now

	if err := m.storage.SaveState(ctx, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Publish cancellation event
	event := &domain.Event{
		ID:        uuid.New().String(),
		Type:      domain.EventTypeGraphCancelled,
		GraphID:   graphID,
		Timestamp: time.Now(),
	}

	// Convert to ports.Event
	portsEvent := ports.Event{
		ID:          event.ID,
		Type:        ports.EventType(event.Type),
		Timestamp:   event.Timestamp,
		ExecutionID: event.GraphID,
	}

	if err := m.eventBus.Publish(ctx, "graph.events", portsEvent); err != nil {
		m.logger.Error("failed to publish graph cancelled event",
			zap.String("graph_id", graphID),
			zap.Error(err))
	}

	m.logger.Info("graph execution cancelled",
		zap.String("graph_id", graphID))

	return nil
}

// monitorExecution monitors graph execution and handles timeouts
func (m *Manager) monitorExecution(ctx context.Context, graphID string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Check if it's a timeout
			if ctx.Err() == context.DeadlineExceeded {
				m.handleTimeout(graphID)
			}
			return

		case <-ticker.C:
			// Check execution status
			stateInterface, err := m.storage.GetState(context.Background(), graphID)
			if err != nil {
				m.logger.Error("failed to get state during monitoring",
					zap.String("graph_id", graphID),
					zap.Error(err))
				continue
			}

			// Type assert to GraphState
			state, ok := stateInterface.(*domain.GraphState)
			if !ok {
				m.logger.Error("invalid state type during monitoring",
					zap.String("graph_id", graphID))
				continue
			}

			// Check if execution is complete
			if state.Status == domain.ExecutionStatusCompleted ||
				state.Status == domain.ExecutionStatusFailed ||
				state.Status == domain.ExecutionStatusCancelled {
				m.executions.Delete(graphID)
				return
			}
		}
	}
}

// handleTimeout handles graph execution timeout
func (m *Manager) handleTimeout(graphID string) {
	m.logger.Warn("graph execution timed out",
		zap.String("graph_id", graphID))

	ctx := context.Background()

	// Update state
	stateInterface, err := m.storage.GetState(ctx, graphID)
	if err != nil {
		m.logger.Error("failed to get state during timeout",
			zap.String("graph_id", graphID),
			zap.Error(err))
		return
	}

	// Type assert to GraphState
	state, ok := stateInterface.(*domain.GraphState)
	if !ok {
		m.logger.Error("invalid state type during timeout",
			zap.String("graph_id", graphID))
		return
	}

	now := time.Now()
	state.Status = domain.ExecutionStatusFailed
	state.Error = "execution timeout"
	state.CompletedAt = &now

	if err := m.storage.SaveState(ctx, state); err != nil {
		m.logger.Error("failed to save state during timeout",
			zap.String("graph_id", graphID),
			zap.Error(err))
	}

	// Publish timeout event
	event := &domain.Event{
		ID:        uuid.New().String(),
		Type:      domain.EventTypeGraphFailed,
		GraphID:   graphID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"error": "execution timeout",
		},
	}

	// Convert to ports.Event
	portsEvent := ports.Event{
		ID:          event.ID,
		Type:        ports.EventType(event.Type),
		Timestamp:   event.Timestamp,
		ExecutionID: event.GraphID,
		Data:        event.Data,
	}

	if err := m.eventBus.Publish(ctx, "graph.events", portsEvent); err != nil {
		m.logger.Error("failed to publish timeout event",
			zap.String("graph_id", graphID),
			zap.Error(err))
	}

	m.executions.Delete(graphID)
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info("shutting down orchestrator manager")

	// Cancel all active executions
	m.executions.Range(func(key, value interface{}) bool {
		execCtx := value.(*executionContext)
		execCtx.cancelFunc()
		return true
	})

	m.logger.Info("orchestrator manager shut down complete")
	return nil
}
