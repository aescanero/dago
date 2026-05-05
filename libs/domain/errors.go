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
)
