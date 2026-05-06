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

	var aud []string
	switch a := mc["aud"].(type) {
	case []any:
		for _, v := range a {
			if s, ok := v.(string); ok {
				aud = append(aud, s)
			}
		}
	case string:
		aud = []string{a}
	}

	var attrs domain.ClaimsAttrs
	if a, ok := mc["attrs"].(map[string]any); ok {
		if tags, ok := a["tags"].([]any); ok {
			for _, t := range tags {
				if s, ok := t.(string); ok {
					attrs.Tags = append(attrs.Tags, s)
				}
			}
		}
		attrs.OrgUnit, _ = a["org_unit"].(string)
		attrs.OrgPath, _ = a["org_path"].(string)
	}
	if attrs.Tags == nil {
		attrs.Tags = []string{}
	}

	exp, _ := mc.GetExpirationTime()
	iat, _ := mc.GetIssuedAt()
	expTime := time.Time{}
	iatTime := time.Time{}
	if exp != nil {
		expTime = exp.Time
	}
	if iat != nil {
		iatTime = iat.Time
	}

	return &domain.Claims{
		Subject:    sub,
		Issuer:     iss,
		Audience:   aud,
		Scope:      scope,
		ClientType: clientType,
		Attrs:      attrs,
		ExpiresAt:  expTime,
		IssuedAt:   iatTime,
	}, nil
}
