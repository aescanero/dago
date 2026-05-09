package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/aescanero/dago/services/orchestrator/internal/statemachine"
)

// StartExecutionInput carries validated data for starting an execution.
type StartExecutionInput struct {
	GraphID   uuid.UUID
	Variables json.RawMessage
	Auth      string
}

// ExecutionUseCase implements execution business logic.
type ExecutionUseCase struct {
	graphRepo ports.GraphRepository
	execRepo  ports.ExecutionRepository
	publisher ports.EventPublisher
}

// NewExecutionUseCase builds an ExecutionUseCase with its dependencies.
func NewExecutionUseCase(
	graphRepo ports.GraphRepository,
	execRepo ports.ExecutionRepository,
	publisher ports.EventPublisher,
) *ExecutionUseCase {
	return &ExecutionUseCase{graphRepo: graphRepo, execRepo: execRepo, publisher: publisher}
}

// StartExecution validates the graph, creates a running execution, and publishes the first node event.
func (u *ExecutionUseCase) StartExecution(ctx context.Context, in StartExecutionInput) (*domain.Execution, error) {
	g, err := u.graphRepo.FindByID(ctx, in.GraphID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("StartExecution.findGraph: %w", err)
	}

	var graphDef domain.GraphDefinition
	if err := json.Unmarshal(g.Definition, &graphDef); err != nil {
		return nil, fmt.Errorf("%w: unmarshal graph definition: %s", domain.ErrGraphValidation, err.Error())
	}

	if err := statemachine.ValidateGraph(graphDef); err != nil {
		return nil, fmt.Errorf("StartExecution.validateGraph: %w", err)
	}

	active, err := u.execRepo.CountActiveByGraph(ctx, in.GraphID)
	if err != nil {
		return nil, fmt.Errorf("StartExecution.countActive: %w", err)
	}
	if active > 0 {
		return nil, domain.ErrConflict
	}

	vars := in.Variables
	if vars == nil {
		vars = json.RawMessage("{}")
	}
	exec := &domain.Execution{
		ID:          uuid.New(),
		GraphID:     in.GraphID,
		Status:      domain.ExecutionStatusRunning,
		CurrentNode: graphDef.EntryNode,
		Variables:   vars,
		Messages:    json.RawMessage("[]"),
		NodeResults: json.RawMessage("{}"),
	}
	created, err := u.execRepo.Create(ctx, exec)
	if err != nil {
		return nil, fmt.Errorf("StartExecution.create: %w", err)
	}

	if err := u.publishFirstNode(ctx, created, graphDef, in.Auth); err != nil {
		return nil, fmt.Errorf("StartExecution.publishFirstNode: %w", err)
	}
	return created, nil
}

func (u *ExecutionUseCase) publishFirstNode(
	ctx context.Context,
	exec *domain.Execution,
	graphDef domain.GraphDefinition,
	auth string,
) error {
	entryNode := graphDef.Nodes[graphDef.EntryNode]
	payload, err := json.Marshal(map[string]any{
		"execution_id": exec.ID.String(),
		"graph_id":     exec.GraphID.String(),
		"node_id":      uuid.New().String(),
		"node_key":     graphDef.EntryNode,
		"pattern":      entryNode.Pattern,
		"config":       entryNode.Config,
		"auth":         auth,
	})
	if err != nil {
		return fmt.Errorf("marshal node.execute.requested payload: %w", err)
	}
	evt := domain.Event{
		ID:              uuid.New().String(),
		Type:            domain.EventTypeNodeExecuteRequested,
		Source:          "orchestrator",
		SpecVersion:     "1.0",
		Time:            exec.CreatedAt,
		DataContentType: "application/json",
		Data:            payload,
	}
	if err := u.publisher.Publish(ctx, evt, ports.PublishOptions{Stream: domain.StreamNodeExecuteRequested}); err != nil {
		return fmt.Errorf("publish node.execute.requested: %w", err)
	}
	return nil
}
