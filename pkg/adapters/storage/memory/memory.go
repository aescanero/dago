package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/domain/state"
)

// InMemoryStateStorage implements StateStorage using in-memory map
// This is for testing purposes only
type InMemoryStateStorage struct {
	states map[string]interface{} // stores both state.State and domain.GraphState
	mu     sync.RWMutex
}

// NewInMemoryStateStorage creates a new in-memory state storage
func NewInMemoryStateStorage() *InMemoryStateStorage {
	return &InMemoryStateStorage{
		states: make(map[string]interface{}),
	}
}

// Save persists state for an execution (ports.StateStorage interface)
func (s *InMemoryStateStorage) Save(ctx context.Context, executionID string, st state.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[executionID] = st
	return nil
}

// Load retrieves state for an execution (ports.StateStorage interface)
func (s *InMemoryStateStorage) Load(ctx context.Context, executionID string) (state.State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.states[executionID]
	if !ok {
		return nil, fmt.Errorf("state not found: %s", executionID)
	}

	// Type assert to state.State (map[string]interface{})
	st, ok := data.(state.State)
	if !ok {
		// Try to convert if it's a map
		if m, ok := data.(map[string]interface{}); ok {
			return state.State(m), nil
		}
		return nil, fmt.Errorf("invalid state type in storage")
	}

	return st, nil
}

// Delete removes state for an execution (ports.StateStorage interface)
func (s *InMemoryStateStorage) Delete(ctx context.Context, executionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, executionID)
	return nil
}

// Exists checks if state exists for an execution (ports.StateStorage interface)
func (s *InMemoryStateStorage) Exists(ctx context.Context, executionID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.states[executionID]
	return ok, nil
}

// SetTTL sets a time-to-live for state data (ports.StateStorage interface)
// Note: In-memory storage doesn't implement TTL
func (s *InMemoryStateStorage) SetTTL(ctx context.Context, executionID string, ttl time.Duration) error {
	// No-op for in-memory storage
	return nil
}

// List returns all execution IDs that have stored state (ports.StateStorage interface)
func (s *InMemoryStateStorage) List(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	executionIDs := make([]string, 0, len(s.states))
	for id := range s.states {
		executionIDs = append(executionIDs, id)
	}

	return executionIDs, nil
}

// SaveState saves graph state to memory (compatibility method)
func (s *InMemoryStateStorage) SaveState(ctx context.Context, state interface{}) error {
	// Type assert to GraphState
	graphState, ok := state.(*domain.GraphState)
	if !ok {
		return fmt.Errorf("invalid state type")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Deep copy to avoid mutations
	stateCopy := *graphState
	s.states[graphState.GraphID] = &stateCopy

	return nil
}

// GetState retrieves graph state from memory (compatibility method)
func (s *InMemoryStateStorage) GetState(ctx context.Context, graphID string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.states[graphID]
	if !ok {
		return nil, fmt.Errorf("state not found: %s", graphID)
	}

	return state, nil
}
