package domain

import (
	"time"

	"github.com/google/uuid"
)

// User is the domain entity for an authenticated user.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string // write-only; never serialize to JSON
	Tags         []string
	OrgUnitID    *uuid.UUID
	OrgPath      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Credentials holds the plaintext login input; only used during authentication.
type Credentials struct {
	Email    string
	Password string
}
