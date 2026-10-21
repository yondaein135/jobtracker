package domain

// Status represents a canonical application pipeline status.
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
