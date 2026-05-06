//go:build integration

package integration_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authutil "github.com/aescanero/dago/services/auth-server/testutil"
)

// TestAuthFlowIntegration verifies the complete register → login → validate flow.
// This test uses a real test server wired with in-memory fakes (no Docker required);
// the "integration" build tag groups it with tests that may need external services.
func TestAuthFlowIntegration(t *testing.T) {
	srv := authutil.NewTestServer(t)
	t.Cleanup(srv.Close)
	base := srv.URL

	// Step 1: Register user.
	regBody, _ := json.Marshal(map[string]string{
		"email":    "integ@example.com",
		"password": "securepass123",
	})
	regResp, err := http.Post(base+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	require.NoError(t, err)
	defer regResp.Body.Close()
	require.Equal(t, http.StatusCreated, regResp.StatusCode, "register should return 201")

	var regOut map[string]any
	require.NoError(t, json.NewDecoder(regResp.Body).Decode(&regOut))
	userID, ok := regOut["id"].(string)
	require.True(t, ok, "user id should be a string")
	assert.NotEmpty(t, userID)
	assert.NotContains(t, regOut, "password_hash", "password_hash must never appear in response")

	// Step 2: Login → obtain JWT.
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "integ@example.com",
		"password": "securepass123",
	})
	loginResp, err := http.Post(base+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	require.NoError(t, err)
	defer loginResp.Body.Close()
	require.Equal(t, http.StatusOK, loginResp.StatusCode, "login should return 200")

	var loginOut map[string]any
	require.NoError(t, json.NewDecoder(loginResp.Body).Decode(&loginOut))
	token, ok := loginOut["access_token"].(string)
	require.True(t, ok, "access_token should be a string")
	assert.Equal(t, "Bearer", loginOut["token_type"])
	assert.Equal(t, float64(3600), loginOut["expires_in"])

	// Step 3: Parse JWT and verify claims.
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "JWT should have 3 parts")

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	var header map[string]any
	require.NoError(t, json.Unmarshal(headerJSON, &header))
	assert.Equal(t, "RS256", header["alg"], "JWT algorithm should be RS256")

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(payloadJSON, &payload))
	assert.Equal(t, userID, payload["sub"], "sub should equal registered user ID")
	assert.Contains(t, payload, "scope", "JWT should include scope claim")
	attrs, ok := payload["attrs"].(map[string]any)
	require.True(t, ok, "JWT should include attrs claim")
	tags, ok := attrs["tags"].([]any)
	assert.True(t, ok, "attrs.tags should be an array")
	assert.Empty(t, tags, "new user should have empty tags")

	// Step 4: Validate JWT against JWKS endpoint.
	jwksResp, err := http.Get(base + "/.well-known/jwks.json")
	require.NoError(t, err)
	defer jwksResp.Body.Close()
	assert.Equal(t, http.StatusOK, jwksResp.StatusCode)
	var jwksOut map[string]any
	require.NoError(t, json.NewDecoder(jwksResp.Body).Decode(&jwksOut))
	keys, ok := jwksOut["keys"].([]any)
	require.True(t, ok)
	assert.NotEmpty(t, keys, "JWKS should contain at least one key")

	// Step 5: Login with wrong password → 401.
	badBody, _ := json.Marshal(map[string]string{
		"email":    "integ@example.com",
		"password": "wrongpassword!!",
	})
	badResp, err := http.Post(base+"/api/v1/auth/login", "application/json", bytes.NewReader(badBody))
	require.NoError(t, err)
	defer badResp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, badResp.StatusCode, "wrong password should return 401")
	var badOut map[string]any
	require.NoError(t, json.NewDecoder(badResp.Body).Decode(&badOut))
	assert.Equal(t, "INVALID_CREDENTIALS", badOut["code"])
}
