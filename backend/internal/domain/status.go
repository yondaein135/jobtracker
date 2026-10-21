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

// IsTerminal reports whether s is a final outcome status from which no further transitions are allowed.
func (s Status) IsTerminal() bool {
	switch s {
	case StatusAccepted, StatusRejected, StatusGhosted, StatusWithdrawn:
		return true
	}
	return false
}

// allowedForwardTransitions defines the unidirectional DAG of permitted status changes.
// - SAVED can only transition forward to APPLIED.
// - APPLIED, RESPONSE_RECEIVED, INTERVIEWING, OFFER can transition forward to downstream stages or terminal outcomes.
// - Terminal statuses (ACCEPTED, REJECTED, GHOSTED, WITHDRAWN) have no outgoing transitions.
var allowedForwardTransitions = map[Status][]Status{
	StatusSaved: {
		StatusApplied,
	},
	StatusApplied: {
		StatusResponseReceived,
		StatusInterviewing,
		StatusOffer,
		StatusAccepted,
		StatusRejected,
		StatusGhosted,
		StatusWithdrawn,
	},
	StatusResponseReceived: {
		StatusInterviewing,
		StatusOffer,
		StatusAccepted,
		StatusRejected,
		StatusGhosted,
		StatusWithdrawn,
	},
	StatusInterviewing: {
		StatusOffer,
		StatusAccepted,
		StatusRejected,
		StatusGhosted,
		StatusWithdrawn,
	},
	StatusOffer: {
		StatusAccepted,
		StatusRejected,
		StatusGhosted,
		StatusWithdrawn,
	},
	StatusAccepted:  {},
	StatusRejected:  {},
	StatusGhosted:   {},
	StatusWithdrawn: {},
}

// NextAllowedStatuses returns a slice of valid target statuses reachable from s.
func (s Status) NextAllowedStatuses() []Status {
	targets, ok := allowedForwardTransitions[s]
	if !ok || len(targets) == 0 {
		return nil
	}
	res := make([]Status, len(targets))
	copy(res, targets)
	return res
}

// CanTransitionTo reports whether an application at status s is allowed to transition to target.
func (s Status) CanTransitionTo(target Status) bool {
	if !target.Valid() || s == target {
		return false
	}
	targets, ok := allowedForwardTransitions[s]
	if !ok {
		return false
	}
	for _, allowed := range targets {
		if allowed == target {
			return true
		}
	}
	return false
}
