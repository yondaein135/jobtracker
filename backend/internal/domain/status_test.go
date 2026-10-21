package domain_test

import (
	"testing"

	"job-tracker/backend/internal/domain"
)

func TestStatus_Valid(t *testing.T) {
	validStatuses := []domain.Status{
		domain.StatusSaved,
		domain.StatusApplied,
		domain.StatusResponseReceived,
		domain.StatusInterviewing,
		domain.StatusOffer,
		domain.StatusAccepted,
		domain.StatusRejected,
		domain.StatusGhosted,
		domain.StatusWithdrawn,
	}

	for _, s := range validStatuses {
		if !s.Valid() {
			t.Errorf("expected status %s to be valid", s)
		}
		if s.Rank() < 0 {
			t.Errorf("expected status %s to have non-negative rank, got %d", s, s.Rank())
		}
	}

	invalidStatus := domain.Status("UNKNOWN")
	if invalidStatus.Valid() {
		t.Error("expected UNKNOWN status to be invalid")
	}
	if invalidStatus.Rank() != -1 {
		t.Errorf("expected invalid status rank -1, got %d", invalidStatus.Rank())
	}
}

func TestStatus_IsActive(t *testing.T) {
	activeStatuses := []domain.Status{
		domain.StatusApplied,
		domain.StatusResponseReceived,
		domain.StatusInterviewing,
		domain.StatusOffer,
	}

	for _, s := range activeStatuses {
		if !s.IsActive() {
			t.Errorf("expected status %s to be active", s)
		}
	}

	inactiveStatuses := []domain.Status{
		domain.StatusSaved,
		domain.StatusAccepted,
		domain.StatusRejected,
		domain.StatusGhosted,
		domain.StatusWithdrawn,
	}

	for _, s := range inactiveStatuses {
		if s.IsActive() {
			t.Errorf("expected status %s to be inactive", s)
		}
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	terminal := []domain.Status{
		domain.StatusAccepted,
		domain.StatusRejected,
		domain.StatusGhosted,
		domain.StatusWithdrawn,
	}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("expected %s to be terminal", s)
		}
		if next := s.NextAllowedStatuses(); len(next) != 0 {
			t.Errorf("expected terminal status %s to have no next statuses, got %v", s, next)
		}
	}

	nonTerminal := []domain.Status{
		domain.StatusSaved,
		domain.StatusApplied,
		domain.StatusResponseReceived,
		domain.StatusInterviewing,
		domain.StatusOffer,
	}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("expected %s to be non-terminal", s)
		}
	}
}

func TestStatus_Transitions(t *testing.T) {
	tests := []struct {
		from     domain.Status
		to       domain.Status
		expected bool
	}{
		// SAVED transitions
		{domain.StatusSaved, domain.StatusApplied, true},
		{domain.StatusSaved, domain.StatusSaved, false},
		{domain.StatusSaved, domain.StatusInterviewing, false},
		{domain.StatusSaved, domain.StatusRejected, false},
		{domain.StatusSaved, domain.StatusWithdrawn, false},

		// APPLIED transitions (can step or skip forward)
		{domain.StatusApplied, domain.StatusResponseReceived, true},
		{domain.StatusApplied, domain.StatusInterviewing, true},
		{domain.StatusApplied, domain.StatusOffer, true},
		{domain.StatusApplied, domain.StatusAccepted, true},
		{domain.StatusApplied, domain.StatusRejected, true},
		{domain.StatusApplied, domain.StatusGhosted, true},
		{domain.StatusApplied, domain.StatusWithdrawn, true},
		{domain.StatusApplied, domain.StatusApplied, false},
		{domain.StatusApplied, domain.StatusSaved, false}, // backward

		// RESPONSE_RECEIVED transitions
		{domain.StatusResponseReceived, domain.StatusInterviewing, true},
		{domain.StatusResponseReceived, domain.StatusOffer, true},
		{domain.StatusResponseReceived, domain.StatusAccepted, true},
		{domain.StatusResponseReceived, domain.StatusRejected, true},
		{domain.StatusResponseReceived, domain.StatusGhosted, true},
		{domain.StatusResponseReceived, domain.StatusWithdrawn, true},
		{domain.StatusResponseReceived, domain.StatusResponseReceived, false},
		{domain.StatusResponseReceived, domain.StatusApplied, false}, // backward
		{domain.StatusResponseReceived, domain.StatusSaved, false},   // backward

		// INTERVIEWING transitions
		{domain.StatusInterviewing, domain.StatusOffer, true},
		{domain.StatusInterviewing, domain.StatusAccepted, true},
		{domain.StatusInterviewing, domain.StatusRejected, true},
		{domain.StatusInterviewing, domain.StatusGhosted, true},
		{domain.StatusInterviewing, domain.StatusWithdrawn, true},
		{domain.StatusInterviewing, domain.StatusInterviewing, false},
		{domain.StatusInterviewing, domain.StatusResponseReceived, false}, // backward
		{domain.StatusInterviewing, domain.StatusApplied, false},          // backward
		{domain.StatusInterviewing, domain.StatusSaved, false},            // backward

		// OFFER transitions
		{domain.StatusOffer, domain.StatusAccepted, true},
		{domain.StatusOffer, domain.StatusRejected, true},
		{domain.StatusOffer, domain.StatusGhosted, true},
		{domain.StatusOffer, domain.StatusWithdrawn, true},
		{domain.StatusOffer, domain.StatusOffer, false},
		{domain.StatusOffer, domain.StatusInterviewing, false}, // backward
		{domain.StatusOffer, domain.StatusApplied, false},      // backward

		// Terminal states cannot transition anywhere
		{domain.StatusAccepted, domain.StatusOffer, false},
		{domain.StatusAccepted, domain.StatusAccepted, false},
		{domain.StatusRejected, domain.StatusInterviewing, false},
		{domain.StatusRejected, domain.StatusApplied, false},
		{domain.StatusGhosted, domain.StatusInterviewing, false},
		{domain.StatusGhosted, domain.StatusRejected, false},
		{domain.StatusWithdrawn, domain.StatusApplied, false},
		{domain.StatusWithdrawn, domain.StatusRejected, false},

		// Unknown / invalid status
		{domain.StatusApplied, domain.Status("UNKNOWN"), false},
		{domain.Status("UNKNOWN"), domain.StatusApplied, false},
	}

	for _, tt := range tests {
		got := tt.from.CanTransitionTo(tt.to)
		if got != tt.expected {
			t.Errorf("CanTransitionTo(%s -> %s) = %v; want %v", tt.from, tt.to, got, tt.expected)
		}
	}
}
