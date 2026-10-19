package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"job-tracker/backend/internal/models"
)

// dbTimeLayout matches SQLite's CURRENT_TIMESTAMP format (UTC), so explicitly
// supplied timestamps sort correctly against defaulted ones.
const dbTimeLayout = "2006-01-02 15:04:05"

const (
	maxBodyBytes            = 1 << 20 // 1 MiB
	maxShortField           = 300
	maxNotesLength          = 20_000
	maxJobDescriptionLength = 50_000
)

// Handler holds dependencies for the HTTP handlers.
type Handler struct {
	DB *sql.DB
}

// ---------------------------------------------------------------------------
// SQL
// ---------------------------------------------------------------------------

// latestStatusCTE resolves each application's current status from the
// append-only history log. Timestamps are normalised to ISO-8601 UTC.
const latestStatusCTE = `
WITH LatestStatus AS (
    SELECT
        application_id,
        status AS current_status,
        changed_at AS status_updated_at,
        ROW_NUMBER() OVER (
            PARTITION BY application_id
            ORDER BY changed_at DESC, id DESC
        ) AS rn
    FROM application_status_history
)
SELECT
    a.id, a.company_name, a.role_title,
    COALESCE(a.location, ''), COALESCE(a.job_url, ''),
    COALESCE(a.salary_range, ''), COALESCE(a.notes, ''),
    COALESCE(a.job_description, ''),
    strftime('%Y-%m-%dT%H:%M:%SZ', a.created_at),
    ls.current_status,
    strftime('%Y-%m-%dT%H:%M:%SZ', ls.status_updated_at)
FROM applications a
JOIN LatestStatus ls ON a.id = ls.application_id AND ls.rn = 1
`

const listApplicationsSQL = latestStatusCTE + `
WHERE (?1 = '' OR ls.current_status = ?1)
  AND (?2 = '' OR a.company_name LIKE ?3 ESCAPE '\' OR a.role_title LIKE ?3 ESCAPE '\' OR a.job_description LIKE ?3 ESCAPE '\' OR a.notes LIKE ?3 ESCAPE '\')
ORDER BY ls.status_updated_at DESC, a.id DESC;
`

const getApplicationSQL = latestStatusCTE + `WHERE a.id = ?1;`

const historySQL = `
SELECT id, application_id, status, COALESCE(notes, ''),
       strftime('%Y-%m-%dT%H:%M:%SZ', changed_at)
FROM application_status_history
WHERE application_id = ?1
ORDER BY changed_at ASC, id ASC;
`

// orderedTransitionsCTE numbers every status event per application and pairs
// it with the previous status via LAG(). first_applied marks the position of
// the first APPLIED event so the funnel only covers post-application flow.
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

// graphLinksSQL aggregates distinct-application transitions, ignoring
// self-loops and anything that happened before the first APPLIED event.
const graphLinksSQL = orderedTransitionsCTE + `
SELECT o.prev_status, o.status, COUNT(DISTINCT o.application_id) AS value
FROM ordered o
JOIN first_applied f ON f.application_id = o.application_id
WHERE o.seq > f.applied_seq
  AND o.prev_status IS NOT NULL
  AND o.prev_status <> o.status
GROUP BY o.prev_status, o.status;
`

// graphReachedSQL counts, per status, how many applied applications ever
// reached it (at or after their first APPLIED event).
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

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// ListApplications handles GET /api/applications?status=&search=.
func (h *Handler) ListApplications(w http.ResponseWriter, r *http.Request) {
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "ALL" {
		status = ""
	}
	if status != "" && !models.Status(status).Valid() {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid status %q", status))
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	pattern := "%" + escapeLike(search) + "%"

	rows, err := h.DB.QueryContext(r.Context(), listApplicationsSQL, status, search, pattern)
	if err != nil {
		internalError(w, "list applications", err)
		return
	}
	defer rows.Close()

	apps := make([]models.Application, 0)
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			internalError(w, "scan application", err)
			return
		}
		apps = append(apps, a)
	}
	if err := rows.Err(); err != nil {
		internalError(w, "iterate applications", err)
		return
	}
	writeJSON(w, http.StatusOK, apps)
}

// CreateApplication handles POST /api/applications. The application row and
// its initial status event are inserted atomically.
func (h *Handler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var req models.CreateApplicationRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.RoleTitle = strings.TrimSpace(req.RoleTitle)
	req.Location = strings.TrimSpace(req.Location)
	req.JobURL = strings.TrimSpace(req.JobURL)
	req.SalaryRange = strings.TrimSpace(req.SalaryRange)
	req.Notes = strings.TrimSpace(req.Notes)
	req.JobDescription = strings.TrimSpace(req.JobDescription)
	if req.InitialStatus == "" {
		req.InitialStatus = models.StatusApplied
	}

	if msg := validateFields(&req.CompanyName, &req.RoleTitle, &req.Location, &req.JobURL, &req.SalaryRange, &req.Notes, &req.JobDescription, true); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if req.InitialStatus != models.StatusSaved && req.InitialStatus != models.StatusApplied {
		writeError(w, http.StatusBadRequest, "initial_status must be SAVED or APPLIED")
		return
	}

	ctx := r.Context()
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		internalError(w, "begin tx", err)
		return
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	res, err := tx.ExecContext(ctx, `
		INSERT INTO applications (company_name, role_title, location, job_url, salary_range, notes, job_description)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.CompanyName, req.RoleTitle, req.Location, req.JobURL, req.SalaryRange, req.Notes, req.JobDescription)
	if err != nil {
		internalError(w, "insert application", err)
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		internalError(w, "last insert id", err)
		return
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO application_status_history (application_id, status, notes, changed_at)
		VALUES (?, ?, ?, ?)`,
		id, req.InitialStatus, "Application created", nowDB()); err != nil {
		internalError(w, "insert initial status", err)
		return
	}

	if err := tx.Commit(); err != nil {
		internalError(w, "commit", err)
		return
	}

	detail, err := h.loadDetail(ctx, id)
	if err != nil {
		internalError(w, "load created application", err)
		return
	}
	writeJSON(w, http.StatusCreated, detail)
}

// GetApplication handles GET /api/applications/{id}.
func (h *Handler) GetApplication(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	detail, err := h.loadDetail(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "application not found")
		return
	}
	if err != nil {
		internalError(w, "load application", err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// UpdateApplication handles PATCH /api/applications/{id}. Only provided
// fields are updated.
func (h *Handler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req models.UpdateApplicationRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	type col struct {
		name string
		val  *string
	}
	cols := []col{
		{"company_name", req.CompanyName},
		{"role_title", req.RoleTitle},
		{"location", req.Location},
		{"job_url", req.JobURL},
		{"salary_range", req.SalaryRange},
		{"notes", req.Notes},
		{"job_description", req.JobDescription},
	}

	sets := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols)+1)
	for _, c := range cols {
		if c.val == nil {
			continue
		}
		v := strings.TrimSpace(*c.val)
		*c.val = v
		sets = append(sets, c.name+" = ?")
		args = append(args, v)
	}
	if len(sets) == 0 {
		writeError(w, http.StatusBadRequest, "no updatable fields provided")
		return
	}

	// Validate only provided fields; required fields may not be blanked.
	if msg := validatePatch(&req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	args = append(args, id)
	// Column names come from the fixed whitelist above, never from input.
	res, err := h.DB.ExecContext(r.Context(),
		"UPDATE applications SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
	if err != nil {
		internalError(w, "update application", err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "application not found")
		return
	}

	detail, err := h.loadDetail(r.Context(), id)
	if err != nil {
		internalError(w, "load updated application", err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// AddStatus handles POST /api/applications/{id}/status, appending a new event
// to the status history log.
func (h *Handler) AddStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req models.StatusChangeRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	req.Status = models.Status(strings.ToUpper(strings.TrimSpace(string(req.Status))))
	if !req.Status.Valid() {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid status %q", req.Status))
		return
	}
	req.Notes = strings.TrimSpace(req.Notes)
	if len(req.Notes) > maxNotesLength {
		writeError(w, http.StatusBadRequest, "notes too long")
		return
	}

	changedAt := nowDB()
	if s := strings.TrimSpace(req.ChangedAt); s != "" {
		t, err := parseTimestamp(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "changed_at must be an ISO-8601 date or datetime")
			return
		}
		changedAt = t.UTC().Format(dbTimeLayout)
	}

	ctx := r.Context()
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		internalError(w, "begin tx", err)
		return
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT 1 FROM applications WHERE id = ?", id).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "application not found")
			return
		}
		internalError(w, "check application", err)
		return
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO application_status_history (application_id, status, notes, changed_at)
		VALUES (?, ?, ?, ?)`, id, req.Status, req.Notes, changedAt); err != nil {
		internalError(w, "insert status", err)
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "commit", err)
		return
	}

	detail, err := h.loadDetail(ctx, id)
	if err != nil {
		internalError(w, "load application", err)
		return
	}
	writeJSON(w, http.StatusCreated, detail)
}

// DeleteApplication handles DELETE /api/applications/{id}. History rows are
// removed via ON DELETE CASCADE.
func (h *Handler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	res, err := h.DB.ExecContext(r.Context(), "DELETE FROM applications WHERE id = ?", id)
	if err != nil {
		internalError(w, "delete application", err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "application not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GraphAnalytics handles GET /api/analytics/graph.
func (h *Handler) GraphAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	out := models.GraphAnalytics{
		CurrentDistribution: make(map[models.Status]int, len(models.AllStatuses)),
		ReachedCounts:       make(map[models.Status]int, len(models.AllStatuses)),
		SankeyNodes:         make([]models.SankeyNode, 0),
		SankeyLinks:         make([]models.SankeyLink, 0),
	}
	for _, s := range models.AllStatuses {
		out.CurrentDistribution[s] = 0
		out.ReachedCounts[s] = 0
	}

	// Current distribution across all applications.
	if err := scanStatusCounts(ctx, h.DB, currentDistributionSQL, func(s models.Status, n int) {
		out.CurrentDistribution[s] = n
		out.TotalApplications += n
	}); err != nil {
		internalError(w, "current distribution", err)
		return
	}

	// How many applied applications ever reached each stage.
	if err := scanStatusCounts(ctx, h.DB, graphReachedSQL, func(s models.Status, n int) {
		out.ReachedCounts[s] = n
	}); err != nil {
		internalError(w, "reached counts", err)
		return
	}
	out.TotalApplied = out.ReachedCounts[models.StatusApplied]

	if err := h.DB.QueryRowContext(ctx, graphRatesSQL).Scan(&out.RespondedCount, &out.OfferCount); err != nil {
		internalError(w, "rates", err)
		return
	}

	// Sankey links from LAG()-derived transitions.
	rows, err := h.DB.QueryContext(ctx, graphLinksSQL)
	if err != nil {
		internalError(w, "graph links", err)
		return
	}
	defer rows.Close()

	used := map[models.Status]bool{}
	for rows.Next() {
		var l models.SankeyLink
		if err := rows.Scan(&l.Source, &l.Target, &l.Value); err != nil {
			internalError(w, "scan link", err)
			return
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
		internalError(w, "iterate links", err)
		return
	}

	if out.TotalApplied > 0 {
		used[models.StatusApplied] = true
	}
	for _, s := range models.AllStatuses {
		if !used[s] {
			continue
		}
		n := models.SankeyNode{Name: s}
		if s == models.StatusApplied {
			// Give APPLIED its full height even if some apps never moved on.
			n.Value = out.TotalApplied
		}
		out.SankeyNodes = append(out.SankeyNodes, n)
	}

	sort.SliceStable(out.SankeyLinks, func(i, j int) bool {
		a, b := out.SankeyLinks[i], out.SankeyLinks[j]
		if a.Source.Rank() != b.Source.Rank() {
			return a.Source.Rank() < b.Source.Rank()
		}
		return a.Target.Rank() < b.Target.Rank()
	})

	writeJSON(w, http.StatusOK, out)
}

// Health handles GET /api/health.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.DB.PingContext(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type rowScanner interface {
	Scan(dest ...any) error
}

func scanApplication(s rowScanner) (models.Application, error) {
	var a models.Application
	err := s.Scan(&a.ID, &a.CompanyName, &a.RoleTitle, &a.Location, &a.JobURL,
		&a.SalaryRange, &a.Notes, &a.JobDescription, &a.CreatedAt, &a.CurrentStatus, &a.StatusUpdatedAt)
	return a, err
}

func (h *Handler) loadDetail(ctx context.Context, id int64) (models.ApplicationDetail, error) {
	var d models.ApplicationDetail
	app, err := scanApplication(h.DB.QueryRowContext(ctx, getApplicationSQL, id))
	if err != nil {
		return d, err
	}
	d.Application = app
	d.History = make([]models.StatusHistoryEntry, 0)

	rows, err := h.DB.QueryContext(ctx, historySQL, id)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var e models.StatusHistoryEntry
		if err := rows.Scan(&e.ID, &e.ApplicationID, &e.Status, &e.Notes, &e.ChangedAt); err != nil {
			return d, err
		}
		d.History = append(d.History, e)
	}
	return d, rows.Err()
}

func scanStatusCounts(ctx context.Context, db *sql.DB, query string, fn func(models.Status, int)) error {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var s models.Status
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return err
		}
		fn(s, n)
	}
	return rows.Err()
}

func validateFields(company, role, location, jobURL, salary, notes, jobDesc *string, requireCore bool) string {
	if requireCore && *company == "" {
		return "company_name is required"
	}
	if requireCore && *role == "" {
		return "role_title is required"
	}
	for name, v := range map[string]string{
		"company_name": *company, "role_title": *role,
		"location": *location, "salary_range": *salary,
	} {
		if len(v) > maxShortField {
			return name + " too long"
		}
	}
	if len(*notes) > maxNotesLength {
		return "notes too long"
	}
	if jobDesc != nil && len(*jobDesc) > maxJobDescriptionLength {
		return "job_description too long"
	}
	if msg := validateURL(*jobURL); msg != "" {
		return msg
	}
	return ""
}

func validatePatch(req *models.UpdateApplicationRequest) string {
	if req.CompanyName != nil && *req.CompanyName == "" {
		return "company_name cannot be empty"
	}
	if req.RoleTitle != nil && *req.RoleTitle == "" {
		return "role_title cannot be empty"
	}
	empty := ""
	or := func(p *string) *string {
		if p == nil {
			return &empty
		}
		return p
	}
	return validateFields(or(req.CompanyName), or(req.RoleTitle), or(req.Location),
		or(req.JobURL), or(req.SalaryRange), or(req.Notes), or(req.JobDescription), false)
}

// validateURL rejects non-http(s) schemes (e.g. javascript:) since job_url is
// rendered as a clickable link in the UI.
func validateURL(raw string) string {
	if raw == "" {
		return ""
	}
	if len(raw) > 2048 {
		return "job_url too long"
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "job_url must be a valid http(s) URL"
	}
	return ""
}

func parseTimestamp(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised timestamp %q", s)
}

func nowDB() string { return time.Now().UTC().Format(dbTimeLayout) }

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid application id")
		return 0, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		case errors.Is(err, io.EOF):
			writeError(w, http.StatusBadRequest, "request body is empty")
		default:
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		}
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func internalError(w http.ResponseWriter, op string, err error) {
	log.Printf("ERROR %s: %v", op, err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}
