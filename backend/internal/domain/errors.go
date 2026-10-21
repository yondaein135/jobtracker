package domain

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when an entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrInvalidInput is returned when user input fails domain validation.
	ErrInvalidInput = errors.New("invalid input")

	// ErrInvalidTransition is returned when a status transition violates the directional flow.
	ErrInvalidTransition = errors.New("invalid status transition")
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

// StatusTransitionError represents an invalid status transition attempt.
type StatusTransitionError struct {
	From Status
	To   Status
}

func (e *StatusTransitionError) Error() string {
	if e.From.IsTerminal() {
		return fmt.Sprintf("cannot transition from terminal status %s", e.From)
	}
	if e.From == e.To {
		return fmt.Sprintf("application is already in status %s", e.From)
	}
	return fmt.Sprintf("cannot transition status backward or across invalid path from %s to %s", e.From, e.To)
}

func (e *StatusTransitionError) Is(target error) bool {
	return target == ErrInvalidTransition || target == ErrInvalidInput
}
