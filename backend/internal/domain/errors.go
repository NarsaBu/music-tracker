// Package domain contains business entities and rules. It must not import
// any other package of this project or any infrastructure library.
package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	// ErrConflict: the operation does not fit the current state of the resource.
	ErrConflict            = errors.New("conflict")
	ErrValidation          = errors.New("validation failed")
	ErrUpstreamUnavailable = errors.New("upstream unavailable")
)

// FieldError describes a validation problem of a single input field.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError carries field-level details and matches ErrValidation via errors.Is.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return ErrValidation.Error()
	}
	return ErrValidation.Error() + ": " + e.Fields[0].Field + ": " + e.Fields[0].Message
}

func (e *ValidationError) Is(target error) bool { return target == ErrValidation }
