package usecase

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/google/uuid"
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
		return nil, err
	}
	return created, nil
}

func (u *GraphUseCase) GetGraph(ctx context.Context, id uuid.UUID) (*domain.Graph, error) {
	g, err := u.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return g, nil
}

func (u *GraphUseCase) ListGraphs(ctx context.Context, in ListGraphsInput) ([]*domain.Graph, int, error) {
	return u.repo.List(ctx, ports.ListOptions{
		Page:    in.Page,
		PerPage: in.PerPage,
		Status:  in.Status,
	})
}

func (u *GraphUseCase) UpdateGraph(ctx context.Context, id uuid.UUID, in UpdateGraphInput) (*domain.Graph, error) {
	g, err := u.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
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
	return u.repo.Update(ctx, g)
}

func (u *GraphUseCase) ArchiveGraph(ctx context.Context, id uuid.UUID) error {
	if _, err := u.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	count, err := u.execRepo.CountActiveByGraph(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrConflict
	}
	return u.repo.Archive(ctx, id)
}
