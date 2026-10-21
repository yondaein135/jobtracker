package app

import (
	"context"

	"job-tracker/backend/internal/domain"
	"job-tracker/backend/internal/ports"
)

// AnalyticsService orchestrates analytics data queries and business calculations.
type AnalyticsService struct {
	repo ports.AnalyticsRepository
}

// NewAnalyticsService creates a new AnalyticsService instance.
func NewAnalyticsService(repo ports.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// GetGraphAnalytics retrieves funnel distribution, reach counts, and Sankey transitions.
func (s *AnalyticsService) GetGraphAnalytics(ctx context.Context) (domain.GraphAnalytics, error) {
	return s.repo.GetGraphAnalytics(ctx)
}
