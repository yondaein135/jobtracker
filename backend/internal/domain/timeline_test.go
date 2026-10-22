package domain_test

import (
	"errors"
	"testing"
	"time"

	"job-tracker/backend/internal/domain"
)

func TestValidateTimeline(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	ev := func(s domain.Status, at time.Time) domain.StatusEvent {
		return domain.StatusEvent{Status: s, ChangedAt: at}
	}

	tests := []struct {
		name       string
		events     []domain.StatusEvent
		wantErr    bool
		transition bool
	}{
		{"single applied", []domain.StatusEvent{ev(domain.StatusApplied, day(1))}, false, false},
		{"single saved", []domain.StatusEvent{ev(domain.StatusSaved, day(1))}, false, false},
		{"full flow", []domain.StatusEvent{
			ev(domain.StatusSaved, day(1)),
			ev(domain.StatusApplied, day(2)),
			ev(domain.StatusInterviewing, day(10)),
			ev(domain.StatusRejected, day(15)),
		}, false, false},
		{"same day steps", []domain.StatusEvent{
			ev(domain.StatusApplied, day(1)),
			ev(domain.StatusRejected, day(1)),
		}, false, false},
		{"today is allowed", []domain.StatusEvent{ev(domain.StatusApplied, now)}, false, false},

		{"empty", nil, true, false},
		{"starts mid-pipeline", []domain.StatusEvent{ev(domain.StatusInterviewing, day(1))}, true, false},
		{"missing date", []domain.StatusEvent{{Status: domain.StatusApplied}}, true, false},
		{"future date", []domain.StatusEvent{ev(domain.StatusApplied, now.Add(time.Hour))}, true, false},
		{"dates out of order", []domain.StatusEvent{
			ev(domain.StatusApplied, day(5)),
			ev(domain.StatusInterviewing, day(3)),
		}, true, false},
		{"backward transition", []domain.StatusEvent{
			ev(domain.StatusApplied, day(1)),
			ev(domain.StatusInterviewing, day(2)),
			ev(domain.StatusResponseReceived, day(3)),
		}, true, true},
		{"after terminal", []domain.StatusEvent{
			ev(domain.StatusApplied, day(1)),
			ev(domain.StatusRejected, day(2)),
			ev(domain.StatusOffer, day(3)),
		}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateTimeline(tt.events, now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTimeline() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("expected error to match ErrInvalidInput, got %v", err)
			}
			if tt.transition && !errors.Is(err, domain.ErrInvalidTransition) {
				t.Errorf("expected ErrInvalidTransition, got %v", err)
			}
		})
	}
}
