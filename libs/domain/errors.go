package domain

import "errors"

// Domain sentinel errors for use-case-level error handling.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned on uniqueness violations or state conflicts.
	ErrConflict = errors.New("conflict")
	// ErrValidation is returned when input fails domain validation rules.
	ErrValidation = errors.New("validation error")
	// ErrInvalidGraphStatus is returned when an operation is not allowed in the current graph status.
	ErrInvalidGraphStatus = errors.New("invalid graph status for operation")
	// ErrInvalidCredentials is returned when authentication credentials are incorrect.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUnauthorized indicates invalid LLM provider credentials (HTTP 401).
	ErrUnauthorized = errors.New("unauthorized")

	// ErrRateLimited indicates the LLM provider rejected the request due to rate limiting (HTTP 429).
	// The caller can retry with exponential backoff.
	ErrRateLimited = errors.New("rate limited")

	// ErrProviderUnavailable indicates the LLM provider is temporarily unavailable (HTTP 500/529). Retryable.
	ErrProviderUnavailable = errors.New("provider unavailable")

	// ErrGraphValidation is returned when the graph definition fails structural validation
	// (e.g. unsupported edge type, unreachable node, missing entry_node).
	ErrGraphValidation = errors.New("domain: graph validation failed")

	// ErrRetryable signals that an event consumer must NACK so the broker re-delivers the message.
	ErrRetryable = errors.New("domain: retryable — consumer must NACK")
)
