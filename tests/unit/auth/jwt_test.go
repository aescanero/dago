package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authadapter "github.com/aescanero/dago/adapters/auth"
	"github.com/aescanero/dago/libs/domain"
)

func newRSAKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key, &key.PublicKey
}

func newTestUser() *domain.User {
	return &domain.User{
		ID:    uuid.New(),
		Email: "jwt-test@example.com",
		Tags:  []string{"admin", "user"},
	}
}

func TestIssueProducesValidJWT(t *testing.T) {
	ctx := context.Background()
	priv, pub := newRSAKeyPair(t)
	issuer := authadapter.NewJWTIssuer(priv, "dago-test", "dago-api", 3600)
	validator := authadapter.NewRSAValidator(pub, "dago-test", "dago-api")

	user := newTestUser()
	token, err := issuer.Issue(ctx, user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := validator.Validate(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), claims.Subject)
	assert.Equal(t, "dago-test", claims.Issuer)
	assert.Contains(t, claims.Audience, "dago-api")
	assert.Equal(t, domain.DefaultUserScopes, claims.Scope)
}

func TestIssueIncludesUserTags(t *testing.T) {
	ctx := context.Background()
	priv, pub := newRSAKeyPair(t)
	issuer := authadapter.NewJWTIssuer(priv, "dago-test", "dago-api", 3600)
	validator := authadapter.NewRSAValidator(pub, "dago-test", "dago-api")

	user := newTestUser()
	token, err := issuer.Issue(ctx, user)
	require.NoError(t, err)

	claims, err := validator.Validate(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, user.Tags, claims.Attrs.Tags)
}

func TestValidateAcceptsValidJWT(t *testing.T) {
	ctx := context.Background()
	priv, pub := newRSAKeyPair(t)
	issuer := authadapter.NewJWTIssuer(priv, "dago-test", "dago-api", 3600)
	validator := authadapter.NewRSAValidator(pub, "dago-test", "dago-api")

	token, err := issuer.Issue(ctx, newTestUser())
	require.NoError(t, err)

	claims, err := validator.Validate(ctx, token)
	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.WithinDuration(t, time.Now().Add(time.Hour), claims.ExpiresAt, 10*time.Second)
}

func TestValidateRejectsExpiredJWT(t *testing.T) {
	ctx := context.Background()
	priv, pub := newRSAKeyPair(t)
	issuer := authadapter.NewJWTIssuer(priv, "dago-test", "dago-api", -1) // expired
	validator := authadapter.NewRSAValidator(pub, "dago-test", "dago-api")

	token, err := issuer.Issue(ctx, newTestUser())
	require.NoError(t, err)

	_, err = validator.Validate(ctx, token)
	assert.Error(t, err, "expired token should be rejected")
}

func TestValidateRejectsWrongSignature(t *testing.T) {
	ctx := context.Background()
	priv, _ := newRSAKeyPair(t)
	_, otherPub := newRSAKeyPair(t)

	issuer := authadapter.NewJWTIssuer(priv, "dago-test", "dago-api", 3600)
	validator := authadapter.NewRSAValidator(otherPub, "dago-test", "dago-api")

	token, err := issuer.Issue(ctx, newTestUser())
	require.NoError(t, err)

	_, err = validator.Validate(ctx, token)
	assert.Error(t, err, "wrong signature should be rejected")
}

func TestValidateRejectsWrongAudience(t *testing.T) {
	ctx := context.Background()
	priv, pub := newRSAKeyPair(t)
	issuer := authadapter.NewJWTIssuer(priv, "dago-test", "dago-api", 3600)
	validator := authadapter.NewRSAValidator(pub, "dago-test", "wrong-audience")

	token, err := issuer.Issue(ctx, newTestUser())
	require.NoError(t, err)

	_, err = validator.Validate(ctx, token)
	assert.Error(t, err, "wrong audience should be rejected")
}
