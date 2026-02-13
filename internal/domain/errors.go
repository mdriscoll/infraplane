package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

// ValidationError represents a validation failure.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.Message)
}

// ErrValidation creates a new ValidationError.
func ErrValidation(msg string) error {
	return &ValidationError{Message: msg}
}

// IsValidationError checks if an error is a ValidationError.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
