package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/aescanero/dago/libs/domain"
)

// RSAValidator validates RS256 JWTs using a known public key (for tests/dev).
type RSAValidator struct {
	publicKey *rsa.PublicKey
	issuer    string
	audience  string
}

// NewRSAValidator creates a validator that uses a static RSA public key.
func NewRSAValidator(pub *rsa.PublicKey, issuer, audience string) *RSAValidator {
	return &RSAValidator{publicKey: pub, issuer: issuer, audience: audience}
}

// Validate parses and verifies a JWT, returning the embedded claims.
func (v *RSAValidator) Validate(_ context.Context, tokenStr string) (*domain.Claims, error) {
	token, err := jwt.Parse(tokenStr,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return v.publicKey, nil
		},
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}
	mc, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return mapToClaims(mc)
}

func mapToClaims(mc jwt.MapClaims) (*domain.Claims, error) {
	sub, _ := mc["sub"].(string)
	iss, _ := mc["iss"].(string)
	scope, _ := mc["scope"].(string)
	clientType, _ := mc["client_type"].(string)
	return &domain.Claims{
		Subject:    sub,
		Issuer:     iss,
		Audience:   extractAudience(mc),
		Scope:      scope,
		ClientType: clientType,
		Attrs:      extractAttrs(mc),
		ExpiresAt:  extractTime(mc, "exp"),
		IssuedAt:   extractTime(mc, "iat"),
	}, nil
}

func extractAudience(mc jwt.MapClaims) []string {
	switch a := mc["aud"].(type) {
	case []any:
		out := make([]string, 0, len(a))
		for _, v := range a {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{a}
	default:
		return nil
	}
}

func extractAttrs(mc jwt.MapClaims) domain.ClaimsAttrs {
	a, ok := mc["attrs"].(map[string]any)
	if !ok {
		return domain.ClaimsAttrs{Tags: []string{}}
	}
	attrs := domain.ClaimsAttrs{
		Tags: extractStringSlice(a["tags"]),
	}
	attrs.OrgUnit, _ = a["org_unit"].(string)
	attrs.OrgPath, _ = a["org_path"].(string)
	return attrs
}

func extractStringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func extractTime(mc jwt.MapClaims, key string) time.Time {
	switch key {
	case "exp":
		t, _ := mc.GetExpirationTime()
		if t != nil {
			return t.Time
		}
	case "iat":
		t, _ := mc.GetIssuedAt()
		if t != nil {
			return t.Time
		}
	}
	return time.Time{}
}
