package auth

import (
	"context"
	"fmt"

	"github.com/aescanero/dago/ent"
	entuser "github.com/aescanero/dago/ent/user"
	"github.com/aescanero/dago/libs/domain"
)

// EntUserRepository implements ports.UserRepository using Ent.
type EntUserRepository struct {
	client *ent.Client
}

// NewEntUserRepository creates a new EntUserRepository.
func NewEntUserRepository(client *ent.Client) *EntUserRepository {
	return &EntUserRepository{client: client}
}

// Create persists a new user and returns the created record.
func (r *EntUserRepository) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	created, err := r.client.User.Create().
		SetID(u.ID).
		SetEmail(u.Email).
		SetPasswordHash(u.PasswordHash).
		SetTags(u.Tags).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("%w: email already registered", domain.ErrConflict)
		}
		return nil, fmt.Errorf("storage.CreateUser: %w", err)
	}
	return entUserToDomain(created), nil
}

// FindByEmail looks up a user by email, returning domain.ErrNotFound if absent.
func (r *EntUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := r.client.User.Query().
		Where(entuser.EmailEQ(email)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("storage.FindUserByEmail: %w", err)
	}
	return entUserToDomain(u), nil
}

func entUserToDomain(u *ent.User) *domain.User {
	du := &domain.User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Tags:         u.Tags,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
	if du.Tags == nil {
		du.Tags = []string{}
	}
	return du
}
