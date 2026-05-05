package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/aescanero/dago/libs/domain"
)

// ListOptions configures pagination and filters for List queries.
type ListOptions struct {
	Page    int
	PerPage int
	Status  string
}

// GraphRepository manages graph persistence.
type GraphRepository interface {
	Create(ctx context.Context, g *domain.Graph) (*domain.Graph, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Graph, error)
	List(ctx context.Context, opts ListOptions) ([]*domain.Graph, int, error)
	Update(ctx context.Context, g *domain.Graph) (*domain.Graph, error)
	Archive(ctx context.Context, id uuid.UUID) error
}

// ExecutionRepository manages execution persistence.
type ExecutionRepository interface {
	Create(ctx context.Context, e *domain.Execution) (*domain.Execution, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Execution, error)
	CountActiveByGraph(ctx context.Context, graphID uuid.UUID) (int, error)
}
