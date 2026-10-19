// Package db owns the SQLite connection lifecycle and schema migrations.
package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // CGO-free SQLite driver, registers as "sqlite"
)

//go:embed schema.sql
var schemaSQL string

// migrations holds incremental, versioned schema changes applied after the
// baseline schema. Each entry bumps PRAGMA user_version by one. Append new
// statements here; never edit or reorder existing entries.
var migrations = []string{
	// v1: baseline schema (schema.sql) — nothing extra to apply.
	``,
	// v2: add job_description column to applications.
	`ALTER TABLE applications ADD COLUMN job_description TEXT NOT NULL DEFAULT '';`,
}

// Open opens (creating if necessary) the SQLite database at path, configured
// for WAL mode with foreign keys enforced on every pooled connection.
func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	// PRAGMAs in the DSN are executed by the driver on *every* new connection,
	// which matters because foreign_keys is a per-connection setting.
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "synchronous(NORMAL)")
	// BEGIN IMMEDIATE avoids SQLITE_BUSY deadlocks when a read tx upgrades to write.
	q.Set("_txlock", "immediate")
	dsn := "file:" + path + "?" + q.Encode()

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(4)
	conn.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return conn, nil
}

// Migrate applies the embedded baseline schema (idempotent) and then any
// pending versioned migrations, tracked via PRAGMA user_version.
func Migrate(ctx context.Context, conn *sql.DB) error {
	// The baseline contains PRAGMA journal_mode, which cannot run inside a
	// transaction, so it is executed directly. All statements are IF NOT EXISTS.
	if _, err := conn.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply baseline schema: %w", err)
	}

	var version int
	if err := conn.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	for i := version; i < len(migrations); i++ {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", i+1, err)
		}
		if migrations[i] != "" {
			apply := true
			if strings.Contains(migrations[i], "job_description") {
				exists, err := columnExists(ctx, tx, "applications", "job_description")
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("check column job_description: %w", err)
				}
				if exists {
					apply = false
				}
			}
			if apply {
				if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
					tx.Rollback()
					return fmt.Errorf("apply migration %d: %w", i+1, err)
				}
			}
		}
		// PRAGMA does not accept bound parameters; i is a trusted integer.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("bump user_version to %d: %w", i+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", i+1, err)
		}
	}

	return verify(ctx, conn)
}

func columnExists(ctx context.Context, tx *sql.Tx, table, col string) (bool, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dfltVal any
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltVal, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, col) {
			return true, nil
		}
	}
	return false, rows.Err()
}

// verify asserts that the connection-level guarantees we rely on are active.
func verify(ctx context.Context, conn *sql.DB) error {
	var fk int
	if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		return fmt.Errorf("read foreign_keys: %w", err)
	}
	if fk != 1 {
		return fmt.Errorf("foreign_keys pragma is not enabled")
	}
	var mode string
	if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		return fmt.Errorf("read journal_mode: %w", err)
	}
	if mode != "wal" {
		return fmt.Errorf("journal_mode is %q, expected wal", mode)
	}
	return nil
}
