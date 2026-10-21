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

// Create creates a new job application and logs its initial status event.
func (s *ApplicationService) Create(ctx context.Context, cmd ports.CreateApplicationCommand) (domain.ApplicationDetail, error) {
	cmd.CompanyName = strings.TrimSpace(cmd.CompanyName)
	cmd.RoleTitle = strings.TrimSpace(cmd.RoleTitle)
	cmd.Location = strings.TrimSpace(cmd.Location)
	cmd.JobURL = strings.TrimSpace(cmd.JobURL)
	cmd.SalaryRange = strings.TrimSpace(cmd.SalaryRange)
	cmd.Notes = strings.TrimSpace(cmd.Notes)
	cmd.JobDescription = strings.TrimSpace(cmd.JobDescription)

	if cmd.InitialStatus == "" {
		cmd.InitialStatus = domain.StatusApplied
	}

	if err := validateFields(&cmd.CompanyName, &cmd.RoleTitle, &cmd.Location, &cmd.JobURL, &cmd.SalaryRange, &cmd.Notes, &cmd.JobDescription, true); err != nil {
		return domain.ApplicationDetail{}, err
	}

	if cmd.InitialStatus != domain.StatusSaved && cmd.InitialStatus != domain.StatusApplied {
		return domain.ApplicationDetail{}, domain.NewValidationError("initial_status must be SAVED or APPLIED")
	}

	now := time.Now().UTC()
	appliedOn := now
	if cmd.AppliedOn != nil {
		appliedOn = cmd.AppliedOn.UTC()
	}

	initialChangedAt := now
	if cmd.InitialStatus == domain.StatusApplied {
		initialChangedAt = appliedOn
	}

	app := domain.Application{
		CompanyName:    cmd.CompanyName,
		RoleTitle:      cmd.RoleTitle,
		Location:       cmd.Location,
		JobURL:         cmd.JobURL,
		SalaryRange:    cmd.SalaryRange,
		Notes:          cmd.Notes,
		JobDescription: cmd.JobDescription,
		AppliedOn:      appliedOn,
		CreatedAt:      now,
		CurrentStatus:  cmd.InitialStatus,
	}

	return s.repo.Create(ctx, app, cmd.InitialStatus, appliedOn, initialChangedAt)
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
	if cmd.AppliedOn != nil {
		hasField = true
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

	changedAt := time.Now().UTC()
	if cmd.ChangedAt != nil {
		changedAt = cmd.ChangedAt.UTC()
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

	return s.repo.AddStatus(ctx, id, cmd.Status, cmd.Notes, changedAt)
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
