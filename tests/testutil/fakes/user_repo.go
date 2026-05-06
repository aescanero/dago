package fakes

import (
	"context"
	"sync"

	"github.com/aescanero/dago/libs/domain"
)

// InMemoryUserRepository is an in-memory fake for tests.
type InMemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User // keyed by email
}

// NewInMemoryUserRepository creates an empty in-memory user repository.
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{users: make(map[string]*domain.User)}
}

// Create stores a new user, returning ErrConflict if the email already exists.
func (r *InMemoryUserRepository) Create(_ context.Context, u *domain.User) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.users[u.Email]; exists {
		return nil, domain.ErrConflict
	}
	r.users[u.Email] = u
	return u, nil
}

// FindByEmail looks up a user by email, returning ErrNotFound if absent.
func (r *InMemoryUserRepository) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}
