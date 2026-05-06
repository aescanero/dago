package auth

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

type jwk struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

// PublicKeyToJWKSJSON serialises an RSA public key as a JWKS JSON document.
func PublicKeyToJWKSJSON(pub *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	digest := sha256.Sum256(der)
	kid := base64.RawURLEncoding.EncodeToString(digest[:8])

	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eBig := big.NewInt(int64(pub.E))
	e := base64.RawURLEncoding.EncodeToString(eBig.Bytes())

	doc := jwks{Keys: []jwk{{Kty: "RSA", Alg: "RS256", Use: "sig", Kid: kid, N: n, E: e}}}
	return json.Marshal(doc)
}
