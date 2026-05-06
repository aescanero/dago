package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
)

// MustGenerateRSAKeyPair generates a 2048-bit RSA keypair or panics.
// Used only in development/startup when no key files are configured.
func MustGenerateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Errorf("generate RSA key: %w", err))
	}
	return key, &key.PublicKey
}
