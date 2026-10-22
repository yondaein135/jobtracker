package app

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"job-tracker/backend/internal/domain"
	"job-tracker/backend/internal/ports"
)

const (
	maxShortField           = 300
	maxNotesLength          = 20_000
	maxJobDescriptionLength = 50_000
	maxURLLength            = 2048
)

// ApplicationService orchestrates application business use-cases.
type ApplicationService struct {
	repo ports.ApplicationRepository
}

// NewApplicationService creates a new ApplicationService instance.
func NewApplicationService(repo ports.ApplicationRepository) *ApplicationService {
	return &ApplicationService{repo: repo}
}

// List retrieves applications filtered by status and/or search keywords.
func (s *ApplicationService) List(ctx context.Context, filter ports.ListApplicationsFilter) ([]domain.Application, error) {
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, domain.NewValidationError(fmt.Sprintf("invalid status %q", filter.Status))
	}
	return s.repo.List(ctx, filter)
}

// GetByID returns the complete application aggregate including status history.
func (s *ApplicationService) GetByID(ctx context.Context, id int64) (domain.ApplicationDetail, error) {
	if id <= 0 {
		return domain.ApplicationDetail{}, domain.NewValidationError("invalid application id")
	}
	return s.repo.GetByID(ctx, id)
}

// Create records a new job application as of now, with a single initial status
// of SAVED or APPLIED. Past applications go through Backfill instead.
func (s *ApplicationService) Create(ctx context.Context, cmd ports.CreateApplicationCommand) (domain.ApplicationDetail, error) {
	app := domain.Application{
		CompanyName:    cmd.CompanyName,
		RoleTitle:      cmd.RoleTitle,
		Location:       cmd.Location,
		JobURL:         cmd.JobURL,
		SalaryRange:    cmd.SalaryRange,
		Notes:          cmd.Notes,
		JobDescription: cmd.JobDescription,
	}
	if err := normalizeAndValidate(&app); err != nil {
		return domain.ApplicationDetail{}, err
	}

	if cmd.InitialStatus == "" {
		cmd.InitialStatus = domain.StatusApplied
	}
	if cmd.InitialStatus != domain.StatusSaved && cmd.InitialStatus != domain.StatusApplied {
		return domain.ApplicationDetail{}, domain.NewValidationError("initial_status must be SAVED or APPLIED")
	}

	now := time.Now().UTC()
	app.AppliedOn = now
	app.CreatedAt = now
	app.CurrentStatus = cmd.InitialStatus

	history := []domain.StatusEvent{{Status: cmd.InitialStatus, Notes: "Application created", ChangedAt: now}}
	return s.repo.Create(ctx, app, history)
}

// Backfill records an application that already happened, with its full dated
// status history. The history must be a valid forward path and cannot be in the future.
func (s *ApplicationService) Backfill(ctx context.Context, cmd ports.BackfillApplicationCommand) (domain.ApplicationDetail, error) {
	app := domain.Application{
		CompanyName:    cmd.CompanyName,
		RoleTitle:      cmd.RoleTitle,
		Location:       cmd.Location,
		JobURL:         cmd.JobURL,
		SalaryRange:    cmd.SalaryRange,
		Notes:          cmd.Notes,
		JobDescription: cmd.JobDescription,
	}
	if err := normalizeAndValidate(&app); err != nil {
		return domain.ApplicationDetail{}, err
	}

	history := make([]domain.StatusEvent, 0, len(cmd.History))
	for i, e := range cmd.History {
		notes := strings.TrimSpace(e.Notes)
		if len(notes) > maxNotesLength {
			return domain.ApplicationDetail{}, domain.NewValidationError(fmt.Sprintf("history[%d]: notes too long", i))
		}
		history = append(history, domain.StatusEvent{
			Status:    domain.Status(strings.ToUpper(strings.TrimSpace(string(e.Status)))),
			Notes:     notes,
			ChangedAt: e.ChangedAt.UTC(),
		})
	}

	now := time.Now().UTC()
	if err := domain.ValidateTimeline(history, now); err != nil {
		return domain.ApplicationDetail{}, err
	}

	// applied_on is when the APPLIED step happened. A history that never
	// reached APPLIED falls back to its first step, matching Create for SAVED.
	app.AppliedOn = history[0].ChangedAt
	for _, e := range history {
		if e.Status == domain.StatusApplied {
			app.AppliedOn = e.ChangedAt
			break
		}
	}
	app.CreatedAt = now
	app.CurrentStatus = history[len(history)-1].Status

	return s.repo.Create(ctx, app, history)
}

// Update partially updates fields on an existing job application.
func (s *ApplicationService) Update(ctx context.Context, id int64, cmd ports.UpdateApplicationCommand) (domain.ApplicationDetail, error) {
	if id <= 0 {
		return domain.ApplicationDetail{}, domain.NewValidationError("invalid application id")
	}

	hasField := false
	if cmd.CompanyName != nil {
		hasField = true
		v := strings.TrimSpace(*cmd.CompanyName)
		cmd.CompanyName = &v
		if v == "" {
			return domain.ApplicationDetail{}, domain.NewValidationError("company_name cannot be empty")
		}
	}
	if cmd.RoleTitle != nil {
		hasField = true
		v := strings.TrimSpace(*cmd.RoleTitle)
		cmd.RoleTitle = &v
		if v == "" {
			return domain.ApplicationDetail{}, domain.NewValidationError("role_title cannot be empty")
		}
	}
	if cmd.Location != nil {
		hasField = true
		v := strings.TrimSpace(*cmd.Location)
		cmd.Location = &v
	}
	if cmd.JobURL != nil {
		hasField = true
		v := strings.TrimSpace(*cmd.JobURL)
		cmd.JobURL = &v
	}
	if cmd.SalaryRange != nil {
		hasField = true
		v := strings.TrimSpace(*cmd.SalaryRange)
		cmd.SalaryRange = &v
	}
	if cmd.Notes != nil {
		hasField = true
		v := strings.TrimSpace(*cmd.Notes)
		cmd.Notes = &v
	}
	if cmd.JobDescription != nil {
		hasField = true
		v := strings.TrimSpace(*cmd.JobDescription)
		cmd.JobDescription = &v
	}
	if !hasField {
		return domain.ApplicationDetail{}, domain.NewValidationError("no updatable fields provided")
	}

	empty := ""
	or := func(p *string) *string {
		if p == nil {
			return &empty
		}
		return p
	}

	if err := validateFields(or(cmd.CompanyName), or(cmd.RoleTitle), or(cmd.Location), or(cmd.JobURL), or(cmd.SalaryRange), or(cmd.Notes), or(cmd.JobDescription), false); err != nil {
		return domain.ApplicationDetail{}, err
	}

	return s.repo.Update(ctx, id, cmd)
}

// Delete removes an application and all associated status history.
func (s *ApplicationService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return domain.NewValidationError("invalid application id")
	}
	return s.repo.Delete(ctx, id)
}

// AddStatus appends a new status event to an application's history.
func (s *ApplicationService) AddStatus(ctx context.Context, id int64, cmd ports.AddStatusCommand) (domain.ApplicationDetail, error) {
	if id <= 0 {
		return domain.ApplicationDetail{}, domain.NewValidationError("invalid application id")
	}

	cmd.Status = domain.Status(strings.ToUpper(strings.TrimSpace(string(cmd.Status))))
	if !cmd.Status.Valid() {
		return domain.ApplicationDetail{}, domain.NewValidationError(fmt.Sprintf("invalid status %q", cmd.Status))
	}

	cmd.Notes = strings.TrimSpace(cmd.Notes)
	if len(cmd.Notes) > maxNotesLength {
		return domain.ApplicationDetail{}, domain.NewValidationError("notes too long")
	}

	detail, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.ApplicationDetail{}, err
	}

	if !detail.Application.CurrentStatus.CanTransitionTo(cmd.Status) {
		return domain.ApplicationDetail{}, &domain.StatusTransitionError{
			From: detail.Application.CurrentStatus,
			To:   cmd.Status,
		}
	}

	return s.repo.AddStatus(ctx, id, cmd.Status, cmd.Notes, time.Now().UTC())
}

// normalizeAndValidate trims the application's text fields and checks them.
func normalizeAndValidate(a *domain.Application) error {
	for _, f := range []*string{&a.CompanyName, &a.RoleTitle, &a.Location, &a.JobURL, &a.SalaryRange, &a.Notes, &a.JobDescription} {
		*f = strings.TrimSpace(*f)
	}
	return validateFields(&a.CompanyName, &a.RoleTitle, &a.Location, &a.JobURL, &a.SalaryRange, &a.Notes, &a.JobDescription, true)
}

func validateFields(company, role, location, jobURL, salary, notes, jobDesc *string, requireCore bool) error {
	if requireCore && *company == "" {
		return domain.NewValidationError("company_name is required")
	}
	if requireCore && *role == "" {
		return domain.NewValidationError("role_title is required")
	}
	for name, v := range map[string]string{
		"company_name": *company,
		"role_title":   *role,
		"location":     *location,
		"salary_range": *salary,
	} {
		if len(v) > maxShortField {
			return domain.NewValidationError(name + " too long")
		}
	}
	if len(*notes) > maxNotesLength {
		return domain.NewValidationError("notes too long")
	}
	if jobDesc != nil && len(*jobDesc) > maxJobDescriptionLength {
		return domain.NewValidationError("job_description too long")
	}
	if err := validateURL(*jobURL); err != nil {
		return err
	}
	return nil
}

func validateURL(raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > maxURLLength {
		return domain.NewValidationError("job_url too long")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return domain.NewValidationError("job_url must be a valid http(s) URL")
	}
	return nil
}
