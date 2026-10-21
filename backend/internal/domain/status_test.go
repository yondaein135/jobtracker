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
