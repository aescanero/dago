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

// CreateGraphInput carries validated data for graph creation.
type CreateGraphInput struct {
	Name         string
	Version      string
	Description  string
	EntryNode    string
	Definition   json.RawMessage
	MemoryConfig json.RawMessage
}

// UpdateGraphInput carries validated data for graph update.
type UpdateGraphInput = CreateGraphInput

// ListGraphsInput carries pagination and filter params.
type ListGraphsInput struct {
	Page    int
	PerPage int
	Status  string
}

// GraphUseCase implements graph business logic.
type GraphUseCase struct {
	repo     ports.GraphRepository
	execRepo ports.ExecutionRepository
}

// NewGraphUseCase builds a GraphUseCase with its dependencies.
func NewGraphUseCase(repo ports.GraphRepository, execRepo ports.ExecutionRepository) *GraphUseCase {
	return &GraphUseCase{repo: repo, execRepo: execRepo}
}

// CreateGraph validates and persists a new graph with status=draft.
func (u *GraphUseCase) CreateGraph(ctx context.Context, in CreateGraphInput) (*domain.Graph, error) {
	if err := validateSemver(in.Version); err != nil {
		return nil, err
	}
	g := &domain.Graph{
		ID:           uuid.New(),
		Name:         in.Name,
		Version:      in.Version,
		Description:  in.Description,
		EntryNode:    in.EntryNode,
		Definition:   in.Definition,
		MemoryConfig: in.MemoryConfig,
		Status:       domain.GraphStatusDraft,
	}
	created, err := u.repo.Create(ctx, g)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("CreateGraph: %w", err)
	}
	return created, nil
}

// GetGraph returns the graph with the given ID.
func (u *GraphUseCase) GetGraph(ctx context.Context, id uuid.UUID) (*domain.Graph, error) {
	g, err := u.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("GetGraph: %w", err)
	}
	return g, nil
}

// ListGraphs returns a paginated list of graphs.
func (u *GraphUseCase) ListGraphs(ctx context.Context, in ListGraphsInput) ([]*domain.Graph, int, error) {
	gs, total, err := u.repo.List(ctx, ports.ListOptions{Page: in.Page, PerPage: in.PerPage, Status: in.Status})
	if err != nil {
		return nil, 0, fmt.Errorf("ListGraphs: %w", err)
	}
	return gs, total, nil
}

// UpdateGraph applies changes to a graph that is in draft status.
func (u *GraphUseCase) UpdateGraph(ctx context.Context, id uuid.UUID, in UpdateGraphInput) (*domain.Graph, error) {
	g, err := u.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("UpdateGraph.find: %w", err)
	}
	if !g.IsDraft() {
		return nil, domain.ErrInvalidGraphStatus
	}
	g.Name = in.Name
	g.Version = in.Version
	g.Description = in.Description
	g.EntryNode = in.EntryNode
	g.Definition = in.Definition
	g.MemoryConfig = in.MemoryConfig
	updated, err := u.repo.Update(ctx, g)
	if err != nil {
		return nil, fmt.Errorf("UpdateGraph.save: %w", err)
	}
	return updated, nil
}

// ArchiveGraph sets the graph status to archived if it has no active executions.
func (u *GraphUseCase) ArchiveGraph(ctx context.Context, id uuid.UUID) error {
	if _, err := u.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("ArchiveGraph.find: %w", err)
	}
	count, err := u.execRepo.CountActiveByGraph(ctx, id)
	if err != nil {
		return fmt.Errorf("ArchiveGraph.count: %w", err)
	}
	if count > 0 {
		return domain.ErrConflict
	}
	if err := u.repo.Archive(ctx, id); err != nil {
		return fmt.Errorf("ArchiveGraph.archive: %w", err)
	}
	return nil
}
