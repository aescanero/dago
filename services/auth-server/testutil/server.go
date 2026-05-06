package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http/httptest"
	"testing"

	authadapter "github.com/aescanero/dago/adapters/auth"
	"github.com/aescanero/dago/services/auth-server/internal/handler"
	"github.com/aescanero/dago/services/auth-server/internal/router"
	"github.com/aescanero/dago/services/auth-server/internal/usecase"
	"github.com/aescanero/dago/tests/testutil/fakes"
)

// NewTestServer creates an httptest.Server wired with in-memory fakes and a generated RSA keypair.
func NewTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	priv, pub := mustGenRSA(t)
	jwksJSON, err := authadapter.PublicKeyToJWKSJSON(pub)
	if err != nil {
		t.Fatalf("generate JWKS: %v", err)
	}
	repo := fakes.NewInMemoryUserRepository()
	hasher := authadapter.NewArgon2idHasher()
	issuer := authadapter.NewJWTIssuer(priv, "dago-test", "dago-api", 3600)
	registerUC := usecase.NewRegisterUser(repo, hasher)
	loginUC := usecase.NewLoginUser(repo, hasher, issuer)
	authH := handler.NewAuthHandler(registerUC, loginUC, jwksJSON)
	r := router.NewRouter(authH)
	return httptest.NewServer(r)
}

func mustGenRSA(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return key, &key.PublicKey
}
