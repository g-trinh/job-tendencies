package kernel

import (
	"errors"
	"fmt"
)

// Sentinel errors for domain-level failures. HTTP handlers map these to status codes
// via RespondError; see internal/handler/http.
var (
	// ErrNotFound is returned when an aggregate or entity cannot be found by its ID.
	ErrNotFound = errors.New("not found")

	// ErrConflict is returned when an operation would violate a uniqueness constraint
	// (e.g. duplicate content hash, duplicate job fingerprint).
	ErrConflict = errors.New("conflict")

	// ErrInvalidInput is returned when caller-supplied data fails domain validation.
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized is returned when the caller lacks permission for the operation.
	ErrUnauthorized = errors.New("unauthorized")
)

// ValidationError carries structured field-level validation failures for a single
// domain object. Use errors.As to extract the details at the handler boundary.
//
// Example:
//
//	var ve *kernel.ValidationError
//	if errors.As(err, &ve) {
//	    // render field-level error in the response
//	}
type ValidationError struct {
	// Field is the name of the invalid field (empty for object-level violations).
	Field string
	// Message is a human-readable description of the violation.
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("validation error: %s", e.Message)
	}
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// Is allows errors.Is(err, kernel.ErrInvalidInput) to match any ValidationError.
func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidInput
}

// NotFoundError carries context for a missing resource lookup.
//
// Example:
//
//	var nfe *kernel.NotFoundError
//	if errors.As(err, &nfe) {
//	    // render 404 with kind-specific message
//	}
type NotFoundError struct {
	// Kind is the resource type that was not found (e.g. "job", "board").
	Kind string
	// ID is the identifier that was looked up.
	ID string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Kind, e.ID)
}

// Is allows errors.Is(err, kernel.ErrNotFound) to match any NotFoundError.
func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}
