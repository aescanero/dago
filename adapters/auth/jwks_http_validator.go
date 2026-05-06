package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/aescanero/dago/libs/domain"
)

// JWKSHTTPValidator validates RS256 JWTs by fetching the public key from a JWKS HTTP endpoint.
type JWKSHTTPValidator struct {
	jwksURL  string
	issuer   string
	audience string

	mu        sync.RWMutex
	cachedKey *rsa.PublicKey
	fetchedAt time.Time
	cacheTTL  time.Duration
}

// NewJWKSHTTPValidator creates a validator that fetches the public key from a JWKS URL.
func NewJWKSHTTPValidator(jwksURL, issuer, audience string) *JWKSHTTPValidator {
	return &JWKSHTTPValidator{
		jwksURL:  jwksURL,
		issuer:   issuer,
		audience: audience,
		cacheTTL: 5 * time.Minute,
	}
}

// Validate parses and verifies a JWT using the public key from the JWKS endpoint.
func (v *JWKSHTTPValidator) Validate(ctx context.Context, tokenStr string) (*domain.Claims, error) {
	pub, err := v.getPublicKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("get jwks key: %w", err)
	}
	inner := &RSAValidator{publicKey: pub, issuer: v.issuer, audience: v.audience}
	return inner.Validate(ctx, tokenStr)
}

func (v *JWKSHTTPValidator) getPublicKey(ctx context.Context) (*rsa.PublicKey, error) {
	v.mu.RLock()
	if v.cachedKey != nil && time.Since(v.fetchedAt) < v.cacheTTL {
		key := v.cachedKey
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()

	v.mu.Lock()
	defer v.mu.Unlock()
	if v.cachedKey != nil && time.Since(v.fetchedAt) < v.cacheTTL {
		return v.cachedKey, nil
	}
	key, err := fetchFirstRSAKey(ctx, v.jwksURL)
	if err != nil {
		return nil, err
	}
	v.cachedKey = key
	v.fetchedAt = time.Now()
	return key, nil
}

func fetchFirstRSAKey(ctx context.Context, url string) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		return parseRSAKey(k.N, k.E)
	}
	return nil, fmt.Errorf("no RSA key found in JWKS")
}

func parseRSAKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())
	_ = jwt.SigningMethodRS256 // ensure jwt is used
	return &rsa.PublicKey{N: n, E: e}, nil
}
