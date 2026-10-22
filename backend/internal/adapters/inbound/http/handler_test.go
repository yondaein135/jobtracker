package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	httpadapter "job-tracker/backend/internal/adapters/inbound/http"
	"job-tracker/backend/internal/adapters/outbound/sqlite"
	"job-tracker/backend/internal/app"
	"job-tracker/backend/internal/domain"
)

type testHarness struct {
	handler *httpadapter.Handler
	router  http.Handler
}

func setupTestHarness(t *testing.T) testHarness {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	conn, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	if err := sqlite.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	appRepo := sqlite.NewApplicationRepository(conn)
	analyticsRepo := sqlite.NewAnalyticsRepository(conn)
	healthChecker := sqlite.NewHealthChecker(conn)

	appService := app.NewApplicationService(appRepo)
	analyticsService := app.NewAnalyticsService(analyticsRepo)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := httpadapter.NewHandler(appService, analyticsService, healthChecker, logger)
	router := httpadapter.NewRouter(h, []string{"http://localhost:5173"}, "", logger)

	return testHarness{
		handler: h,
		router:  router,
	}
}

func TestCreateApplication_DefaultAppliedOn(t *testing.T) {
	harness := setupTestHarness(t)

	reqBody := map[string]any{
		"company_name":   "Acme Corp",
		"role_title":     "Software Engineer",
		"initial_status": "APPLIED",
	}
	body, _ := json.Marshal(reqBody)

	r := httptest.NewRequest(http.MethodPost, "/api/applications", bytes.NewReader(body))
	w := httptest.NewRecorder()

	harness.handler.CreateApplication(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var detail httpadapter.ApplicationDetailResponse
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

func doJSON(t *testing.T, h http.Handler, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestCreateApplication_RejectsDates(t *testing.T) {
	harness := setupTestHarness(t)

	w := doJSON(t, harness.router, http.MethodPost, "/api/applications", map[string]any{
		"company_name": "Acme", "role_title": "Engineer", "applied_on": "2026-10-12",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for applied_on on create, got %d: %s", w.Code, w.Body.String())
	}

	w = doJSON(t, harness.router, http.MethodPost, "/api/applications", map[string]any{
		"company_name": "Acme", "role_title": "Engineer",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	w = doJSON(t, harness.router, http.MethodPost, "/api/applications/1/status", map[string]any{
		"status": "INTERVIEWING", "changed_at": "2026-10-01",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for changed_at on status change, got %d: %s", w.Code, w.Body.String())
	}

	w = doJSON(t, harness.router, http.MethodPatch, "/api/applications/1", map[string]any{
		"applied_on": "2026-09-01",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for applied_on on update, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBackfillApplication(t *testing.T) {
	harness := setupTestHarness(t)

	w := doJSON(t, harness.router, http.MethodPost, "/api/applications/backfill", map[string]any{
		"company_name": "Past Corp",
		"role_title":   "Backend Lead",
		"history": []map[string]any{
			{"status": "SAVED", "changed_at": "2026-08-30"},
			{"status": "APPLIED", "changed_at": "2026-09-01"},
			{"status": "INTERVIEWING", "changed_at": "2026-09-10T14:00:00Z", "notes": "Tech round"},
			{"status": "REJECTED", "changed_at": "2026-09-20"},
		},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var detail httpadapter.ApplicationDetailResponse
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if detail.CurrentStatus != domain.StatusRejected {
		t.Errorf("expected current status REJECTED, got %s", detail.CurrentStatus)
	}
	if !strings.HasPrefix(detail.AppliedOn, "2026-09-01") {
		t.Errorf("expected applied_on from the APPLIED step, got %s", detail.AppliedOn)
	}
	if !strings.HasPrefix(detail.StatusUpdatedAt, "2026-09-20") {
		t.Errorf("expected status_updated_at from the last step, got %s", detail.StatusUpdatedAt)
	}
	wantOrder := []domain.Status{domain.StatusSaved, domain.StatusApplied, domain.StatusInterviewing, domain.StatusRejected}
	if len(detail.History) != len(wantOrder) {
		t.Fatalf("expected %d history entries, got %d", len(wantOrder), len(detail.History))
	}
	for i, s := range wantOrder {
		if detail.History[i].Status != s {
			t.Errorf("history[%d]: expected %s, got %s", i, s, detail.History[i].Status)
		}
	}
	if detail.History[2].Notes != "Tech round" {
		t.Errorf("expected note on interview step, got %q", detail.History[2].Notes)
	}
}

func TestBackfillApplication_Invalid(t *testing.T) {
	harness := setupTestHarness(t)
	base := func(history []map[string]any) map[string]any {
		return map[string]any{"company_name": "Acme", "role_title": "Engineer", "history": history}
	}

	cases := []struct {
		name    string
		history []map[string]any
	}{
		{"empty history", []map[string]any{}},
		{"bad date", []map[string]any{{"status": "APPLIED", "changed_at": "last tuesday"}}},
		{"missing date", []map[string]any{{"status": "APPLIED"}}},
		{"future date", []map[string]any{{"status": "APPLIED", "changed_at": "2999-01-01"}}},
		{"starts mid-pipeline", []map[string]any{{"status": "OFFER", "changed_at": "2026-09-01"}}},
		{"backward step", []map[string]any{
			{"status": "APPLIED", "changed_at": "2026-09-01"},
			{"status": "OFFER", "changed_at": "2026-09-05"},
			{"status": "INTERVIEWING", "changed_at": "2026-09-06"},
		}},
		{"dates out of order", []map[string]any{
			{"status": "APPLIED", "changed_at": "2026-09-05"},
			{"status": "REJECTED", "changed_at": "2026-09-01"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(t, harness.router, http.MethodPost, "/api/applications/backfill", base(tc.history))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}

	// Nothing invalid should have been written.
	w := doJSON(t, harness.router, http.MethodGet, "/api/applications", nil)
	if strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("expected no applications to be stored, got %s", w.Body.String())
	}
}

func TestApplication_FullLifecycle(t *testing.T) {
	harness := setupTestHarness(t)

	// 1. Create Application
	createPayload := map[string]any{
		"company_name":   "Google",
		"role_title":     "Staff SWE",
		"location":       "Zurich, Switzerland",
		"salary_range":   "CHF 250'000",
		"initial_status": "SAVED",
	}
	b, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/applications", bytes.NewReader(b))
	rec := httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}
	var created httpadapter.ApplicationDetailResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.CurrentStatus != domain.StatusSaved {
		t.Fatalf("expected status SAVED, got %s", created.CurrentStatus)
	}

	// 2. Add Status
	statusPayload := map[string]any{
		"status": "APPLIED",
		"notes":  "Submitted resume online",
	}
	b, _ = json.Marshal(statusPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/applications/1/status", bytes.NewReader(b))
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on status add, got %d: %s", rec.Code, rec.Body.String())
	}
	var updated httpadapter.ApplicationDetailResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated.CurrentStatus != domain.StatusApplied {
		t.Fatalf("expected status APPLIED, got %s", updated.CurrentStatus)
	}
	if len(updated.History) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(updated.History))
	}

	// 3. Get Application
	req = httptest.NewRequest(http.MethodGet, "/api/applications/1", nil)
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on get, got %d", rec.Code)
	}

	// 4. List Applications
	req = httptest.NewRequest(http.MethodGet, "/api/applications?search=Google", nil)
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on list, got %d", rec.Code)
	}
	var list []httpadapter.ApplicationResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 || list[0].CompanyName != "Google" {
		t.Fatalf("expected 1 result with company Google, got %v", list)
	}

	// 5. Analytics Graph
	req = httptest.NewRequest(http.MethodGet, "/api/analytics/graph", nil)
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on analytics, got %d", rec.Code)
	}
	var analytics httpadapter.GraphAnalyticsResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &analytics)
	if analytics.TotalApplications != 1 {
		t.Fatalf("expected 1 total application in analytics, got %d", analytics.TotalApplications)
	}

	// 6. Delete Application
	req = httptest.NewRequest(http.MethodDelete, "/api/applications/1", nil)
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content on delete, got %d", rec.Code)
	}

	// 7. Verify Not Found after deletion
	req = httptest.NewRequest(http.MethodGet, "/api/applications/1", nil)
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found after delete, got %d", rec.Code)
	}
}

func TestHealth(t *testing.T) {
	harness := setupTestHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for health, got %d", rec.Code)
	}
}

func TestStatusTransitions_DirectionalFlow(t *testing.T) {
	harness := setupTestHarness(t)

	// Create application at SAVED
	createPayload := map[string]any{
		"company_name":   "Flow Corp",
		"role_title":     "Platform Engineer",
		"initial_status": "SAVED",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/applications", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", rec.Body.String())
	}

	var created httpadapter.ApplicationDetailResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	appID := created.ID

	// 1. Invalid: Cannot jump from SAVED directly to INTERVIEWING
	badReq := map[string]any{"status": "INTERVIEWING"}
	b, _ := json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/applications/1/status", bytes.NewReader(b))
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on SAVED -> INTERVIEWING, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Valid: SAVED -> APPLIED
	validReq := map[string]any{"status": "APPLIED"}
	b, _ = json.Marshal(validReq)
	req = httptest.NewRequest(http.MethodPost, "/api/applications/1/status", bytes.NewReader(b))
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on SAVED -> APPLIED, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Invalid: APPLIED -> SAVED (backward transition)
	badReq = map[string]any{"status": "SAVED"}
	b, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/applications/1/status", bytes.NewReader(b))
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on backward APPLIED -> SAVED, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify RFC 9457 Problem Details JSON format
	var prob httpadapter.ProblemDetails
	_ = json.Unmarshal(rec.Body.Bytes(), &prob)
	if !strings.Contains(prob.Title, "Invalid Status Transition") {
		t.Errorf("expected Title to contain 'Invalid Status Transition', got %q", prob.Title)
	}

	// 4. Valid skip: APPLIED -> INTERVIEWING (skipping response)
	validReq = map[string]any{"status": "INTERVIEWING"}
	b, _ = json.Marshal(validReq)
	req = httptest.NewRequest(http.MethodPost, "/api/applications/1/status", bytes.NewReader(b))
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on APPLIED -> INTERVIEWING, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Invalid: INTERVIEWING -> RESPONSE_RECEIVED (backward)
	badReq = map[string]any{"status": "RESPONSE_RECEIVED"}
	b, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/applications/1/status", bytes.NewReader(b))
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on backward INTERVIEWING -> RESPONSE_RECEIVED, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Valid terminal outcome: INTERVIEWING -> REJECTED
	validReq = map[string]any{"status": "REJECTED"}
	b, _ = json.Marshal(validReq)
	req = httptest.NewRequest(http.MethodPost, "/api/applications/1/status", bytes.NewReader(b))
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on INTERVIEWING -> REJECTED, got %d: %s", rec.Code, rec.Body.String())
	}

	// 7. Invalid: REJECTED is terminal, cannot transition to anything
	badReq = map[string]any{"status": "OFFER"}
	b, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/applications/1/status", bytes.NewReader(b))
	rec = httptest.NewRecorder()
	harness.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on transition from terminal REJECTED, got %d: %s", rec.Code, rec.Body.String())
	}

	_ = appID
}
