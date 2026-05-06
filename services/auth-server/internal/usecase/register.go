package usecase

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// RegisterUser creates a new user account with a hashed password.
type RegisterUser struct {
	repo   ports.UserRepository
	hasher ports.PasswordHasher
}

// NewRegisterUser creates a new RegisterUser use case.
func NewRegisterUser(repo ports.UserRepository, hasher ports.PasswordHasher) *RegisterUser {
	return &RegisterUser{repo: repo, hasher: hasher}
}

// Execute validates credentials, hashes the password, and persists the new user.
func (uc *RegisterUser) Execute(ctx context.Context, creds domain.Credentials) (*domain.User, error) {
	if err := validateCredentials(creds); err != nil {
		return nil, err
	}
	hash, err := uc.hasher.Hash(creds.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u := &domain.User{
		ID:           uuid.New(),
		Email:        creds.Email,
		PasswordHash: hash,
		Tags:         []string{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	created, err := uc.repo.Create(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}

func validateCredentials(creds domain.Credentials) error {
	if !emailPattern.MatchString(creds.Email) {
		return fmt.Errorf("%w: invalid email format", domain.ErrValidation)
	}
	if len(creds.Password) < 12 {
		return fmt.Errorf("%w: password must be at least 12 characters", domain.ErrValidation)
	}
	return nil
}
