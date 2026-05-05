package fakes

import (
	"context"
	"fmt"
	"sync"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/google/uuid"
)

// InMemoryGraphRepository is a thread-safe in-memory implementation of ports.GraphRepository.
type InMemoryGraphRepository struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*domain.Graph
}

// NewInMemoryGraphRepository creates an empty in-memory graph repository.
func NewInMemoryGraphRepository() *InMemoryGraphRepository {
	return &InMemoryGraphRepository{data: make(map[uuid.UUID]*domain.Graph)}
}

func (r *InMemoryGraphRepository) Create(ctx context.Context, g *domain.Graph) (*domain.Graph, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.data {
		if existing.Name == g.Name && existing.Version == g.Version {
			return nil, fmt.Errorf("%w: graph %s@%s already exists", domain.ErrConflict, g.Name, g.Version)
		}
	}
	copy := *g
	r.data[g.ID] = &copy
	return &copy, nil
}

func (r *InMemoryGraphRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Graph, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.data[id]
	if !ok {
		return nil, fmt.Errorf("%w: graph %s", domain.ErrNotFound, id)
	}
	copy := *g
	return &copy, nil
}

func (r *InMemoryGraphRepository) List(ctx context.Context, opts ports.ListOptions) ([]*domain.Graph, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []*domain.Graph
	for _, g := range r.data {
		if opts.Status != "" && string(g.Status) != opts.Status {
			continue
		}
		copy := *g
		all = append(all, &copy)
	}
	total := len(all)
	page := opts.Page
	if page < 1 {
		page = 1
	}
	perPage := opts.PerPage
	if perPage < 1 {
		perPage = 20
	}
	start := (page - 1) * perPage
	if start >= total {
		return []*domain.Graph{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func (r *InMemoryGraphRepository) Update(ctx context.Context, g *domain.Graph) (*domain.Graph, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[g.ID]; !ok {
		return nil, fmt.Errorf("%w: graph %s", domain.ErrNotFound, g.ID)
	}
	copy := *g
	r.data[g.ID] = &copy
	return &copy, nil
}

func (r *InMemoryGraphRepository) Archive(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.data[id]
	if !ok {
		return fmt.Errorf("%w: graph %s", domain.ErrNotFound, id)
	}
	g.Status = domain.GraphStatusArchived
	return nil
}

// compile-time interface check
var _ ports.GraphRepository = (*InMemoryGraphRepository)(nil)
