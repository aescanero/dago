package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authadapter "github.com/aescanero/dago/adapters/auth"
)

func newHasher(t *testing.T) *authadapter.Argon2idHasher {
	t.Helper()
	return authadapter.NewArgon2idHasher()
}

func TestHashProducesValidPHCString(t *testing.T) {
	h := newHasher(t)
	hash, err := h.Hash("securepass123")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "$argon2id$v=19$"), "expected PHC prefix")
}

func TestVerifyCorrectPassword(t *testing.T) {
	h := newHasher(t)
	hash, err := h.Hash("securepass123")
	require.NoError(t, err)
	ok, err := h.Verify("securepass123", hash)
	require.NoError(t, err)
	assert.True(t, ok, "correct password should verify")
}

func TestVerifyWrongPassword(t *testing.T) {
	h := newHasher(t)
	hash, err := h.Hash("securepass123")
	require.NoError(t, err)
	ok, err := h.Verify("wrongpass", hash)
	require.NoError(t, err)
	assert.False(t, ok, "wrong password should not verify")
}

func TestHashesAreDifferentForSamePassword(t *testing.T) {
	h := newHasher(t)
	hash1, err := h.Hash("securepass123")
	require.NoError(t, err)
	hash2, err := h.Hash("securepass123")
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash2, "same password should produce different hashes (random salt)")
}

func TestVerifyTimingSafe(t *testing.T) {
	h := newHasher(t)
	hash, err := h.Hash("securepass123")
	require.NoError(t, err)

	const iterations = 3
	var correctTotal, wrongTotal time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, _ = h.Verify("securepass123", hash)
		correctTotal += time.Since(start)

		start = time.Now()
		_, _ = h.Verify("wrongpassword!", hash)
		wrongTotal += time.Since(start)
	}
	correctAvg := correctTotal / iterations
	wrongAvg := wrongTotal / iterations

	// Allow up to 50% deviation — argon2id always runs the full computation.
	ratio := float64(correctAvg) / float64(wrongAvg)
	assert.True(t, ratio > 0.5 && ratio < 2.0,
		"Verify() timing should be similar for correct and wrong passwords (ratio=%v)", ratio)
}
