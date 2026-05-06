package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/aescanero/dago/libs/domain"
)

// JWTIssuer issues RS256-signed JWT tokens.
type JWTIssuer struct {
	privateKey *rsa.PrivateKey
	issuer     string
	audience   string
	ttl        int // seconds
}

// NewJWTIssuer creates a JWTIssuer with the given RSA private key and config.
func NewJWTIssuer(key *rsa.PrivateKey, issuer, audience string, ttl int) *JWTIssuer {
	return &JWTIssuer{privateKey: key, issuer: issuer, audience: audience, ttl: ttl}
}

// Issue signs a JWT with RS256 and embeds the user's ABAC claims.
func (i *JWTIssuer) Issue(_ context.Context, user *domain.User) (string, error) {
	now := time.Now()
	tags := user.Tags
	if tags == nil {
		tags = []string{}
	}
	orgUnit := ""
	if user.OrgUnitID != nil {
		orgUnit = user.OrgUnitID.String()
	}
	claims := jwt.MapClaims{
		"iss":         i.issuer,
		"sub":         user.ID.String(),
		"aud":         []string{i.audience},
		"exp":         now.Add(time.Duration(i.ttl) * time.Second).Unix(),
		"iat":         now.Unix(),
		"scope":       domain.DefaultUserScopes,
		"client_type": "user",
		"attrs": map[string]any{
			"tags":     tags,
			"org_unit": orgUnit,
			"org_path": user.OrgPath,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(i.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
