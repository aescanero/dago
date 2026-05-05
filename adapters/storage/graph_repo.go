package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/aescanero/dago/ent"
	"github.com/aescanero/dago/ent/execution"
	entgraph "github.com/aescanero/dago/ent/graph"
	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

// EntGraphRepository implements ports.GraphRepository using Ent.
type EntGraphRepository struct {
	client *ent.Client
}

// NewEntGraphRepository creates a new EntGraphRepository.
func NewEntGraphRepository(client *ent.Client) *EntGraphRepository {
	return &EntGraphRepository{client: client}
}

// Create persists a new graph and returns it with its generated fields.
func (r *EntGraphRepository) Create(ctx context.Context, g *domain.Graph) (*domain.Graph, error) {
	q := r.client.Graph.Create().
		SetID(g.ID).
		SetName(g.Name).
		SetVersion(g.Version).
		SetEntryNode(g.EntryNode).
		SetDefinition(g.Definition).
		SetStatus(entgraph.Status(g.Status))
	if g.Description != "" {
		q = q.SetDescription(g.Description)
	}
	if g.MemoryConfig != nil {
		q = q.SetMemoryConfig(g.MemoryConfig)
	}
	created, err := q.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("%w: %w", domain.ErrConflict, err)
		}
		return nil, fmt.Errorf("storage.CreateGraph: %w", err)
	}
	return entGraphToDomain(created), nil
}

// FindByID returns the graph with the given ID or domain.ErrNotFound.
func (r *EntGraphRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Graph, error) {
	g, err := r.client.Graph.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: graph %s", domain.ErrNotFound, id)
		}
		return nil, fmt.Errorf("storage.FindGraph: %w", err)
	}
	return entGraphToDomain(g), nil
}

// List returns a paginated slice of graphs and the total count.
func (r *EntGraphRepository) List(ctx context.Context, opts ports.ListOptions) ([]*domain.Graph, int, error) {
	q := r.client.Graph.Query()
	if opts.Status != "" {
		q = q.Where(entgraph.StatusEQ(entgraph.Status(opts.Status)))
	}
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("storage.CountGraphs: %w", err)
	}
	page := opts.Page
	if page < 1 {
		page = 1
	}
	perPage := opts.PerPage
	if perPage < 1 {
		perPage = 20
	}
	rows, err := q.Offset((page - 1) * perPage).Limit(perPage).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("storage.ListGraphs: %w", err)
	}
	result := make([]*domain.Graph, len(rows))
	for i, row := range rows {
		result[i] = entGraphToDomain(row)
	}
	return result, total, nil
}

// Update persists changes to an existing graph.
func (r *EntGraphRepository) Update(ctx context.Context, g *domain.Graph) (*domain.Graph, error) {
	q := r.client.Graph.UpdateOneID(g.ID).
		SetName(g.Name).
		SetVersion(g.Version).
		SetEntryNode(g.EntryNode).
		SetDefinition(g.Definition).
		SetStatus(entgraph.Status(g.Status))
	if g.Description != "" {
		q = q.SetDescription(g.Description)
	} else {
		q = q.ClearDescription()
	}
	if g.MemoryConfig != nil {
		q = q.SetMemoryConfig(g.MemoryConfig)
	} else {
		q = q.ClearMemoryConfig()
	}
	updated, err := q.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: graph %s", domain.ErrNotFound, g.ID)
		}
		return nil, fmt.Errorf("storage.UpdateGraph: %w", err)
	}
	return entGraphToDomain(updated), nil
}

// Archive sets the graph status to archived (soft delete).
func (r *EntGraphRepository) Archive(ctx context.Context, id uuid.UUID) error {
	err := r.client.Graph.UpdateOneID(id).
		SetStatus(entgraph.StatusArchived).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("%w: graph %s", domain.ErrNotFound, id)
		}
		return fmt.Errorf("storage.ArchiveGraph: %w", err)
	}
	return nil
}

// EntExecutionRepository implements ports.ExecutionRepository using Ent.
type EntExecutionRepository struct {
	client *ent.Client
}

// NewEntExecutionRepository creates a new EntExecutionRepository.
func NewEntExecutionRepository(client *ent.Client) *EntExecutionRepository {
	return &EntExecutionRepository{client: client}
}

// Create persists a new execution record.
func (r *EntExecutionRepository) Create(ctx context.Context, e *domain.Execution) (*domain.Execution, error) {
	q := r.client.Execution.Create().
		SetID(e.ID).
		SetGraphID(e.GraphID).
		SetStatus(execution.Status(e.Status)).
		SetVariables(e.Variables).
		SetMessages(e.Messages).
		SetNodeResults(e.NodeResults)
	if e.CurrentNode != "" {
		q = q.SetCurrentNode(e.CurrentNode)
	}
	if e.Error != "" {
		q = q.SetError(e.Error)
	}
	created, err := q.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.CreateExecution: %w", err)
	}
	return entExecToDomain(created, e.GraphID), nil
}

// FindByID returns the execution with the given ID or domain.ErrNotFound.
func (r *EntExecutionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Execution, error) {
	e, err := r.client.Execution.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: execution %s", domain.ErrNotFound, id)
		}
		return nil, fmt.Errorf("storage.FindExecution: %w", err)
	}
	var graphID uuid.UUID
	if e.Edges.Graph != nil {
		graphID = e.Edges.Graph.ID
	}
	return entExecToDomain(e, graphID), nil
}

// CountActiveByGraph returns the number of running executions for the given graph.
func (r *EntExecutionRepository) CountActiveByGraph(ctx context.Context, graphID uuid.UUID) (int, error) {
	count, err := r.client.Execution.Query().
		Where(
			execution.HasGraphWith(entgraph.IDEQ(graphID)),
			execution.StatusEQ(execution.StatusRunning),
		).Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("storage.CountActiveExecutions: %w", err)
	}
	return count, nil
}

func entGraphToDomain(g *ent.Graph) *domain.Graph {
	desc := ""
	if g.Description != nil {
		desc = *g.Description
	}
	return &domain.Graph{
		ID:           g.ID,
		Name:         g.Name,
		Version:      g.Version,
		Description:  desc,
		EntryNode:    g.EntryNode,
		Definition:   g.Definition,
		MemoryConfig: g.MemoryConfig,
		Status:       domain.GraphStatus(g.Status),
		CreatedAt:    g.CreatedAt,
		UpdatedAt:    g.UpdatedAt,
	}
}

func entExecToDomain(e *ent.Execution, graphID uuid.UUID) *domain.Execution {
	d := &domain.Execution{
		ID:          e.ID,
		GraphID:     graphID,
		Status:      domain.ExecutionStatus(e.Status),
		Variables:   e.Variables,
		Messages:    e.Messages,
		NodeResults: e.NodeResults,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
	if e.CurrentNode != nil {
		d.CurrentNode = *e.CurrentNode
	}
	if e.Error != nil {
		d.Error = *e.Error
	}
	d.StartedAt = e.StartedAt
	d.CompletedAt = e.CompletedAt
	return d
}

// compile-time interface checks
var (
	_ ports.GraphRepository     = (*EntGraphRepository)(nil)
	_ ports.ExecutionRepository = (*EntExecutionRepository)(nil)
)
