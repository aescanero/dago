package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/services/auth-server/internal/oauth"
)

// OAuthLoginPort authenticates a user and returns a token pair.
type OAuthLoginPort interface {
	Execute(ctx context.Context, creds domain.Credentials) (*domain.TokenPair, error)
}

// OAuthUserPort looks up a user by email for token issuance.
type OAuthUserPort interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}

// OAuthTokenPort issues a raw JWT string for a given user.
type OAuthTokenPort interface {
	Issue(ctx context.Context, user *domain.User) (string, error)
}

// OAuthHandler handles the PKCE authorization and token endpoints.
type OAuthHandler struct {
	codeStore oauth.AuthorizationCodeStore
	login     OAuthLoginPort
	users     OAuthUserPort
	issuer    OAuthTokenPort
}

// NewOAuthHandler creates an OAuthHandler with its dependencies.
func NewOAuthHandler(
	cs oauth.AuthorizationCodeStore,
	login OAuthLoginPort,
	users OAuthUserPort,
	issuer OAuthTokenPort,
) *OAuthHandler {
	return &OAuthHandler{codeStore: cs, login: login, users: users, issuer: issuer}
}

var loginFormTmpl = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>Sign in — Dago</title>
<style>body{font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f5f5f5}
form{background:#fff;padding:2rem;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,.1);width:320px}
h2{margin:0 0 1.5rem;font-size:1.25rem}
label{display:block;margin-bottom:.25rem;font-size:.875rem;color:#555}
input{width:100%;box-sizing:border-box;padding:.5rem;border:1px solid #ddd;border-radius:4px;margin-bottom:1rem;font-size:1rem}
button{width:100%;padding:.75rem;background:#2563eb;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer}
.error{color:#dc2626;font-size:.875rem;margin-bottom:1rem}</style>
</head>
<body><form method="POST" action="/authorize">
<h2>Sign in to Dago</h2>
{{if .Error}}<p class="error">{{.Error}}</p>{{end}}
<input type="hidden" name="response_type" value="{{.ResponseType}}">
<input type="hidden" name="client_id" value="{{.ClientID}}">
<input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
<input type="hidden" name="code_challenge" value="{{.CodeChallenge}}">
<input type="hidden" name="code_challenge_method" value="{{.CodeChallengeMethod}}">
<input type="hidden" name="state" value="{{.State}}">
<label for="email">Email</label>
<input id="email" type="email" name="email" required autofocus>
<label for="password">Password</label>
<input id="password" type="password" name="password" required>
<button type="submit">Sign in</button>
</form></body></html>`))

type loginFormData struct {
	Error               string
	ResponseType        string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	State               string
}

// GetAuthorize renders an HTML login form after validating PKCE parameters.
func (h *OAuthHandler) GetAuthorize(c *gin.Context) {
	params, ok := parseAuthorizeQueryParams(c)
	if !ok {
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	_ = loginFormTmpl.Execute(c.Writer, loginFormData{
		ResponseType:        params["response_type"],
		ClientID:            params["client_id"],
		RedirectURI:         params["redirect_uri"],
		CodeChallenge:       params["code_challenge"],
		CodeChallengeMethod: params["code_challenge_method"],
		State:               params["state"],
	})
}

// PostAuthorize processes the login form, generates a PKCE auth code, and redirects.
func (h *OAuthHandler) PostAuthorize(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, errResp("BAD_REQUEST", "malformed form"))
		return
	}
	params := map[string]string{
		"response_type":         c.Request.FormValue("response_type"),
		"client_id":             c.Request.FormValue("client_id"),
		"redirect_uri":          c.Request.FormValue("redirect_uri"),
		"code_challenge":        c.Request.FormValue("code_challenge"),
		"code_challenge_method": c.Request.FormValue("code_challenge_method"),
		"state":                 c.Request.FormValue("state"),
	}
	if !validateAuthorizeParams(c, params) {
		return
	}

	email := c.Request.FormValue("email")
	password := c.Request.FormValue("password")
	if _, err := h.login.Execute(c.Request.Context(), domain.Credentials{Email: email, Password: password}); err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(http.StatusOK)
		_ = loginFormTmpl.Execute(c.Writer, loginFormData{
			Error:               "Invalid email or password.",
			ResponseType:        params["response_type"],
			ClientID:            params["client_id"],
			RedirectURI:         params["redirect_uri"],
			CodeChallenge:       params["code_challenge"],
			CodeChallengeMethod: params["code_challenge_method"],
			State:               params["state"],
		})
		return
	}

	codeBytes := make([]byte, 32)
	if _, err := rand.Read(codeBytes); err != nil {
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", "internal server error"))
		return
	}
	codePlaintext := base64.RawURLEncoding.EncodeToString(codeBytes)

	authCode := oauth.AuthorizationCode{
		CodeHash:      oauth.HashCode(codePlaintext),
		Email:         email,
		CodeChallenge: params["code_challenge"],
		RedirectURI:   params["redirect_uri"],
		ClientID:      params["client_id"],
		ExpiresAt:     time.Now().Add(60 * time.Second),
	}
	if err := h.codeStore.Store(c.Request.Context(), authCode); err != nil {
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", "internal server error"))
		return
	}

	redirectURL := params["redirect_uri"] + "?code=" + codePlaintext
	if s := params["state"]; s != "" {
		redirectURL += "&state=" + s
	}
	c.Redirect(http.StatusFound, redirectURL)
}

type tokenRequest struct {
	GrantType    string `json:"grant_type"    binding:"required"`
	Code         string `json:"code"          binding:"required"`
	CodeVerifier string `json:"code_verifier" binding:"required"`
	RedirectURI  string `json:"redirect_uri"  binding:"required"`
	ClientID     string `json:"client_id"     binding:"required"`
}

// PostToken exchanges a PKCE authorization code for a JWT access token.
func (h *OAuthHandler) PostToken(c *gin.Context) {
	var req tokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("BAD_REQUEST", err.Error()))
		return
	}
	if req.GrantType != "authorization_code" {
		c.JSON(http.StatusBadRequest, errResp("UNSUPPORTED_GRANT_TYPE", "only authorization_code is supported"))
		return
	}

	entry, err := h.codeStore.Consume(c.Request.Context(), oauth.HashCode(req.Code))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("INVALID_CODE", "authorization code not found or expired"))
		return
	}

	// Verify PKCE S256: base64url(SHA-256(verifier)) must equal stored challenge (RFC 7636 §4.6).
	if computeS256Challenge(req.CodeVerifier) != entry.CodeChallenge {
		c.JSON(http.StatusBadRequest, errResp("INVALID_CODE_VERIFIER", "code verifier does not match challenge"))
		return
	}
	if entry.RedirectURI != req.RedirectURI || entry.ClientID != req.ClientID {
		c.JSON(http.StatusBadRequest, errResp("INVALID_REQUEST", "redirect_uri or client_id mismatch"))
		return
	}

	user, err := h.users.FindByEmail(c.Request.Context(), entry.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", "internal server error"))
		return
	}
	accessToken, err := h.issuer.Issue(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", "internal server error"))
		return
	}
	c.JSON(http.StatusOK, tokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	})
}

// computeS256Challenge returns base64url(SHA-256(verifier)) per RFC 7636.
func computeS256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func parseAuthorizeQueryParams(c *gin.Context) (map[string]string, bool) {
	params := map[string]string{
		"response_type":         c.Query("response_type"),
		"client_id":             c.Query("client_id"),
		"redirect_uri":          c.Query("redirect_uri"),
		"code_challenge":        c.Query("code_challenge"),
		"code_challenge_method": c.Query("code_challenge_method"),
		"state":                 c.Query("state"),
	}
	return params, validateAuthorizeParams(c, params)
}

func validateAuthorizeParams(c *gin.Context, params map[string]string) bool {
	if params["response_type"] != "code" {
		c.JSON(http.StatusBadRequest, errResp("INVALID_REQUEST", "response_type must be code"))
		return false
	}
	if params["client_id"] == "" {
		c.JSON(http.StatusBadRequest, errResp("INVALID_REQUEST", "client_id is required"))
		return false
	}
	if params["redirect_uri"] == "" {
		c.JSON(http.StatusBadRequest, errResp("INVALID_REQUEST", "redirect_uri is required"))
		return false
	}
	// TODO(SPRINT-ABAC): validate redirect_uri against registered clients
	if !strings.HasPrefix(params["redirect_uri"], "http://localhost") {
		c.JSON(http.StatusBadRequest, errResp("INVALID_REDIRECT_URI", "redirect_uri must start with http://localhost"))
		return false
	}
	if params["code_challenge"] == "" {
		c.JSON(http.StatusBadRequest, errResp("INVALID_REQUEST", "code_challenge is required"))
		return false
	}
	if params["code_challenge_method"] != "S256" {
		c.JSON(http.StatusBadRequest, errResp("INVALID_REQUEST", "code_challenge_method must be S256"))
		return false
	}
	return true
}
