//go:build contract

package contract_test

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

func newAuthContractServer(t *testing.T) string {
	t.Helper()
	srv := authutil.NewTestServer(t)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestRegisterContract(t *testing.T) {
	base := newAuthContractServer(t)
	body := mustJSON(map[string]any{"email": "reg@example.com", "password": "securepass123"})
	resp, err := http.Post(base+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	assert.Contains(t, out, "id")
	assert.Contains(t, out, "email")
	assert.Contains(t, out, "tags")
	assert.NotContains(t, out, "password_hash", "password_hash must never appear in UserResponse")
}

func TestRegisterDuplicateEmailContract(t *testing.T) {
	base := newAuthContractServer(t)
	regBody := mustJSON(map[string]any{"email": "dup@example.com", "password": "securepass123"})
	first, err := http.Post(base+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	require.NoError(t, err)
	first.Body.Close()

	second, err := http.Post(base+"/api/v1/auth/register", "application/json",
		bytes.NewReader(mustJSON(map[string]any{"email": "dup@example.com", "password": "securepass123"})))
	require.NoError(t, err)
	defer second.Body.Close()
	assert.Equal(t, http.StatusConflict, second.StatusCode)
}

func TestLoginSuccessContract(t *testing.T) {
	base := newAuthContractServer(t)
	regBody := mustJSON(map[string]any{"email": "login@example.com", "password": "securepass123"})
	regResp, err := http.Post(base+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	require.NoError(t, err)
	regResp.Body.Close()

	resp, err := http.Post(base+"/api/v1/auth/login", "application/json",
		bytes.NewReader(mustJSON(map[string]any{"email": "login@example.com", "password": "securepass123"})))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	assert.Contains(t, out, "access_token")
	assert.Equal(t, "Bearer", out["token_type"])
	assert.Equal(t, float64(3600), out["expires_in"])
}

func TestLoginWrongPasswordContract(t *testing.T) {
	base := newAuthContractServer(t)
	regBody := mustJSON(map[string]any{"email": "wp@example.com", "password": "securepass123"})
	regResp, err := http.Post(base+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	require.NoError(t, err)
	regResp.Body.Close()

	resp, err := http.Post(base+"/api/v1/auth/login", "application/json",
		bytes.NewReader(mustJSON(map[string]any{"email": "wp@example.com", "password": "wrongpass12345"})))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	assert.Equal(t, "INVALID_CREDENTIALS", out["code"])
}

func TestJWKSEndpointContract(t *testing.T) {
	base := newAuthContractServer(t)
	resp, err := http.Get(base + "/.well-known/jwks.json")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	keys, ok := out["keys"].([]any)
	require.True(t, ok, "expected 'keys' array in JWKS response")
	require.NotEmpty(t, keys)

	key, ok := keys[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "RSA", key["kty"])
	assert.Equal(t, "RS256", key["alg"])
}

func TestLoginReturnsJWTWithCorrectClaims(t *testing.T) {
	base := newAuthContractServer(t)
	regBody := mustJSON(map[string]any{"email": "claims@example.com", "password": "securepass123"})
	regResp, err := http.Post(base+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	require.NoError(t, err)
	regResp.Body.Close()

	resp, err := http.Post(base+"/api/v1/auth/login", "application/json",
		bytes.NewReader(mustJSON(map[string]any{"email": "claims@example.com", "password": "securepass123"})))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	token, ok := out["access_token"].(string)
	require.True(t, ok, "access_token should be a string")

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "JWT should have 3 parts")

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	var header map[string]any
	require.NoError(t, json.Unmarshal(headerJSON, &header))
	assert.Equal(t, "RS256", header["alg"])

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(payloadJSON, &payload))
	assert.Contains(t, payload, "sub")
	assert.Contains(t, payload, "iss")
	assert.Contains(t, payload, "aud")
	assert.Contains(t, payload, "scope")
	assert.Contains(t, payload, "attrs")
}
