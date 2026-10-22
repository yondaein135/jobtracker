package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"job-tracker/backend/internal/app"
	"job-tracker/backend/internal/domain"
	"job-tracker/backend/internal/ports"
)

type mockAppRepo struct {
	listFn      func(ctx context.Context, filter ports.ListApplicationsFilter) ([]domain.Application, error)
	getByIDFn   func(ctx context.Context, id int64) (domain.ApplicationDetail, error)
	createFn    func(ctx context.Context, a domain.Application, history []domain.StatusEvent) (domain.ApplicationDetail, error)
	updateFn    func(ctx context.Context, id int64, cmd ports.UpdateApplicationCommand) (domain.ApplicationDetail, error)
	deleteFn    func(ctx context.Context, id int64) error
	addStatusFn func(ctx context.Context, id int64, s domain.Status, notes string, changed time.Time) (domain.ApplicationDetail, error)
}

func (m *mockAppRepo) List(ctx context.Context, filter ports.ListApplicationsFilter) ([]domain.Application, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockAppRepo) GetByID(ctx context.Context, id int64) (domain.ApplicationDetail, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return domain.ApplicationDetail{}, nil
}

func (m *mockAppRepo) Create(ctx context.Context, a domain.Application, history []domain.StatusEvent) (domain.ApplicationDetail, error) {
	if m.createFn != nil {
		return m.createFn(ctx, a, history)
	}
	return domain.ApplicationDetail{Application: a}, nil
}

func (m *mockAppRepo) Update(ctx context.Context, id int64, cmd ports.UpdateApplicationCommand) (domain.ApplicationDetail, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, cmd)
	}
	return domain.ApplicationDetail{}, nil
}

func (m *mockAppRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockAppRepo) AddStatus(ctx context.Context, id int64, s domain.Status, notes string, changed time.Time) (domain.ApplicationDetail, error) {
	if m.addStatusFn != nil {
		return m.addStatusFn(ctx, id, s, notes, changed)
	}
	return domain.ApplicationDetail{}, nil
}

func TestApplicationService_Validation(t *testing.T) {
	ctx := context.Background()
	svc := app.NewApplicationService(&mockAppRepo{})

	// 1. Missing company name
	_, err := svc.Create(ctx, ports.CreateApplicationCommand{
		RoleTitle: "Engineer",
	})
	if err == nil || !strings.Contains(err.Error(), "company_name is required") {
		t.Fatalf("expected company_name is required error, got %v", err)
	}

	// 2. Missing role title
	_, err = svc.Create(ctx, ports.CreateApplicationCommand{
		CompanyName: "Acme",
	})
	if err == nil || !strings.Contains(err.Error(), "role_title is required") {
		t.Fatalf("expected role_title is required error, got %v", err)
	}

	// 3. Invalid initial status
	_, err = svc.Create(ctx, ports.CreateApplicationCommand{
		CompanyName:   "Acme",
		RoleTitle:     "Engineer",
		InitialStatus: domain.StatusRejected,
	})
	if err == nil || !strings.Contains(err.Error(), "initial_status must be SAVED or APPLIED") {
		t.Fatalf("expected initial_status error, got %v", err)
	}

	// 4. Invalid URL scheme
	_, err = svc.Create(ctx, ports.CreateApplicationCommand{
		CompanyName: "Acme",
		RoleTitle:   "Engineer",
		JobURL:      "javascript:alert(1)",
	})
	if err == nil || !strings.Contains(err.Error(), "valid http(s) URL") {
		t.Fatalf("expected valid url error, got %v", err)
	}

	// 5. Update with empty company_name
	empty := ""
	_, err = svc.Update(ctx, 1, ports.UpdateApplicationCommand{
		CompanyName: &empty,
	})
	if err == nil || !strings.Contains(err.Error(), "company_name cannot be empty") {
		t.Fatalf("expected company_name cannot be empty error, got %v", err)
	}

	// 6. Update with no fields
	_, err = svc.Update(ctx, 1, ports.UpdateApplicationCommand{})
	if err == nil || !strings.Contains(err.Error(), "no updatable fields provided") {
		t.Fatalf("expected no updatable fields provided error, got %v", err)
	}

	// 7. AddStatus with invalid status
	_, err = svc.AddStatus(ctx, 1, ports.AddStatusCommand{
		Status: domain.Status("INVALID_STATUS"),
	})
	if err == nil || !strings.Contains(err.Error(), "invalid status") {
		t.Fatalf("expected invalid status error, got %v", err)
	}
}

func TestApplicationService_PassThrough(t *testing.T) {
	ctx := context.Background()
	var capturedID int64
	mock := &mockAppRepo{
		deleteFn: func(_ context.Context, id int64) error {
			capturedID = id
			return nil
		},
		getByIDFn: func(_ context.Context, id int64) (domain.ApplicationDetail, error) {
			if id == 42 {
				return domain.ApplicationDetail{Application: domain.Application{ID: 42, CompanyName: "Acme"}}, nil
			}
			return domain.ApplicationDetail{}, domain.ErrNotFound
		},
	}
	svc := app.NewApplicationService(mock)

	// Delete
	if err := svc.Delete(ctx, 99); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != 99 {
		t.Fatalf("expected captured ID 99, got %d", capturedID)
	}

	// GetByID success
	detail, err := svc.GetByID(ctx, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Application.CompanyName != "Acme" {
		t.Fatalf("expected company Acme, got %s", detail.Application.CompanyName)
	}

	// GetByID not found
	_, err = svc.GetByID(ctx, 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestApplicationService_AddStatus_Transitions(t *testing.T) {
	ctx := context.Background()

	mock := &mockAppRepo{
		getByIDFn: func(_ context.Context, id int64) (domain.ApplicationDetail, error) {
			switch id {
			case 1:
				return domain.ApplicationDetail{Application: domain.Application{ID: 1, CurrentStatus: domain.StatusSaved}}, nil
			case 2:
				return domain.ApplicationDetail{Application: domain.Application{ID: 2, CurrentStatus: domain.StatusApplied}}, nil
			case 3:
				return domain.ApplicationDetail{Application: domain.Application{ID: 3, CurrentStatus: domain.StatusInterviewing}}, nil
			case 4:
				return domain.ApplicationDetail{Application: domain.Application{ID: 4, CurrentStatus: domain.StatusOffer}}, nil
			case 5:
				return domain.ApplicationDetail{Application: domain.Application{ID: 5, CurrentStatus: domain.StatusRejected}}, nil
			default:
				return domain.ApplicationDetail{}, domain.ErrNotFound
			}
		},
		addStatusFn: func(_ context.Context, id int64, s domain.Status, notes string, changed time.Time) (domain.ApplicationDetail, error) {
			return domain.ApplicationDetail{Application: domain.Application{ID: id, CurrentStatus: s}}, nil
		},
	}
	svc := app.NewApplicationService(mock)

	// Valid forward transitions
	validCases := []struct {
		id int64
		to domain.Status
	}{
		{1, domain.StatusApplied},
		{2, domain.StatusResponseReceived},
		{2, domain.StatusInterviewing},
		{2, domain.StatusRejected},
		{3, domain.StatusOffer},
		{3, domain.StatusGhosted},
		{4, domain.StatusAccepted},
	}
	for _, tc := range validCases {
		res, err := svc.AddStatus(ctx, tc.id, ports.AddStatusCommand{Status: tc.to})
		if err != nil {
			t.Errorf("expected valid transition for app %d to %s, got err: %v", tc.id, tc.to, err)
		}
		if res.Application.CurrentStatus != tc.to {
			t.Errorf("expected new status %s, got %s", tc.to, res.Application.CurrentStatus)
		}
	}

	// Invalid transitions (backward, self, or from terminal)
	invalidCases := []struct {
		id int64
		to domain.Status
	}{
		{1, domain.StatusInterviewing},     // SAVED can only go to APPLIED
		{1, domain.StatusSaved},            // self
		{2, domain.StatusSaved},            // backward
		{2, domain.StatusApplied},          // self
		{3, domain.StatusApplied},          // backward
		{3, domain.StatusResponseReceived}, // backward
		{4, domain.StatusInterviewing},     // backward
		{5, domain.StatusInterviewing},     // from terminal REJECTED
		{5, domain.StatusAccepted},         // from terminal REJECTED
	}
	for _, tc := range invalidCases {
		_, err := svc.AddStatus(ctx, tc.id, ports.AddStatusCommand{Status: tc.to})
		if err == nil {
			t.Errorf("expected transition error for app %d to %s, got nil", tc.id, tc.to)
		}
		var transErr *domain.StatusTransitionError
		if !errors.As(err, &transErr) {
			t.Errorf("expected *domain.StatusTransitionError, got %T: %v", err, err)
		}
		if !errors.Is(err, domain.ErrInvalidTransition) {
			t.Errorf("expected errors.Is(err, domain.ErrInvalidTransition), got false")
		}
	}
}

func TestApplicationService_Backfill(t *testing.T) {
	ctx := context.Background()
	day := func(d int) time.Time { return time.Date(2026, 9, d, 0, 0, 0, 0, time.UTC) }

	var gotApp domain.Application
	var gotHistory []domain.StatusEvent
	svc := app.NewApplicationService(&mockAppRepo{
		createFn: func(_ context.Context, a domain.Application, h []domain.StatusEvent) (domain.ApplicationDetail, error) {
			gotApp, gotHistory = a, h
			return domain.ApplicationDetail{Application: a}, nil
		},
	})

	_, err := svc.Backfill(ctx, ports.BackfillApplicationCommand{
		CompanyName: "  Acme ",
		RoleTitle:   "Engineer",
		History: []ports.BackfillEvent{
			{Status: "saved", ChangedAt: day(1)},
			{Status: domain.StatusApplied, ChangedAt: day(3), Notes: " sent CV "},
			{Status: domain.StatusRejected, ChangedAt: day(20)},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotApp.CompanyName != "Acme" {
		t.Errorf("expected trimmed company name, got %q", gotApp.CompanyName)
	}
	if !gotApp.AppliedOn.Equal(day(3)) {
		t.Errorf("expected applied_on to be the APPLIED step date, got %v", gotApp.AppliedOn)
	}
	if gotApp.CurrentStatus != domain.StatusRejected {
		t.Errorf("expected current status REJECTED, got %s", gotApp.CurrentStatus)
	}
	if len(gotHistory) != 3 || gotHistory[0].Status != domain.StatusSaved || gotHistory[1].Notes != "sent CV" {
		t.Errorf("unexpected history passed to repo: %+v", gotHistory)
	}

	// Invalid timelines never reach the repo.
	gotHistory = nil
	_, err = svc.Backfill(ctx, ports.BackfillApplicationCommand{
		CompanyName: "Acme",
		RoleTitle:   "Engineer",
		History: []ports.BackfillEvent{
			{Status: domain.StatusApplied, ChangedAt: day(5)},
			{Status: domain.StatusInterviewing, ChangedAt: day(2)},
		},
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected validation error for out-of-order dates, got %v", err)
	}
	if gotHistory != nil {
		t.Fatal("repo should not be called for an invalid timeline")
	}
}

func TestApplicationService_Create_UsesNow(t *testing.T) {
	var gotHistory []domain.StatusEvent
	svc := app.NewApplicationService(&mockAppRepo{
		createFn: func(_ context.Context, a domain.Application, h []domain.StatusEvent) (domain.ApplicationDetail, error) {
			gotHistory = h
			return domain.ApplicationDetail{Application: a}, nil
		},
	})
	before := time.Now().UTC()
	_, err := svc.Create(context.Background(), ports.CreateApplicationCommand{CompanyName: "Acme", RoleTitle: "Engineer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotHistory) != 1 || gotHistory[0].Status != domain.StatusApplied {
		t.Fatalf("expected a single APPLIED event, got %+v", gotHistory)
	}
	if gotHistory[0].ChangedAt.Before(before) {
		t.Errorf("expected initial event at current time, got %v", gotHistory[0].ChangedAt)
	}
}
