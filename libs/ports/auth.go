package ports

import (
	"context"

	"github.com/aescanero/dago/libs/domain"
)

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) (bool, error)
}

// TokenIssuer produces signed JWT tokens for a given user.
type TokenIssuer interface {
	Issue(ctx context.Context, user *domain.User) (string, error)
}

// TokenValidator parses and verifies JWT tokens, returning the embedded claims.
type TokenValidator interface {
	Validate(ctx context.Context, token string) (*domain.Claims, error)
}

// UserRepository manages user persistence.
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}
