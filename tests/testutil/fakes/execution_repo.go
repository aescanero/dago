package fakes

import (
	"context"
	"fmt"
	"sync"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/google/uuid"
)

// InMemoryExecutionRepository is a thread-safe in-memory implementation of ports.ExecutionRepository.
type InMemoryExecutionRepository struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*domain.Execution
}

// NewInMemoryExecutionRepository creates an empty in-memory execution repository.
func NewInMemoryExecutionRepository() *InMemoryExecutionRepository {
	return &InMemoryExecutionRepository{data: make(map[uuid.UUID]*domain.Execution)}
}

func (r *InMemoryExecutionRepository) Create(ctx context.Context, e *domain.Execution) (*domain.Execution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *e
	r.data[e.ID] = &copy
	return &copy, nil
}

func (r *InMemoryExecutionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Execution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.data[id]
	if !ok {
		return nil, fmt.Errorf("%w: execution %s", domain.ErrNotFound, id)
	}
	copy := *e
	return &copy, nil
}

func (r *InMemoryExecutionRepository) CountActiveByGraph(ctx context.Context, graphID uuid.UUID) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, e := range r.data {
		if e.GraphID == graphID && e.Status == domain.ExecutionStatusRunning {
			count++
		}
	}
	return count, nil
}

// compile-time interface check
var _ ports.ExecutionRepository = (*InMemoryExecutionRepository)(nil)
