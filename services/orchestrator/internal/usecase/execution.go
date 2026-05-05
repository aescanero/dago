package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

// StartExecutionInput carries validated data for starting an execution.
type StartExecutionInput struct {
	GraphID   uuid.UUID
	Variables json.RawMessage
}

// ExecutionUseCase implements execution business logic.
type ExecutionUseCase struct {
	graphRepo ports.GraphRepository
	execRepo  ports.ExecutionRepository
}

// NewExecutionUseCase builds an ExecutionUseCase with its dependencies.
func NewExecutionUseCase(graphRepo ports.GraphRepository, execRepo ports.ExecutionRepository) *ExecutionUseCase {
	return &ExecutionUseCase{graphRepo: graphRepo, execRepo: execRepo}
}

// StartExecution creates a pending execution for the given graph.
func (u *ExecutionUseCase) StartExecution(ctx context.Context, in StartExecutionInput) (*domain.Execution, error) {
	if _, err := u.graphRepo.FindByID(ctx, in.GraphID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("StartExecution.findGraph: %w", err)
	}
	vars := in.Variables
	if vars == nil {
		vars = json.RawMessage("{}")
	}
	e := &domain.Execution{
		ID:          uuid.New(),
		GraphID:     in.GraphID,
		Status:      domain.ExecutionStatusPending,
		Variables:   vars,
		Messages:    json.RawMessage("[]"),
		NodeResults: json.RawMessage("{}"),
	}
	created, err := u.execRepo.Create(ctx, e)
	if err != nil {
		return nil, fmt.Errorf("StartExecution.create: %w", err)
	}
	return created, nil
}
