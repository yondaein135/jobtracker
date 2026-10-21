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
	createFn    func(ctx context.Context, a domain.Application, s domain.Status, applied, changed time.Time) (domain.ApplicationDetail, error)
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

func (m *mockAppRepo) Create(ctx context.Context, a domain.Application, s domain.Status, applied, changed time.Time) (domain.ApplicationDetail, error) {
	if m.createFn != nil {
		return m.createFn(ctx, a, s, applied, changed)
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
