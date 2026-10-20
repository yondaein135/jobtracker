package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"job-tracker/backend/internal/api"
	"job-tracker/backend/internal/db"
	"job-tracker/backend/internal/models"
)

func setupTestDB(t *testing.T) *api.Handler {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if err := db.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return &api.Handler{DB: conn}
}

func TestCreateApplication_DefaultAppliedOn(t *testing.T) {
	h := setupTestDB(t)

	reqBody := models.CreateApplicationRequest{
		CompanyName:   "Acme Corp",
		RoleTitle:     "Software Engineer",
		InitialStatus: models.StatusApplied,
	}
	body, _ := json.Marshal(reqBody)

	r := httptest.NewRequest(http.MethodPost, "/api/applications", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateApplication(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var detail models.ApplicationDetail
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if detail.AppliedOn == "" {
		t.Fatal("expected applied_on to be set, got empty")
	}
	if detail.CreatedAt == "" {
		t.Fatal("expected created_at to be set, got empty")
	}

	// Default applied_on should match created_at
	if detail.AppliedOn != detail.CreatedAt {
		t.Fatalf("expected applied_on (%s) to match created_at (%s)", detail.AppliedOn, detail.CreatedAt)
	}
}

func TestCreateApplication_CustomAppliedOn(t *testing.T) {
	h := setupTestDB(t)

	customDate := "2026-10-12"
	reqBody := models.CreateApplicationRequest{
		CompanyName:   "Custom Corp",
		RoleTitle:     "Backend Lead",
		InitialStatus: models.StatusApplied,
		AppliedOn:     &customDate,
	}
	body, _ := json.Marshal(reqBody)

	r := httptest.NewRequest(http.MethodPost, "/api/applications", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateApplication(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var detail models.ApplicationDetail
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !strings.HasPrefix(detail.AppliedOn, "2026-10-12") {
		t.Fatalf("expected applied_on to be 2026-10-12..., got %s", detail.AppliedOn)
	}

	// Initial status history changed_at should match applied_on
	if len(detail.History) == 0 {
		t.Fatal("expected history to have initial status entry")
	}
	if !strings.HasPrefix(detail.History[0].ChangedAt, "2026-10-12") {
		t.Fatalf("expected history changed_at to match applied_on, got %s", detail.History[0].ChangedAt)
	}
}

func TestUpdateApplication_AppliedOn(t *testing.T) {
	h := setupTestDB(t)

	// Create an app first
	createReq := models.CreateApplicationRequest{
		CompanyName:   "Update Corp",
		RoleTitle:     "DevOps Engineer",
		InitialStatus: models.StatusApplied,
	}
	b, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/applications", bytes.NewReader(b))
	w := httptest.NewRecorder()
	h.CreateApplication(w, r)

	var created models.ApplicationDetail
	json.Unmarshal(w.Body.Bytes(), &created)

	// Now patch applied_on
	newDate := "2026-09-01T10:00:00Z"
	patchReq := models.UpdateApplicationRequest{
		AppliedOn: &newDate,
	}
	pb, _ := json.Marshal(patchReq)
	pr := httptest.NewRequest(http.MethodPatch, "/api/applications/1", bytes.NewReader(pb))
	pr.SetPathValue("id", "1")
	pw := httptest.NewRecorder()
	h.UpdateApplication(pw, pr)

	if pw.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", pw.Code, pw.Body.String())
	}

	var updated models.ApplicationDetail
	json.Unmarshal(pw.Body.Bytes(), &updated)
	if !strings.HasPrefix(updated.AppliedOn, "2026-09-01") {
		t.Fatalf("expected updated applied_on to be 2026-09-01..., got %s", updated.AppliedOn)
	}
}

func TestSeedDummyData_AppliedOn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "dummy.db")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	if err := db.SeedDummyData(context.Background(), conn); err != nil {
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
	if count == 0 {
		t.Fatal("expected dummy apps to be seeded")
	}
}
