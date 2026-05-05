package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrValidation         = errors.New("validation error")
	ErrInvalidGraphStatus = errors.New("invalid graph status for operation")
)
