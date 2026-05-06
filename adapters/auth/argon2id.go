package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Memory      = 64 * 1024
	argon2Iterations  = 3
	argon2Parallelism = 4
	argon2SaltLen     = 16
	argon2KeyLen      = 32
)

// Argon2idHasher implements ports.PasswordHasher using argon2id (OWASP 2023 parameters).
type Argon2idHasher struct{}

// NewArgon2idHasher creates a new Argon2idHasher.
func NewArgon2idHasher() *Argon2idHasher { return &Argon2idHasher{} }

// Hash returns a PHC-format argon2id hash for the given password.
func (h *Argon2idHasher) Hash(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("argon2id: generate salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLen)
	return buildPHC(salt, key), nil
}

// Verify checks a plaintext password against an argon2id PHC hash in constant time.
func (h *Argon2idHasher) Verify(password, hash string) (bool, error) {
	salt, key, err := parsePHC(hash)
	if err != nil {
		return false, fmt.Errorf("argon2id: parse hash: %w", err)
	}
	candidate := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLen)
	return subtle.ConstantTimeCompare(key, candidate) == 1, nil
}

func buildPHC(salt, key []byte) string {
	s := base64.RawStdEncoding.EncodeToString(salt)
	k := base64.RawStdEncoding.EncodeToString(key)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Iterations, argon2Parallelism, s, k)
}

func parsePHC(hash string) (salt, key []byte, err error) {
	parts := strings.Split(hash, "$")
	// $argon2id$v=19$m=...,t=...,p=...$<salt>$<key>
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, fmt.Errorf("invalid PHC format")
	}
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, fmt.Errorf("decode salt: %w", err)
	}
	key, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, fmt.Errorf("decode key: %w", err)
	}
	return salt, key, nil
}
