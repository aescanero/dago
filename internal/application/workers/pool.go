package workers

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

// Pool manages a pool of worker goroutines
type Pool struct {
	size       int
	eventBus   ports.EventBus
	storage    ports.StateStorage
	llmClient  ports.LLMClient
	metrics    ports.MetricsCollector
	logger     *zap.Logger
	health     *HealthMonitor

	workers []*worker
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// worker represents a single worker goroutine
type worker struct {
	id      string
	pool    *Pool
	status  WorkerStatus
	mu      sync.RWMutex
	lastJob time.Time
}

// WorkerStatus represents worker status
type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusBusy    WorkerStatus = "busy"
	WorkerStatusStopped WorkerStatus = "stopped"
)

// NewPool creates a new worker pool
func NewPool(
	size int,
	eventBus ports.EventBus,
	storage ports.StateStorage,
	llmClient ports.LLMClient,
	metrics ports.MetricsCollector,
	logger *zap.Logger,
	healthCheckInterval time.Duration,
) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &Pool{
		size:      size,
		eventBus:  eventBus,
		storage:   storage,
		llmClient: llmClient,
		metrics:   metrics,
		logger:    logger,
		workers:   make([]*worker, size),
		ctx:       ctx,
		cancel:    cancel,
	}

	pool.health = NewHealthMonitor(pool, healthCheckInterval, logger)

	return pool
}

// Start starts the worker pool
func (p *Pool) Start() error {
	p.logger.Info("starting worker pool", zap.Int("size", p.size))

	// Create and start workers
	for i := 0; i < p.size; i++ {
		w := &worker{
			id:      fmt.Sprintf("worker-%d", i),
			pool:    p,
			status:  WorkerStatusIdle,
			lastJob: time.Now(),
		}
		p.workers[i] = w

		p.wg.Add(1)
		go w.run(p.ctx)
	}

	// Start health monitor
	p.health.Start()

	p.logger.Info("worker pool started", zap.Int("workers", p.size))
	return nil
}

// Shutdown gracefully shuts down the worker pool
func (p *Pool) Shutdown(ctx context.Context) error {
	p.logger.Info("shutting down worker pool")

	// Stop health monitor
	p.health.Stop()

	// Cancel context to signal workers to stop
	p.cancel()

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("worker pool shut down complete")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout")
	}
}

// GetStatus returns the status of all workers
func (p *Pool) GetStatus() map[string]WorkerStatus {
	status := make(map[string]WorkerStatus)
	for _, w := range p.workers {
		w.mu.RLock()
		status[w.id] = w.status
		w.mu.RUnlock()
	}
	return status
}

// run is the main worker loop
func (w *worker) run(ctx context.Context) {
	defer w.pool.wg.Done()

	w.pool.logger.Info("worker started", zap.String("worker_id", w.id))

	// Subscribe to node execution events
	eventHandler := func(ctx context.Context, event ports.Event) error {
		// Convert ports.Event to domain.Event for internal processing
		domainEvent := &domain.Event{
			ID:        event.ID,
			Type:      domain.EventType(event.Type),
			GraphID:   event.ExecutionID,
			Timestamp: event.Timestamp,
			Data:      event.Data,
		}

		// Handle event asynchronously
		go w.handleNodeExecution(ctx, domainEvent)
		return nil
	}

	if err := w.pool.eventBus.Subscribe(ctx, "node.events", eventHandler); err != nil {
		w.pool.logger.Error("failed to subscribe to events",
			zap.String("worker_id", w.id),
			zap.Error(err))
		return
	}

	// Wait for context cancellation
	<-ctx.Done()
	w.mu.Lock()
	w.status = WorkerStatusStopped
	w.mu.Unlock()
	w.pool.logger.Info("worker stopped", zap.String("worker_id", w.id))
}

// handleNodeExecution processes a node execution request
func (w *worker) handleNodeExecution(ctx context.Context, event *domain.Event) {
	w.mu.Lock()
	w.status = WorkerStatusBusy
	w.lastJob = time.Now()
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.status = WorkerStatusIdle
		w.mu.Unlock()
	}()

	graphID := event.GraphID
	nodeID, ok := event.Data["node_id"].(string)
	if !ok {
		w.pool.logger.Error("invalid node_id in event",
			zap.String("worker_id", w.id),
			zap.String("event_id", event.ID))
		return
	}

	w.pool.logger.Info("executing node",
		zap.String("worker_id", w.id),
		zap.String("graph_id", graphID),
		zap.String("node_id", nodeID))

	startTime := time.Now()

	// Get current state
	stateInterface, err := w.pool.storage.GetState(ctx, graphID)
	if err != nil {
		w.pool.logger.Error("failed to get state",
			zap.String("worker_id", w.id),
			zap.String("graph_id", graphID),
			zap.Error(err))
		return
	}

	// Type assert to GraphState
	state, ok := stateInterface.(*domain.GraphState)
	if !ok {
		w.pool.logger.Error("invalid state type",
			zap.String("worker_id", w.id),
			zap.String("graph_id", graphID))
		return
	}

	// Find the node - Graph.Nodes is a map[string]Node now
	node, exists := state.Graph.Nodes[nodeID]
	if !exists {
		w.pool.logger.Error("node not found",
			zap.String("worker_id", w.id),
			zap.String("graph_id", graphID),
			zap.String("node_id", nodeID))
		return
	}

	// Update node state to running
	nodeState := state.NodeStates[nodeID]
	now := time.Now()
	nodeState.Status = domain.ExecutionStatusRunning
	nodeState.StartedAt = &now

	if err := w.pool.storage.SaveState(ctx, state); err != nil {
		w.pool.logger.Error("failed to save state",
			zap.String("worker_id", w.id),
			zap.String("graph_id", graphID),
			zap.Error(err))
		return
	}

	// Publish node started event
	w.publishEvent(ctx, graphID, nodeID, domain.EventTypeNodeStarted, nil)

	// Execute node based on type - Convert node interface to domain.Node for execution
	var result interface{}
	var execErr error

	// For MVP, only support basic node execution through config
	result, execErr = w.executeGenericNode(ctx, nodeID, node, state)

	duration := time.Since(startTime)

	// Update node state with result
	nodeState = state.NodeStates[nodeID]
	completedAt := time.Now()
	if execErr != nil {
		nodeState.Status = domain.ExecutionStatusFailed
		nodeState.Error = execErr.Error()
		w.publishEvent(ctx, graphID, nodeID, domain.EventTypeNodeFailed, map[string]interface{}{
			"error": execErr.Error(),
		})
		w.pool.metrics.RecordNodeExecuted(string(domain.ExecutionStatusFailed), duration)
	} else {
		nodeState.Status = domain.ExecutionStatusCompleted
		nodeState.Output = result
		w.publishEvent(ctx, graphID, nodeID, domain.EventTypeNodeCompleted, map[string]interface{}{
			"output": result,
		})
		w.pool.metrics.RecordNodeExecuted(string(domain.ExecutionStatusCompleted), duration)
	}

	nodeState.CompletedAt = &completedAt

	// Save final state
	if err := w.pool.storage.SaveState(ctx, state); err != nil {
		w.pool.logger.Error("failed to save final state",
			zap.String("worker_id", w.id),
			zap.String("graph_id", graphID),
			zap.Error(err))
	}

	w.pool.logger.Info("node execution completed",
		zap.String("worker_id", w.id),
		zap.String("graph_id", graphID),
		zap.String("node_id", nodeID),
		zap.String("status", string(nodeState.Status)),
		zap.Duration("duration", duration))
}

// executeGenericNode executes a generic node using LLM
func (w *worker) executeGenericNode(ctx context.Context, nodeID string, node interface{}, state *domain.GraphState) (interface{}, error) {
	// For MVP, execute using LLM with simple config
	// In production, you'd have proper node type handling

	// Build simple request
	req := &domain.LLMRequest{
		Model:       "claude-3-5-sonnet-20241022",
		Messages:    []domain.Message{},
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// Build user message from inputs
	userMessage := fmt.Sprintf("Execute node %s", nodeID)
	req.Messages = append(req.Messages, domain.Message{
		Role:    "user",
		Content: userMessage,
	})

	// Call LLM
	respInterface, err := w.pool.llmClient.GenerateCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Type assert response
	resp, ok := respInterface.(*domain.LLMResponse)
	if !ok {
		return nil, fmt.Errorf("invalid LLM response type")
	}

	return resp.Content, nil
}

// publishEvent publishes an event to the event bus
func (w *worker) publishEvent(ctx context.Context, graphID, nodeID string, eventType domain.EventType, data map[string]interface{}) {
	event := &domain.Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		GraphID:   graphID,
		NodeID:    nodeID,
		Timestamp: time.Now(),
		Data:      data,
	}

	// Convert domain.Event to ports.Event
	portsEvent := ports.Event{
		ID:          event.ID,
		Type:        ports.EventType(event.Type),
		Timestamp:   event.Timestamp,
		ExecutionID: event.GraphID,
		Data:        event.Data,
	}

	if err := w.pool.eventBus.Publish(ctx, "node.events", portsEvent); err != nil {
		w.pool.logger.Error("failed to publish event",
			zap.String("worker_id", w.id),
			zap.String("event_type", string(eventType)),
			zap.Error(err))
	}
}
