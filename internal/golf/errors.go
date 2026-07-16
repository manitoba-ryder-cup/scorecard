package golf

import "errors"

// Sentinel errors let the API layer translate domain failures to HTTP status codes
// without knowing the domain's internals: invalid input -> 400, conflict -> 409.
var (
	// ErrInvalidInput marks a caller-supplied value that failed domain validation.
	ErrInvalidInput = errors.New("invalid input")
	// ErrConflict marks a write that collides with an existing row (e.g. a duplicate
	// team color or tournament). Repositories translate the database's unique-violation
	// into this so the domain and API stay driver-agnostic.
	ErrConflict = errors.New("resource conflict")
)
