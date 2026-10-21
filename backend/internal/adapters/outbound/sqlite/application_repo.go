package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"job-tracker/backend/internal/domain"
	"job-tracker/backend/internal/ports"
)

const dbTimeLayout = "2006-01-02 15:04:05"

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
    strftime('%Y-%m-%dT%H:%M:%SZ', a.applied_on),
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

// ApplicationRepository implements ports.ApplicationRepository using SQLite.
type ApplicationRepository struct {
	db *sql.DB
}

// NewApplicationRepository creates a new SQLite ApplicationRepository.
func NewApplicationRepository(db *sql.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

// List returns applications matching the filter criteria.
func (r *ApplicationRepository) List(ctx context.Context, filter ports.ListApplicationsFilter) ([]domain.Application, error) {
	statusStr := string(filter.Status)
	search := strings.TrimSpace(filter.Search)
	pattern := "%" + escapeLike(search) + "%"

	rows, err := r.db.QueryContext(ctx, listApplicationsSQL, statusStr, search, pattern)
	if err != nil {
		return nil, fmt.Errorf("query applications: %w", err)
	}
	defer rows.Close()

	apps := make([]domain.Application, 0)
	for rows.Next() {
		app, err := scanApplication(rows)
		if err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applications: %w", err)
	}
	return apps, nil
}

// GetByID returns an application aggregate and its full status history.
func (r *ApplicationRepository) GetByID(ctx context.Context, id int64) (domain.ApplicationDetail, error) {
	return r.loadDetail(ctx, id)
}

// Create inserts a new application and its initial status history entry atomically.
func (r *ApplicationRepository) Create(ctx context.Context, app domain.Application, initialStatus domain.Status, appliedOn time.Time, changedAt time.Time) (domain.ApplicationDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ApplicationDetail{}, fmt.Errorf("begin create tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	appliedOnStr := appliedOn.UTC().Format(dbTimeLayout)
	initialChangedAtStr := changedAt.UTC().Format(dbTimeLayout)

	res, err := tx.ExecContext(ctx, `
		INSERT INTO applications (company_name, role_title, location, job_url, salary_range, notes, job_description, applied_on)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		app.CompanyName, app.RoleTitle, app.Location, app.JobURL, app.SalaryRange, app.Notes, app.JobDescription, appliedOnStr)
	if err != nil {
		return domain.ApplicationDetail{}, fmt.Errorf("insert application: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.ApplicationDetail{}, fmt.Errorf("last insert id: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO application_status_history (application_id, status, notes, changed_at)
		VALUES (?, ?, ?, ?)`,
		id, initialStatus, "Application created", initialChangedAtStr); err != nil {
		return domain.ApplicationDetail{}, fmt.Errorf("insert initial status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.ApplicationDetail{}, fmt.Errorf("commit create tx: %w", err)
	}

	return r.loadDetail(ctx, id)
}

// Update updates fields on an existing application.
func (r *ApplicationRepository) Update(ctx context.Context, id int64, cmd ports.UpdateApplicationCommand) (domain.ApplicationDetail, error) {
	type col struct {
		name string
		val  *string
	}
	cols := []col{
		{"company_name", cmd.CompanyName},
		{"role_title", cmd.RoleTitle},
		{"location", cmd.Location},
		{"job_url", cmd.JobURL},
		{"salary_range", cmd.SalaryRange},
		{"notes", cmd.Notes},
		{"job_description", cmd.JobDescription},
	}

	var formattedAppliedOn string
	if cmd.AppliedOn != nil {
		formattedAppliedOn = cmd.AppliedOn.UTC().Format(dbTimeLayout)
		cols = append(cols, col{"applied_on", &formattedAppliedOn})
	}

	sets := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols)+1)
	for _, c := range cols {
		if c.val == nil {
			continue
		}
		sets = append(sets, c.name+" = ?")
		args = append(args, *c.val)
	}

	if len(sets) == 0 {
		return domain.ApplicationDetail{}, domain.NewValidationError("no updatable fields provided")
	}

	args = append(args, id)
	query := "UPDATE applications SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return domain.ApplicationDetail{}, fmt.Errorf("update application: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ApplicationDetail{}, domain.ErrNotFound
	}

	return r.loadDetail(ctx, id)
}

// Delete removes an application and all cascading status history.
func (r *ApplicationRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM applications WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// AddStatus appends a new status history entry for an application.
func (r *ApplicationRepository) AddStatus(ctx context.Context, id int64, status domain.Status, notes string, changedAt time.Time) (domain.ApplicationDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ApplicationDetail{}, fmt.Errorf("begin status tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var currentStatus domain.Status
	if err := tx.QueryRowContext(ctx, `
		SELECT status
		FROM application_status_history
		WHERE application_id = ?
		ORDER BY changed_at DESC, id DESC
		LIMIT 1`, id).Scan(&currentStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var exists int
			if errApp := tx.QueryRowContext(ctx, "SELECT 1 FROM applications WHERE id = ?", id).Scan(&exists); errApp != nil {
				if errors.Is(errApp, sql.ErrNoRows) {
					return domain.ApplicationDetail{}, domain.ErrNotFound
				}
				return domain.ApplicationDetail{}, fmt.Errorf("check application exists: %w", errApp)
			}
			return domain.ApplicationDetail{}, domain.ErrNotFound
		}
		return domain.ApplicationDetail{}, fmt.Errorf("check current status: %w", err)
	}

	if !currentStatus.CanTransitionTo(status) {
		return domain.ApplicationDetail{}, &domain.StatusTransitionError{
			From: currentStatus,
			To:   status,
		}
	}

	changedAtStr := changedAt.UTC().Format(dbTimeLayout)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO application_status_history (application_id, status, notes, changed_at)
		VALUES (?, ?, ?, ?)`, id, status, notes, changedAtStr); err != nil {
		return domain.ApplicationDetail{}, fmt.Errorf("insert status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.ApplicationDetail{}, fmt.Errorf("commit status tx: %w", err)
	}

	return r.loadDetail(ctx, id)
}

func (r *ApplicationRepository) loadDetail(ctx context.Context, id int64) (domain.ApplicationDetail, error) {
	var d domain.ApplicationDetail
	app, err := scanApplication(r.db.QueryRowContext(ctx, getApplicationSQL, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return d, domain.ErrNotFound
		}
		return d, fmt.Errorf("load application %d: %w", id, err)
	}
	d.Application = app
	d.History = make([]domain.StatusHistoryEntry, 0)

	rows, err := r.db.QueryContext(ctx, historySQL, id)
	if err != nil {
		return d, fmt.Errorf("load history %d: %w", id, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			e         domain.StatusHistoryEntry
			changedAt string
		)
		if err := rows.Scan(&e.ID, &e.ApplicationID, &e.Status, &e.Notes, &changedAt); err != nil {
			return d, fmt.Errorf("scan history %d: %w", id, err)
		}
		e.ChangedAt = parseDBTime(changedAt)
		d.History = append(d.History, e)
	}
	if err := rows.Err(); err != nil {
		return d, fmt.Errorf("iterate history %d: %w", id, err)
	}
	return d, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanApplication(s rowScanner) (domain.Application, error) {
	var (
		a               domain.Application
		appliedOnStr    string
		createdAtStr    string
		statusUpdateStr string
	)
	err := s.Scan(
		&a.ID,
		&a.CompanyName,
		&a.RoleTitle,
		&a.Location,
		&a.JobURL,
		&a.SalaryRange,
		&a.Notes,
		&a.JobDescription,
		&appliedOnStr,
		&createdAtStr,
		&a.CurrentStatus,
		&statusUpdateStr,
	)
	if err != nil {
		return a, err
	}
	a.AppliedOn = parseDBTime(appliedOnStr)
	a.CreatedAt = parseDBTime(createdAtStr)
	a.StatusUpdatedAt = parseDBTime(statusUpdateStr)
	return a, nil
}

func parseDBTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}
