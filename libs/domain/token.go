package domain

import "time"

// Scope constants for JWT claims.
const (
	ScopeGraphsRead    = "graphs:read"
	ScopeGraphsExecute = "graphs:execute"
	ScopeGraphsManage  = "graphs:manage"
	DefaultUserScopes  = ScopeGraphsRead + " " + ScopeGraphsExecute + " " + ScopeGraphsManage
)

// Claims holds the verified payload extracted from a JWT.
type Claims struct {
	Subject    string
	Issuer     string
	Audience   []string
	Scope      string
	ClientType string
	Attrs      ClaimsAttrs
	ExpiresAt  time.Time
	IssuedAt   time.Time
}

// ClaimsAttrs contains ABAC-relevant attributes embedded in the JWT.
type ClaimsAttrs struct {
	Tags    []string
	OrgUnit string
	OrgPath string
}

// TokenPair is the response payload returned to the caller after authentication.
type TokenPair struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int
}
