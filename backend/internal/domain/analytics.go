package domain

// SankeyNode represents a node in the pipeline flow diagram.
type SankeyNode struct {
	Name  Status
	Value int
}

// SankeyLink represents an aggregated transition between two pipeline statuses.
type SankeyLink struct {
	Source Status
	Target Status
	Value  int
}

// GraphAnalytics aggregates application funnel metrics and pipeline transitions.
type GraphAnalytics struct {
	TotalApplied         int
	TotalApplications    int
	RespondedCount       int
	OfferCount           int
	CurrentDistribution  map[Status]int
	ReachedCounts        map[Status]int
	SankeyNodes          []SankeyNode
	SankeyLinks          []SankeyLink
	OmittedBackwardLinks int
}
