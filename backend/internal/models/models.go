// Package models defines the API/domain types shared across the backend.
package models

// Status is a canonical application pipeline status.
type Status string

const (
	StatusSaved            Status = "SAVED"             // Bookmarked, not yet applied
	StatusApplied          Status = "APPLIED"           // Application sent
	StatusResponseReceived Status = "RESPONSE_RECEIVED" // Recruiter screen / initial outreach
	StatusInterviewing     Status = "INTERVIEWING"      // Technical / behavioral rounds
	StatusOffer            Status = "OFFER"             // Offer extended
	StatusAccepted         Status = "ACCEPTED"          // Offer signed
	StatusRejected         Status = "REJECTED"          // Rejected at any stage
	StatusGhosted          Status = "GHOSTED"           // No response after threshold
	StatusWithdrawn        Status = "WITHDRAWN"         // Candidate pulled out
)

// AllStatuses lists every status in canonical pipeline order. The order is
// also used as the left-to-right rank for the Sankey diagram.
var AllStatuses = []Status{
	StatusSaved,
	StatusApplied,
	StatusResponseReceived,
	StatusInterviewing,
	StatusOffer,
	StatusGhosted,
	StatusWithdrawn,
	StatusRejected,
	StatusAccepted,
}

var statusRank = func() map[Status]int {
	m := make(map[Status]int, len(AllStatuses))
	for i, s := range AllStatuses {
		m[s] = i
	}
	return m
}()

// Valid reports whether s is one of the canonical statuses.
func (s Status) Valid() bool {
	_, ok := statusRank[s]
	return ok
}

// Rank returns the canonical pipeline position of s (-1 if invalid).
func (s Status) Rank() int {
	if r, ok := statusRank[s]; ok {
		return r
	}
	return -1
}

// IsActive reports whether s represents an in-flight application (applied and
// not yet in a terminal state).
func (s Status) IsActive() bool {
	switch s {
	case StatusApplied, StatusResponseReceived, StatusInterviewing, StatusOffer:
		return true
	}
	return false
}

// Application is the core entity joined with its latest status.
type Application struct {
	ID              int64  `json:"id"`
	CompanyName     string `json:"company_name"`
	RoleTitle       string `json:"role_title"`
	Location        string `json:"location"`
	JobURL          string `json:"job_url"`
	SalaryRange     string `json:"salary_range"`
	Notes           string `json:"notes"`
	JobDescription  string `json:"job_description"`
	AppliedOn       string `json:"applied_on"`
	CreatedAt       string `json:"created_at"`
	CurrentStatus   Status `json:"current_status"`
	StatusUpdatedAt string `json:"status_updated_at"`
}

// StatusHistoryEntry is a single row of the append-only status event log.
type StatusHistoryEntry struct {
	ID            int64  `json:"id"`
	ApplicationID int64  `json:"application_id"`
	Status        Status `json:"status"`
	Notes         string `json:"notes"`
	ChangedAt     string `json:"changed_at"`
}

// ApplicationDetail is an application plus its full chronological history.
type ApplicationDetail struct {
	Application
	History []StatusHistoryEntry `json:"history"`
}

// CreateApplicationRequest is the payload for POST /api/applications.
type CreateApplicationRequest struct {
	CompanyName    string  `json:"company_name"`
	RoleTitle      string  `json:"role_title"`
	Location       string  `json:"location"`
	JobURL         string  `json:"job_url"`
	SalaryRange    string  `json:"salary_range"`
	Notes          string  `json:"notes"`
	JobDescription string  `json:"job_description"`
	InitialStatus  Status  `json:"initial_status"`
	AppliedOn      *string `json:"applied_on,omitempty"`
}

// UpdateApplicationRequest is the payload for PATCH /api/applications/{id}.
// Nil fields are left unchanged.
type UpdateApplicationRequest struct {
	CompanyName    *string `json:"company_name"`
	RoleTitle      *string `json:"role_title"`
	Location       *string `json:"location"`
	JobURL         *string `json:"job_url"`
	SalaryRange    *string `json:"salary_range"`
	Notes          *string `json:"notes"`
	JobDescription *string `json:"job_description"`
	AppliedOn      *string `json:"applied_on,omitempty"`
}

// StatusChangeRequest is the payload for POST /api/applications/{id}/status.
type StatusChangeRequest struct {
	Status    Status `json:"status"`
	Notes     string `json:"notes"`
	ChangedAt string `json:"changed_at"` // optional ISO-8601 date or datetime
}

// SankeyNode is a node in the pipeline flow diagram.
type SankeyNode struct {
	Name  Status `json:"name"`
	Value int    `json:"value,omitempty"`
}

// SankeyLink is an aggregated transition between two statuses.
type SankeyLink struct {
	Source Status `json:"source"`
	Target Status `json:"target"`
	Value  int    `json:"value"`
}

// GraphAnalytics is the payload for GET /api/analytics/graph.
type GraphAnalytics struct {
	TotalApplied        int            `json:"total_applied"`
	TotalApplications   int            `json:"total_applications"`
	RespondedCount      int            `json:"responded_count"`
	OfferCount          int            `json:"offer_count"`
	CurrentDistribution map[Status]int `json:"current_distribution"`
	ReachedCounts       map[Status]int `json:"reached_counts"`
	SankeyNodes         []SankeyNode   `json:"sankey_nodes"`
	SankeyLinks         []SankeyLink   `json:"sankey_links"`
	// OmittedBackwardLinks counts transitions that move backwards in the
	// canonical pipeline (e.g. REJECTED -> INTERVIEWING). They are excluded
	// from SankeyLinks because Sankey diagrams cannot render cycles.
	OmittedBackwardLinks int `json:"omitted_backward_links"`
}
