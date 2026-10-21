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

func TestCreateApplication_CustomAppliedOn(t *testing.T) {
	harness := setupTestHarness(t)

	customDate := "2026-10-12"
	reqBody := map[string]any{
		"company_name":   "Custom Corp",
		"role_title":     "Backend Lead",
		"initial_status": "APPLIED",
		"applied_on":     customDate,
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
	harness := setupTestHarness(t)

	// Create an app first
	createReq := map[string]any{
		"company_name":   "Update Corp",
		"role_title":     "DevOps Engineer",
		"initial_status": "APPLIED",
	}
	b, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/applications", bytes.NewReader(b))
	w := httptest.NewRecorder()
	harness.handler.CreateApplication(w, r)

	var created httpadapter.ApplicationDetailResponse
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Now patch applied_on
	newDate := "2026-09-01T10:00:00Z"
	patchReq := map[string]any{
		"applied_on": newDate,
	}
	pb, _ := json.Marshal(patchReq)
	pr := httptest.NewRequest(http.MethodPatch, "/api/applications/1", bytes.NewReader(pb))
	pr.SetPathValue("id", "1")
	pw := httptest.NewRecorder()
	harness.handler.UpdateApplication(pw, pr)

	if pw.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", pw.Code, pw.Body.String())
	}

	var updated httpadapter.ApplicationDetailResponse
	_ = json.Unmarshal(pw.Body.Bytes(), &updated)
	if !strings.HasPrefix(updated.AppliedOn, "2026-09-01") {
		t.Fatalf("expected updated applied_on to be 2026-09-01..., got %s", updated.AppliedOn)
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
