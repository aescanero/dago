package oauth

import "errors"

// ErrCodeNotFound is returned when an authorization code is missing or expired.
var ErrCodeNotFound = errors.New("authorization code not found or expired")

// ErrInvalidCodeVerifier is returned when the PKCE verifier doesn't match the challenge.
var ErrInvalidCodeVerifier = errors.New("code verifier does not match challenge")
