package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"job-tracker/backend/internal/domain"
)

const orderedTransitionsCTE = `
WITH ordered AS (
    SELECT
        application_id,
        status,
        LAG(status) OVER w AS prev_status,
        ROW_NUMBER() OVER w AS seq
    FROM application_status_history
    WINDOW w AS (PARTITION BY application_id ORDER BY changed_at ASC, id ASC)
),
first_applied AS (
    SELECT application_id, MIN(seq) AS applied_seq
    FROM ordered
    WHERE status = 'APPLIED'
    GROUP BY application_id
)
`

const graphLinksSQL = orderedTransitionsCTE + `
SELECT o.prev_status, o.status, COUNT(DISTINCT o.application_id) AS value
FROM ordered o
JOIN first_applied f ON f.application_id = o.application_id
WHERE o.seq > f.applied_seq
  AND o.prev_status IS NOT NULL
  AND o.prev_status <> o.status
GROUP BY o.prev_status, o.status;
`

const graphReachedSQL = orderedTransitionsCTE + `
SELECT o.status, COUNT(DISTINCT o.application_id)
FROM ordered o
JOIN first_applied f ON f.application_id = o.application_id
WHERE o.seq >= f.applied_seq
GROUP BY o.status;
`

const graphRatesSQL = orderedTransitionsCTE + `
SELECT
    COUNT(DISTINCT CASE WHEN o.status IN ('RESPONSE_RECEIVED','INTERVIEWING','OFFER','ACCEPTED')
                        THEN o.application_id END),
    COUNT(DISTINCT CASE WHEN o.status IN ('OFFER','ACCEPTED')
                        THEN o.application_id END)
FROM ordered o
JOIN first_applied f ON f.application_id = o.application_id
WHERE o.seq >= f.applied_seq;
`

const currentDistributionSQL = `
WITH LatestStatus AS (
    SELECT status,
           ROW_NUMBER() OVER (
               PARTITION BY application_id
               ORDER BY changed_at DESC, id DESC
           ) AS rn
    FROM application_status_history
)
SELECT status, COUNT(*) FROM LatestStatus WHERE rn = 1 GROUP BY status;
`

// AnalyticsRepository implements ports.AnalyticsRepository using SQLite.
type AnalyticsRepository struct {
	db *sql.DB
}

// NewAnalyticsRepository creates a new SQLite AnalyticsRepository.
func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// GetGraphAnalytics queries pipeline statistics and computes Sankey flow links.
func (r *AnalyticsRepository) GetGraphAnalytics(ctx context.Context) (domain.GraphAnalytics, error) {
	out := domain.GraphAnalytics{
		CurrentDistribution: make(map[domain.Status]int, len(domain.AllStatuses)),
		ReachedCounts:       make(map[domain.Status]int, len(domain.AllStatuses)),
		SankeyNodes:         make([]domain.SankeyNode, 0),
		SankeyLinks:         make([]domain.SankeyLink, 0),
	}
	for _, s := range domain.AllStatuses {
		out.CurrentDistribution[s] = 0
		out.ReachedCounts[s] = 0
	}

	// Current distribution across all applications.
	if err := scanStatusCounts(ctx, r.db, currentDistributionSQL, func(s domain.Status, n int) {
		out.CurrentDistribution[s] = n
		out.TotalApplications += n
	}); err != nil {
		return out, fmt.Errorf("current distribution: %w", err)
	}

	// How many applied applications ever reached each stage.
	if err := scanStatusCounts(ctx, r.db, graphReachedSQL, func(s domain.Status, n int) {
		out.ReachedCounts[s] = n
	}); err != nil {
		return out, fmt.Errorf("reached counts: %w", err)
	}
	out.TotalApplied = out.ReachedCounts[domain.StatusApplied]

	if err := r.db.QueryRowContext(ctx, graphRatesSQL).Scan(&out.RespondedCount, &out.OfferCount); err != nil {
		return out, fmt.Errorf("rates: %w", err)
	}

	// Sankey links from LAG()-derived transitions.
	rows, err := r.db.QueryContext(ctx, graphLinksSQL)
	if err != nil {
		return out, fmt.Errorf("graph links: %w", err)
	}
	defer rows.Close()

	used := map[domain.Status]bool{}
	for rows.Next() {
		var l domain.SankeyLink
		if err := rows.Scan(&l.Source, &l.Target, &l.Value); err != nil {
			return out, fmt.Errorf("scan link: %w", err)
		}
		// Sankey diagrams must be acyclic: only keep edges that move forward in
		// the canonical pipeline order.
		if l.Source.Rank() >= l.Target.Rank() {
			out.OmittedBackwardLinks += l.Value
			continue
		}
		used[l.Source], used[l.Target] = true, true
		out.SankeyLinks = append(out.SankeyLinks, l)
	}
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("iterate links: %w", err)
	}

	if out.TotalApplied > 0 {
		used[domain.StatusApplied] = true
	}
	for _, s := range domain.AllStatuses {
		if !used[s] {
			continue
		}
		n := domain.SankeyNode{Name: s}
		if s == domain.StatusApplied {
			// Give APPLIED its full height even if some apps never moved on.
			n.Value = out.TotalApplied
		}
		out.SankeyNodes = append(out.SankeyNodes, n)
	}

	slices.SortStableFunc(out.SankeyLinks, func(a, b domain.SankeyLink) int {
		if diff := a.Source.Rank() - b.Source.Rank(); diff != 0 {
			return diff
		}
		return a.Target.Rank() - b.Target.Rank()
	})

	return out, nil
}

func scanStatusCounts(ctx context.Context, db *sql.DB, query string, fn func(domain.Status, int)) error {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var s domain.Status
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return err
		}
		fn(s, n)
	}
	return rows.Err()
}
