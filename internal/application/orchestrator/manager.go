package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/domain/graph"
	"github.com/aescanero/dago-libs/pkg/ports"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Event topics for worker communication
const (
	TopicExecutorWork   = "executor.work"
	TopicRouterWork     = "router.work"
	TopicNodeCompleted  = "node.completed"
	TopicGraphEvents    = "graph.events"
)

// Manager coordinates graph execution by publishing work to workers
// and listening for completion events
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

	// Context for subscriptions
	ctx    context.Context
	cancel context.CancelFunc
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
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		eventBus:     eventBus,
		storage:      storage,
		metrics:      metrics,
		validator:    validator,
		logger:       logger,
		graphTimeout: graphTimeout,
		nodeTimeout:  nodeTimeout,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start initializes the manager and starts listening for events
func (m *Manager) Start() error {
	m.logger.Info("starting orchestrator manager")

	// Subscribe to node completion events from workers
	if err := m.eventBus.Subscribe(m.ctx, TopicNodeCompleted, m.handleNodeCompleted); err != nil {
		return fmt.Errorf("failed to subscribe to node completed events: %w", err)
	}

	m.logger.Info("orchestrator manager started, listening for node completion events")
	return nil
}

// SubmitGraph validates and submits a graph for execution
func (m *Manager) SubmitGraph(ctx context.Context, g *domain.Graph, inputs map[string]interface{}) (string, error) {
	// Validate graph structure
	if err := m.validator.Validate(g); err != nil {
		m.logger.Error("graph validation failed",
			zap.String("graph_id", g.ID),
			zap.Error(err))
		m.metrics.RecordGraphSubmitted(string(domain.ExecutionStatusFailed))
		return "", fmt.Errorf("validation failed: %w", err)
	}

	// Generate execution ID
	graphID := uuid.New().String()

	// Create initial state
	state := &domain.GraphState{
		GraphID:     graphID,
		Graph:       g,
		Status:      domain.ExecutionStatusRunning,
		Inputs:      inputs,
		NodeStates:  make(map[string]*domain.NodeState),
		SubmittedAt: time.Now(),
	}

	// Initialize node states
	for nodeID := range g.Nodes {
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
	if err := m.publishGraphEvent(ctx, graphID, domain.EventTypeGraphSubmitted, map[string]interface{}{
		"original_graph_id": g.ID,
	}); err != nil {
		return "", err
	}

	// Track execution
	execCtx, cancel := context.WithTimeout(context.Background(), m.graphTimeout)
	m.executions.Store(graphID, &executionContext{
		graphID:    graphID,
		status:     domain.ExecutionStatusRunning,
		startedAt:  time.Now(),
		cancelFunc: cancel,
	})

	m.metrics.RecordGraphSubmitted(string(domain.ExecutionStatusSubmitted))
	m.logger.Info("graph submitted",
		zap.String("graph_id", graphID),
		zap.String("original_graph_id", g.ID),
		zap.String("entry_node", g.EntryNode))

	// Start execution monitoring in background
	go m.monitorExecution(execCtx, graphID)

	// Publish work for entry node
	if err := m.publishNodeWork(ctx, graphID, g.EntryNode, state); err != nil {
		m.logger.Error("failed to publish entry node work",
			zap.String("graph_id", graphID),
			zap.String("node_id", g.EntryNode),
			zap.Error(err))
		return graphID, nil // Return graphID even on error, execution will timeout
	}

	return graphID, nil
}

// handleNodeCompleted processes node completion events from workers
func (m *Manager) handleNodeCompleted(ctx context.Context, event ports.Event) error {
	graphID := event.ExecutionID
	nodeID, _ := event.Data["node_id"].(string)
	output := event.Data["output"]
	errorMsg, hasError := event.Data["error"].(string)
	nextNodeID, _ := event.Data["next_node"].(string) // For router nodes

	m.logger.Info("received node completed event",
		zap.String("graph_id", graphID),
		zap.String("node_id", nodeID),
		zap.Bool("has_error", hasError),
		zap.String("next_node", nextNodeID))

	// Get current state
	stateInterface, err := m.storage.GetState(ctx, graphID)
	if err != nil {
		m.logger.Error("failed to get state on node completion",
			zap.String("graph_id", graphID),
			zap.Error(err))
		return nil // Don't return error to avoid reprocessing
	}

	state, ok := stateInterface.(*domain.GraphState)
	if !ok {
		m.logger.Error("invalid state type",
			zap.String("graph_id", graphID))
		return nil
	}

	// Update node state
	nodeState := state.NodeStates[nodeID]
	if nodeState == nil {
		m.logger.Error("node state not found",
			zap.String("graph_id", graphID),
			zap.String("node_id", nodeID))
		return nil
	}

	now := time.Now()
	nodeState.CompletedAt = &now

	if hasError {
		nodeState.Status = domain.ExecutionStatusFailed
		nodeState.Error = errorMsg
	} else {
		nodeState.Status = domain.ExecutionStatusCompleted
		nodeState.Output = output
	}

	// Save state
	if err := m.storage.SaveState(ctx, state); err != nil {
		m.logger.Error("failed to save state after node completion",
			zap.String("graph_id", graphID),
			zap.Error(err))
	}

	// If node failed, mark graph as failed
	if hasError {
		m.completeGraph(ctx, graphID, state, domain.ExecutionStatusFailed, errorMsg)
		return nil
	}

	// Determine next node
	var nextNode string

	if nextNodeID != "" {
		// Router provided next node
		nextNode = nextNodeID
	} else {
		// Find next node from edges
		nextNode = m.findNextNode(state.Graph, nodeID)
	}

	if nextNode == "" {
		// No more nodes, graph complete
		m.completeGraph(ctx, graphID, state, domain.ExecutionStatusCompleted, "")
		return nil
	}

	// Publish work for next node
	if err := m.publishNodeWork(ctx, graphID, nextNode, state); err != nil {
		m.logger.Error("failed to publish next node work",
			zap.String("graph_id", graphID),
			zap.String("node_id", nextNode),
			zap.Error(err))
	}

	return nil
}

// findNextNode finds the next node to execute based on edges
func (m *Manager) findNextNode(g *domain.Graph, currentNodeID string) string {
	edges := g.GetOutgoingEdges(currentNodeID)
	if len(edges) == 0 {
		return ""
	}
	// For now, just take the first edge (linear flow)
	// Router nodes will provide next_node explicitly
	return edges[0].To
}

// publishNodeWork publishes a work event for a node
func (m *Manager) publishNodeWork(ctx context.Context, graphID, nodeID string, state *domain.GraphState) error {
	node := state.Graph.GetNode(nodeID)
	if node == nil {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// Update node state to running
	nodeState := state.NodeStates[nodeID]
	now := time.Now()
	nodeState.Status = domain.ExecutionStatusRunning
	nodeState.StartedAt = &now

	if err := m.storage.SaveState(ctx, state); err != nil {
		m.logger.Error("failed to save state before node work",
			zap.String("graph_id", graphID),
			zap.String("node_id", nodeID),
			zap.Error(err))
	}

	// Determine topic based on node type
	var topic string
	switch node.GetType() {
	case graph.NodeTypeExecutor:
		topic = TopicExecutorWork
	case graph.NodeTypeRouter:
		topic = TopicRouterWork
	default:
		// For other types (start, end), find next node
		if node.GetType() == graph.NodeTypeEnd {
			m.completeGraph(ctx, graphID, state, domain.ExecutionStatusCompleted, "")
			return nil
		}
		// For start node, find next
		nextNode := m.findNextNode(state.Graph, nodeID)
		if nextNode != "" {
			return m.publishNodeWork(ctx, graphID, nextNode, state)
		}
		return nil
	}

	// Build work event
	event := ports.Event{
		ID:          uuid.New().String(),
		Type:        ports.EventType("node.work"),
		Timestamp:   time.Now(),
		ExecutionID: graphID,
		Data: map[string]interface{}{
			"node_id":    nodeID,
			"node_type":  string(node.GetType()),
			"graph_id":   graphID,
			"state":      state.Inputs,
			"node_state": state.NodeStates,
		},
	}

	m.logger.Info("publishing node work",
		zap.String("topic", topic),
		zap.String("graph_id", graphID),
		zap.String("node_id", nodeID),
		zap.String("node_type", string(node.GetType())))

	if err := m.eventBus.Publish(ctx, topic, event); err != nil {
		return fmt.Errorf("failed to publish work event: %w", err)
	}

	// Publish node started event
	m.publishGraphEvent(ctx, graphID, domain.EventTypeNodeStarted, map[string]interface{}{
		"node_id": nodeID,
	})

	return nil
}

// completeGraph marks a graph execution as complete
func (m *Manager) completeGraph(ctx context.Context, graphID string, state *domain.GraphState, status domain.ExecutionStatus, errorMsg string) {
	now := time.Now()
	state.Status = status
	state.CompletedAt = &now
	if errorMsg != "" {
		state.Error = errorMsg
	}

	if err := m.storage.SaveState(ctx, state); err != nil {
		m.logger.Error("failed to save final state",
			zap.String("graph_id", graphID),
			zap.Error(err))
	}

	// Cancel execution context
	if val, ok := m.executions.Load(graphID); ok {
		execCtx := val.(*executionContext)
		execCtx.cancelFunc()
		m.executions.Delete(graphID)
	}

	// Publish completion event
	eventType := domain.EventTypeGraphCompleted
	if status == domain.ExecutionStatusFailed {
		eventType = domain.EventTypeGraphFailed
	}

	data := map[string]interface{}{}
	if errorMsg != "" {
		data["error"] = errorMsg
	}

	m.publishGraphEvent(ctx, graphID, eventType, data)

	m.logger.Info("graph execution completed",
		zap.String("graph_id", graphID),
		zap.String("status", string(status)))

	// Record metrics
	m.metrics.RecordGraphCompleted(string(status), time.Since(state.SubmittedAt))
}

// publishGraphEvent publishes a graph-level event
func (m *Manager) publishGraphEvent(ctx context.Context, graphID string, eventType domain.EventType, data map[string]interface{}) error {
	event := ports.Event{
		ID:          uuid.New().String(),
		Type:        ports.EventType(eventType),
		Timestamp:   time.Now(),
		ExecutionID: graphID,
		Data:        data,
	}

	if err := m.eventBus.Publish(ctx, TopicGraphEvents, event); err != nil {
		m.logger.Error("failed to publish graph event",
			zap.String("graph_id", graphID),
			zap.String("event_type", string(eventType)),
			zap.Error(err))
		return err
	}
	return nil
}

// GetStatus retrieves the current status of a graph execution
func (m *Manager) GetStatus(ctx context.Context, graphID string) (*domain.GraphState, error) {
	stateInterface, err := m.storage.GetState(ctx, graphID)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

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
	m.publishGraphEvent(ctx, graphID, domain.EventTypeGraphCancelled, nil)

	m.executions.Delete(graphID)

	m.logger.Info("graph execution cancelled",
		zap.String("graph_id", graphID))

	return nil
}

// monitorExecution monitors graph execution and handles timeouts
func (m *Manager) monitorExecution(ctx context.Context, graphID string) {
	<-ctx.Done()

	// Check if it's a timeout (not cancelled)
	if ctx.Err() == context.DeadlineExceeded {
		m.handleTimeout(graphID)
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

	state, ok := stateInterface.(*domain.GraphState)
	if !ok {
		m.logger.Error("invalid state type during timeout",
			zap.String("graph_id", graphID))
		return
	}

	m.completeGraph(ctx, graphID, state, domain.ExecutionStatusFailed, "execution timeout")
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info("shutting down orchestrator manager")

	// Cancel subscriptions
	m.cancel()

	// Cancel all active executions
	m.executions.Range(func(key, value interface{}) bool {
		execCtx := value.(*executionContext)
		execCtx.cancelFunc()
		return true
	})

	m.logger.Info("orchestrator manager shut down complete")
	return nil
}
