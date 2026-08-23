package interfaces

import "errors"

// Domain-level errors returned by repositories and application services.
// The transport layer maps them to HTTP status codes.
var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
	ErrInvalid   = errors.New("invalid input")
)
