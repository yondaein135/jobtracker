package ports

import (
	"context"
	"time"

	"job-tracker/backend/internal/domain"
)

// ApplicationRepository defines the driven outbound port for persisting application data.
type ApplicationRepository interface {
	List(ctx context.Context, filter ListApplicationsFilter) ([]domain.Application, error)
	GetByID(ctx context.Context, id int64) (domain.ApplicationDetail, error)
	Create(ctx context.Context, app domain.Application, history []domain.StatusEvent) (domain.ApplicationDetail, error)
	Update(ctx context.Context, id int64, cmd UpdateApplicationCommand) (domain.ApplicationDetail, error)
	Delete(ctx context.Context, id int64) error
	AddStatus(ctx context.Context, id int64, status domain.Status, notes string, changedAt time.Time) (domain.ApplicationDetail, error)
}

// AnalyticsRepository defines the driven outbound port for reading application metrics.
type AnalyticsRepository interface {
	GetGraphAnalytics(ctx context.Context) (domain.GraphAnalytics, error)
}

// HealthChecker defines the outbound port for infrastructure health probes.
type HealthChecker interface {
	Ping(ctx context.Context) error
}
