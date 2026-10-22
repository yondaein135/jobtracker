package http

import (
	"time"

	"job-tracker/backend/internal/domain"
)

// CreateApplicationRequest is the payload for POST /api/applications.
type CreateApplicationRequest struct {
	CompanyName    string `json:"company_name"`
	RoleTitle      string `json:"role_title"`
	Location       string `json:"location"`
	JobURL         string `json:"job_url"`
	SalaryRange    string `json:"salary_range"`
	Notes          string `json:"notes"`
	JobDescription string `json:"job_description"`
	InitialStatus  string `json:"initial_status"`
}

// BackfillApplicationRequest is the payload for POST /api/applications/backfill.
// History lists past status changes, oldest first.
type BackfillApplicationRequest struct {
	CompanyName    string                 `json:"company_name"`
	RoleTitle      string                 `json:"role_title"`
	Location       string                 `json:"location"`
	JobURL         string                 `json:"job_url"`
	SalaryRange    string                 `json:"salary_range"`
	Notes          string                 `json:"notes"`
	JobDescription string                 `json:"job_description"`
	History        []BackfillEventRequest `json:"history"`
}

// BackfillEventRequest is one past status change in a backfill request.
type BackfillEventRequest struct {
	Status    string `json:"status"`
	Notes     string `json:"notes"`
	ChangedAt string `json:"changed_at"`
}

// UpdateApplicationRequest is the payload for PATCH /api/applications/{id}.
type UpdateApplicationRequest struct {
	CompanyName    *string `json:"company_name"`
	RoleTitle      *string `json:"role_title"`
	Location       *string `json:"location"`
	JobURL         *string `json:"job_url"`
	SalaryRange    *string `json:"salary_range"`
	Notes          *string `json:"notes"`
	JobDescription *string `json:"job_description"`
}

// StatusChangeRequest is the payload for POST /api/applications/{id}/status.
// The change is recorded at the current time.
type StatusChangeRequest struct {
	Status string `json:"status"`
	Notes  string `json:"notes"`
}

// ApplicationResponse is the JSON representation of an application.
type ApplicationResponse struct {
	ID                  int64           `json:"id"`
	CompanyName         string          `json:"company_name"`
	RoleTitle           string          `json:"role_title"`
	Location            string          `json:"location"`
	JobURL              string          `json:"job_url"`
	SalaryRange         string          `json:"salary_range"`
	Notes               string          `json:"notes"`
	JobDescription      string          `json:"job_description"`
	AppliedOn           string          `json:"applied_on"`
	CreatedAt           string          `json:"created_at"`
	CurrentStatus       domain.Status   `json:"current_status"`
	StatusUpdatedAt     string          `json:"status_updated_at"`
	NextAllowedStatuses []domain.Status `json:"next_allowed_statuses"`
}

// StatusHistoryResponse is the JSON representation of a status history entry.
type StatusHistoryResponse struct {
	ID            int64         `json:"id"`
	ApplicationID int64         `json:"application_id"`
	Status        domain.Status `json:"status"`
	Notes         string        `json:"notes"`
	ChangedAt     string        `json:"changed_at"`
}

// ApplicationDetailResponse is the JSON representation of an application and its history.
type ApplicationDetailResponse struct {
	ApplicationResponse
	History []StatusHistoryResponse `json:"history"`
}

// SankeyNodeResponse represents a node in the Sankey visualization.
type SankeyNodeResponse struct {
	Name  domain.Status `json:"name"`
	Value int           `json:"value,omitempty"`
}

// SankeyLinkResponse represents a directed transition in the Sankey visualization.
type SankeyLinkResponse struct {
	Source domain.Status `json:"source"`
	Target domain.Status `json:"target"`
	Value  int           `json:"value"`
}

// GraphAnalyticsResponse is the JSON representation of graph analytics metrics.
type GraphAnalyticsResponse struct {
	TotalApplied         int                   `json:"total_applied"`
	TotalApplications    int                   `json:"total_applications"`
	RespondedCount       int                   `json:"responded_count"`
	OfferCount           int                   `json:"offer_count"`
	CurrentDistribution  map[domain.Status]int `json:"current_distribution"`
	ReachedCounts        map[domain.Status]int `json:"reached_counts"`
	SankeyNodes          []SankeyNodeResponse  `json:"sankey_nodes"`
	SankeyLinks          []SankeyLinkResponse  `json:"sankey_links"`
	OmittedBackwardLinks int                   `json:"omitted_backward_links"`
}

func toApplicationResponse(a domain.Application) ApplicationResponse {
	nextStatuses := a.CurrentStatus.NextAllowedStatuses()
	if nextStatuses == nil {
		nextStatuses = []domain.Status{}
	}
	return ApplicationResponse{
		ID:                  a.ID,
		CompanyName:         a.CompanyName,
		RoleTitle:           a.RoleTitle,
		Location:            a.Location,
		JobURL:              a.JobURL,
		SalaryRange:         a.SalaryRange,
		Notes:               a.Notes,
		JobDescription:      a.JobDescription,
		AppliedOn:           formatRFC3339(a.AppliedOn),
		CreatedAt:           formatRFC3339(a.CreatedAt),
		CurrentStatus:       a.CurrentStatus,
		StatusUpdatedAt:     formatRFC3339(a.StatusUpdatedAt),
		NextAllowedStatuses: nextStatuses,
	}
}

func toStatusHistoryResponse(e domain.StatusHistoryEntry) StatusHistoryResponse {
	return StatusHistoryResponse{
		ID:            e.ID,
		ApplicationID: e.ApplicationID,
		Status:        e.Status,
		Notes:         e.Notes,
		ChangedAt:     formatRFC3339(e.ChangedAt),
	}
}

func toApplicationDetailResponse(d domain.ApplicationDetail) ApplicationDetailResponse {
	history := make([]StatusHistoryResponse, 0, len(d.History))
	for _, h := range d.History {
		history = append(history, toStatusHistoryResponse(h))
	}
	return ApplicationDetailResponse{
		ApplicationResponse: toApplicationResponse(d.Application),
		History:             history,
	}
}

func toGraphAnalyticsResponse(g domain.GraphAnalytics) GraphAnalyticsResponse {
	nodes := make([]SankeyNodeResponse, 0, len(g.SankeyNodes))
	for _, n := range g.SankeyNodes {
		nodes = append(nodes, SankeyNodeResponse{
			Name:  n.Name,
			Value: n.Value,
		})
	}

	links := make([]SankeyLinkResponse, 0, len(g.SankeyLinks))
	for _, l := range g.SankeyLinks {
		links = append(links, SankeyLinkResponse{
			Source: l.Source,
			Target: l.Target,
			Value:  l.Value,
		})
	}

	return GraphAnalyticsResponse{
		TotalApplied:         g.TotalApplied,
		TotalApplications:    g.TotalApplications,
		RespondedCount:       g.RespondedCount,
		OfferCount:           g.OfferCount,
		CurrentDistribution:  g.CurrentDistribution,
		ReachedCounts:        g.ReachedCounts,
		SankeyNodes:          nodes,
		SankeyLinks:          links,
		OmittedBackwardLinks: g.OmittedBackwardLinks,
	}
}

func formatRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
