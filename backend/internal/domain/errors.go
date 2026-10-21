package domain

import "errors"

var (
	// ErrNotFound is returned when an entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrInvalidInput is returned when user input fails domain validation.
	ErrInvalidInput = errors.New("invalid input")
)

// ValidationError represents a business rule or validation constraint violation.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidInput
}

// NewValidationError creates a new ValidationError with the given message.
func NewValidationError(msg string) error {
	return &ValidationError{Message: msg}
}
