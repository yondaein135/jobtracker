package domain

import "time"

// Application is the core entity representing a job application with its current status.
type Application struct {
	ID              int64
	CompanyName     string
	RoleTitle       string
	Location        string
	JobURL          string
	SalaryRange     string
	Notes           string
	JobDescription  string
	AppliedOn       time.Time
	CreatedAt       time.Time
	CurrentStatus   Status
	StatusUpdatedAt time.Time
}

// StatusHistoryEntry is an immutable audit log entry of a status change.
type StatusHistoryEntry struct {
	ID            int64
	ApplicationID int64
	Status        Status
	Notes         string
	ChangedAt     time.Time
}

// ApplicationDetail is an application aggregate containing its chronological status history.
type ApplicationDetail struct {
	Application Application
	History     []StatusHistoryEntry
}
