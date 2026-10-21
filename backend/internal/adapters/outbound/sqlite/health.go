package sqlite

import (
	"context"
	"database/sql"
)

// HealthChecker checks SQLite database connectivity.
type HealthChecker struct {
	db *sql.DB
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker(db *sql.DB) *HealthChecker {
	return &HealthChecker{db: db}
}

// Ping checks if the database is reachable.
func (h *HealthChecker) Ping(ctx context.Context) error {
	return h.db.PingContext(ctx)
}
