package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/orchestrator/internal/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

type fakeValidator struct {
	validateFn func(ctx context.Context, token string) (*domain.Claims, error)
}

func (f *fakeValidator) Validate(ctx context.Context, token string) (*domain.Claims, error) {
	return f.validateFn(ctx, token)
}

func newGetRequest(t *testing.T, path string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	require.NoError(t, err)
	return req
}

func setupMWEngine(t *testing.T, required bool, v middleware.TokenValidatorPort) *gin.Engine {
	t.Helper()
	r := gin.New()
	mw := middleware.NewAuthMiddleware(required, v)
	r.GET("/test", mw, func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestAuthMiddlewareBypassMode(t *testing.T) {
	r := setupMWEngine(t, false, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newGetRequest(t, "/test"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	claims := &domain.Claims{Subject: "user-123", ExpiresAt: time.Now().Add(time.Hour)}
	v := &fakeValidator{
		validateFn: func(_ context.Context, _ string) (*domain.Claims, error) { return claims, nil },
	}
	r := gin.New()
	mw := middleware.NewAuthMiddleware(true, v)
	r.GET("/test", mw, func(c *gin.Context) {
		got, ok := middleware.ClaimsFromContext(c)
		require.True(t, ok)
		assert.Equal(t, "user-123", got.Subject)
		c.Status(http.StatusOK)
	})
	req := newGetRequest(t, "/test")
	req.Header.Set("Authorization", "Bearer valid.jwt.token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddlewareMissingToken(t *testing.T) {
	v := &fakeValidator{
		validateFn: func(_ context.Context, _ string) (*domain.Claims, error) { return nil, nil },
	}
	r := setupMWEngine(t, true, v)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newGetRequest(t, "/test"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddlewareExpiredToken(t *testing.T) {
	v := &fakeValidator{
		validateFn: func(_ context.Context, _ string) (*domain.Claims, error) {
			return nil, domain.ErrInvalidCredentials
		},
	}
	r := setupMWEngine(t, true, v)
	req := newGetRequest(t, "/test")
	req.Header.Set("Authorization", "Bearer expired.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
