package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AuthorizationCodeStore manages one-time PKCE authorization codes.
type AuthorizationCodeStore interface {
	Store(ctx context.Context, code AuthorizationCode) error
	Consume(ctx context.Context, codeHash string) (*AuthorizationCode, error)
}

// AuthorizationCode holds the PKCE state linked to a pending token exchange.
type AuthorizationCode struct {
	CodeHash      string    // SHA-256(code_plaintext) — never store plaintext
	UserID        uuid.UUID // optional: may be zero if Email is set
	Email         string    // authenticated user's email for token issuance in PostToken
	CodeChallenge string    // base64url(SHA-256(code_verifier)) from the client
	RedirectURI   string
	ClientID      string
	ExpiresAt     time.Time
}

// HashCode computes the hex-encoded SHA-256 of a plaintext authorization code.
func HashCode(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// InMemoryCodeStore is a non-distributed implementation for the bootstrap sprint.
// Use sync.RWMutex (not sync.Map) to allow iteration under lock during cleanup.
// TODO(SPRINT-ABAC): replace with Valkey implementation for distributed deployments.
type InMemoryCodeStore struct {
	mu      sync.RWMutex
	entries map[string]AuthorizationCode
}

// NewInMemoryCodeStore creates a store and starts a background cleanup goroutine.
func NewInMemoryCodeStore(cleanupInterval time.Duration) *InMemoryCodeStore {
	s := &InMemoryCodeStore{
		entries: make(map[string]AuthorizationCode),
	}
	go s.runCleanup(cleanupInterval)
	return s
}

// Store saves a code (keyed by CodeHash) with its associated PKCE metadata.
func (s *InMemoryCodeStore) Store(_ context.Context, code AuthorizationCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[code.CodeHash] = code
	return nil
}

// Consume finds, removes, and returns the code for the given hash.
// Returns ErrCodeNotFound if the code is missing or expired.
func (s *InMemoryCodeStore) Consume(_ context.Context, codeHash string) (*AuthorizationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[codeHash]
	if !ok || time.Now().After(entry.ExpiresAt) {
		delete(s.entries, codeHash)
		return nil, ErrCodeNotFound
	}
	delete(s.entries, codeHash)
	return &entry, nil
}

func (s *InMemoryCodeStore) runCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for hash, entry := range s.entries {
			if now.After(entry.ExpiresAt) {
				delete(s.entries, hash)
			}
		}
		s.mu.Unlock()
	}
}
