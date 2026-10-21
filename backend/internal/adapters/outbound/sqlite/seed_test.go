package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"job-tracker/backend/internal/adapters/outbound/sqlite"
)

func TestSeedDummyData_AppliedOn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "dummy.db")
	conn, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := sqlite.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	if err := sqlite.SeedDummyData(context.Background(), conn); err != nil {
		t.Fatalf("seed dummy data: %v", err)
	}

	rows, err := conn.Query("SELECT company_name, applied_on, created_at FROM applications")
	if err != nil {
		t.Fatalf("query applications: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var company, appliedOn, createdAt string
		if err := rows.Scan(&company, &appliedOn, &createdAt); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		if appliedOn == "" {
			t.Fatalf("app %s has empty applied_on", company)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate rows: %v", err)
	}
	if count == 0 {
		t.Fatal("expected dummy apps to be seeded")
	}
}
