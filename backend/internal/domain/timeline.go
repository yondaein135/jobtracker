package domain

import (
	"fmt"
	"time"
)

// StatusEvent is a status change that has not been persisted yet.
type StatusEvent struct {
	Status    Status
	Notes     string
	ChangedAt time.Time
}

// ValidateTimeline checks that events form a valid pipeline history: it starts
// at SAVED or APPLIED, every step is an allowed forward transition, timestamps
// never go backwards, and nothing is later than now.
func ValidateTimeline(events []StatusEvent, now time.Time) error {
	if len(events) == 0 {
		return NewValidationError("history must contain at least one status")
	}

	first := events[0].Status
	if first != StatusSaved && first != StatusApplied {
		return NewValidationError("history must start with SAVED or APPLIED")
	}

	for i, e := range events {
		if !e.Status.Valid() {
			return NewValidationError(fmt.Sprintf("history[%d]: invalid status %q", i, e.Status))
		}
		if e.ChangedAt.IsZero() {
			return NewValidationError(fmt.Sprintf("history[%d]: changed_at is required", i))
		}
		if e.ChangedAt.After(now) {
			return NewValidationError(fmt.Sprintf("history[%d]: changed_at cannot be in the future", i))
		}
		if i == 0 {
			continue
		}
		prev := events[i-1]
		if e.ChangedAt.Before(prev.ChangedAt) {
			return NewValidationError(fmt.Sprintf("history[%d]: changed_at is earlier than the previous status", i))
		}
		if !prev.Status.CanTransitionTo(e.Status) {
			return &StatusTransitionError{From: prev.Status, To: e.Status}
		}
	}
	return nil
}
