package ports

import (
	"context"
	"time"

	"job-tracker/backend/internal/domain"
)

// ListApplicationsFilter specifies filtering options for listing applications.
type ListApplicationsFilter struct {
	Status domain.Status
	Search string
}

// CreateApplicationCommand contains fields required to create a new application.
type CreateApplicationCommand struct {
	CompanyName    string
	RoleTitle      string
	Location       string
	JobURL         string
	SalaryRange    string
	Notes          string
	JobDescription string
	InitialStatus  domain.Status
}

// BackfillApplicationCommand records an application that already happened,
// together with its full dated status history (oldest first).
type BackfillApplicationCommand struct {
	CompanyName    string
	RoleTitle      string
	Location       string
	JobURL         string
	SalaryRange    string
	Notes          string
	JobDescription string
	History        []BackfillEvent
}

// BackfillEvent is one past status change in a backfilled application.
type BackfillEvent struct {
	Status    domain.Status
	Notes     string
	ChangedAt time.Time
}

// UpdateApplicationCommand contains fields that can be updated on an application.
// Nil fields represent values that should remain untouched.
type UpdateApplicationCommand struct {
	CompanyName    *string
	RoleTitle      *string
	Location       *string
	JobURL         *string
	SalaryRange    *string
	Notes          *string
	JobDescription *string
}

// AddStatusCommand represents a new status transition event for an application.
// The change is always recorded at the current time; use Backfill for past events.
type AddStatusCommand struct {
	Status domain.Status
	Notes  string
}

// ApplicationService defines the primary/driving inbound port for application operations.
type ApplicationService interface {
	List(ctx context.Context, filter ListApplicationsFilter) ([]domain.Application, error)
	GetByID(ctx context.Context, id int64) (domain.ApplicationDetail, error)
	Create(ctx context.Context, cmd CreateApplicationCommand) (domain.ApplicationDetail, error)
	Backfill(ctx context.Context, cmd BackfillApplicationCommand) (domain.ApplicationDetail, error)
	Update(ctx context.Context, id int64, cmd UpdateApplicationCommand) (domain.ApplicationDetail, error)
	Delete(ctx context.Context, id int64) error
	AddStatus(ctx context.Context, id int64, cmd AddStatusCommand) (domain.ApplicationDetail, error)
}

// AnalyticsService defines the primary/driving inbound port for analytics computations.
type AnalyticsService interface {
	GetGraphAnalytics(ctx context.Context) (domain.GraphAnalytics, error)
}
