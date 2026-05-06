package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

// LoginUser authenticates a user and issues a JWT token pair.
type LoginUser struct {
	repo   ports.UserRepository
	hasher ports.PasswordHasher
	issuer ports.TokenIssuer
}

// NewLoginUser creates a new LoginUser use case.
func NewLoginUser(repo ports.UserRepository, hasher ports.PasswordHasher, issuer ports.TokenIssuer) *LoginUser {
	return &LoginUser{repo: repo, hasher: hasher, issuer: issuer}
}

// Execute looks up the user, verifies the password, and returns a token pair.
// Returns ErrInvalidCredentials for both "user not found" and "wrong password"
// so that error responses do not reveal whether the email exists.
func (uc *LoginUser) Execute(ctx context.Context, creds domain.Credentials) (*domain.TokenPair, error) {
	user, err := uc.repo.FindByEmail(ctx, creds.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find user: %w", err)
	}
	ok, err := uc.hasher.Verify(creds.Password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, domain.ErrInvalidCredentials
	}
	token, err := uc.issuer.Issue(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}
	return &domain.TokenPair{AccessToken: token, TokenType: "Bearer", ExpiresIn: 3600}, nil
}
