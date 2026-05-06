package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/auth-server/internal/handler"
)

func init() { gin.SetMode(gin.TestMode) }

type fakeRegisterUC struct {
	executeFn func(ctx context.Context, creds domain.Credentials) (*domain.User, error)
}

func (f *fakeRegisterUC) Execute(ctx context.Context, creds domain.Credentials) (*domain.User, error) {
	return f.executeFn(ctx, creds)
}

type fakeLoginUC struct {
	executeFn func(ctx context.Context, creds domain.Credentials) (*domain.TokenPair, error)
}

func (f *fakeLoginUC) Execute(ctx context.Context, creds domain.Credentials) (*domain.TokenPair, error) {
	return f.executeFn(ctx, creds)
}

func setupEngine(t *testing.T, reg handler.RegisterUseCasePort, login handler.LoginUseCasePort) *gin.Engine {
	t.Helper()
	r := gin.New()
	h := handler.NewAuthHandler(reg, login, []byte(`{"keys":[]}`))
	v1 := r.Group("/api/v1/auth")
	v1.POST("/register", h.Register)
	v1.POST("/login", h.Login)
	r.GET("/.well-known/jwks.json", h.JWKS)
	return r
}

func newRequest(t *testing.T, method, path string, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, path, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestRegisterHandlerSuccess(t *testing.T) {
	reg := &fakeRegisterUC{
		executeFn: func(_ context.Context, creds domain.Credentials) (*domain.User, error) {
			return &domain.User{ID: uuid.New(), Email: creds.Email, Tags: []string{}}, nil
		},
	}
	r := setupEngine(t, reg, nil)
	body, _ := json.Marshal(map[string]string{"email": "test@example.com", "password": "securepass123"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest(t, http.MethodPost, "/api/v1/auth/register", body))
	assert.Equal(t, http.StatusCreated, w.Code)

	var out map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&out))
	assert.Contains(t, out, "id")
	assert.NotContains(t, out, "password_hash")
}

func TestRegisterHandlerInvalidEmail(t *testing.T) {
	reg := &fakeRegisterUC{
		executeFn: func(_ context.Context, _ domain.Credentials) (*domain.User, error) {
			return nil, domain.ErrValidation
		},
	}
	r := setupEngine(t, reg, nil)
	body, _ := json.Marshal(map[string]string{"email": "not-an-email", "password": "securepass123"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest(t, http.MethodPost, "/api/v1/auth/register", body))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestLoginHandlerSuccess(t *testing.T) {
	login := &fakeLoginUC{
		executeFn: func(_ context.Context, _ domain.Credentials) (*domain.TokenPair, error) {
			return &domain.TokenPair{AccessToken: "tok", TokenType: "Bearer", ExpiresIn: 3600}, nil
		},
	}
	r := setupEngine(t, nil, login)
	body, _ := json.Marshal(map[string]string{"email": "test@example.com", "password": "securepass123"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest(t, http.MethodPost, "/api/v1/auth/login", body))
	assert.Equal(t, http.StatusOK, w.Code)

	var out map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&out))
	assert.Equal(t, "tok", out["access_token"])
}

func TestLoginHandlerWrongCredentials(t *testing.T) {
	login := &fakeLoginUC{
		executeFn: func(_ context.Context, _ domain.Credentials) (*domain.TokenPair, error) {
			return nil, domain.ErrInvalidCredentials
		},
	}
	r := setupEngine(t, nil, login)
	body, _ := json.Marshal(map[string]string{"email": "test@example.com", "password": "wrong"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest(t, http.MethodPost, "/api/v1/auth/login", body))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var out map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&out))
	assert.Equal(t, "INVALID_CREDENTIALS", out["code"])
}
