package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/domain/state"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// StateStorage implements StateStorage using Redis
type StateStorage struct {
	client *redis.Client
	logger *zap.Logger
	ttl    time.Duration
}

// NewStateStorage creates a new Redis state storage
func NewStateStorage(client *redis.Client, ttl time.Duration, logger *zap.Logger) *StateStorage {
	return &StateStorage{
		client: client,
		logger: logger,
		ttl:    ttl,
	}
}

// Save persists state for an execution (ports.StateStorage interface)
func (s *StateStorage) Save(ctx context.Context, executionID string, st state.State) error {
	key := getStateKey(executionID)

	// Serialize state
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Save to Redis with TTL
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// Load retrieves state for an execution (ports.StateStorage interface)
func (s *StateStorage) Load(ctx context.Context, executionID string) (state.State, error) {
	key := getStateKey(executionID)

	// Get from Redis
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("state not found: %s", executionID)
		}
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	// Deserialize to state
	var st state.State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return st, nil
}

// Delete removes state for an execution (ports.StateStorage interface)
func (s *StateStorage) Delete(ctx context.Context, executionID string) error {
	key := getStateKey(executionID)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	return nil
}

// Exists checks if state exists for an execution (ports.StateStorage interface)
func (s *StateStorage) Exists(ctx context.Context, executionID string) (bool, error) {
	key := getStateKey(executionID)

	result, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return result > 0, nil
}

// SetTTL sets a time-to-live for state data (ports.StateStorage interface)
func (s *StateStorage) SetTTL(ctx context.Context, executionID string, ttl time.Duration) error {
	key := getStateKey(executionID)

	if err := s.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL: %w", err)
	}

	return nil
}

// List returns all execution IDs that have stored state (ports.StateStorage interface)
func (s *StateStorage) List(ctx context.Context) ([]string, error) {
	pattern := "dago:state:*"

	var cursor uint64
	var keys []string

	for {
		var batch []string
		var err error

		batch, cursor, err = s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan keys: %w", err)
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	// Extract execution IDs from keys
	executionIDs := make([]string, 0, len(keys))
	prefix := "dago:state:"
	for _, key := range keys {
		if len(key) > len(prefix) {
			executionIDs = append(executionIDs, key[len(prefix):])
		}
	}

	return executionIDs, nil
}

// SaveState saves graph state to Redis (compatibility method)
func (s *StateStorage) SaveState(ctx context.Context, state interface{}) error {
	// Type assert to GraphState
	graphState, ok := state.(*domain.GraphState)
	if !ok {
		return fmt.Errorf("invalid state type")
	}

	key := getStateKey(graphState.GraphID)

	// Serialize state
	data, err := json.Marshal(graphState)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Save to Redis with TTL
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	s.logger.Debug("state saved",
		zap.String("graph_id", graphState.GraphID),
		zap.String("status", string(graphState.Status)))

	return nil
}

// GetState retrieves graph state from Redis (compatibility method)
func (s *StateStorage) GetState(ctx context.Context, graphID string) (interface{}, error) {
	key := getStateKey(graphID)

	// Get from Redis
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("state not found: %s", graphID)
		}
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	// Deserialize state
	var state domain.GraphState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// DeleteState deletes graph state from Redis
func (s *StateStorage) DeleteState(ctx context.Context, graphID string) error {
	key := getStateKey(graphID)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	s.logger.Debug("state deleted",
		zap.String("graph_id", graphID))

	return nil
}

// ListStates lists all graph states (for admin purposes)
func (s *StateStorage) ListStates(ctx context.Context) ([]*domain.GraphState, error) {
	pattern := "dago:state:*"

	// Scan for keys
	var cursor uint64
	var keys []string

	for {
		var batch []string
		var err error

		batch, cursor, err = s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan keys: %w", err)
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	// Get all states
	states := make([]*domain.GraphState, 0, len(keys))
	for _, key := range keys {
		data, err := s.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var state domain.GraphState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		states = append(states, &state)
	}

	return states, nil
}

// getStateKey returns the Redis key for a graph state
func getStateKey(graphID string) string {
	return fmt.Sprintf("dago:state:%s", graphID)
}
