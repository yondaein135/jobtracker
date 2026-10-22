package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"job-tracker/backend/internal/domain"
	"job-tracker/backend/internal/ports"
)

// Handler handles HTTP requests by coordinating inbound ports.
type Handler struct {
	appService       ports.ApplicationService
	analyticsService ports.AnalyticsService
	healthChecker    ports.HealthChecker
	logger           *slog.Logger
}

// NewHandler creates a new Handler with injected ports.
func NewHandler(
	appService ports.ApplicationService,
	analyticsService ports.AnalyticsService,
	healthChecker ports.HealthChecker,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		appService:       appService,
		analyticsService: analyticsService,
		healthChecker:    healthChecker,
		logger:           logger,
	}
}

// ListApplications handles GET /api/applications.
func (h *Handler) ListApplications(w http.ResponseWriter, r *http.Request) {
	statusStr := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if statusStr == "ALL" {
		statusStr = ""
	}

	search := strings.TrimSpace(r.URL.Query().Get("search"))
	filter := ports.ListApplicationsFilter{
		Status: domain.Status(statusStr),
		Search: search,
	}

	apps, err := h.appService.List(r.Context(), filter)
	if err != nil {
		handleDomainError(w, r, h.logger, err)
		return
	}

	res := make([]ApplicationResponse, 0, len(apps))
	for _, a := range apps {
		res = append(res, toApplicationResponse(a))
	}

	writeJSON(w, http.StatusOK, res)
}

// CreateApplication handles POST /api/applications.
func (h *Handler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var req CreateApplicationRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	cmd := ports.CreateApplicationCommand{
		CompanyName:    req.CompanyName,
		RoleTitle:      req.RoleTitle,
		Location:       req.Location,
		JobURL:         req.JobURL,
		SalaryRange:    req.SalaryRange,
		Notes:          req.Notes,
		JobDescription: req.JobDescription,
		InitialStatus:  domain.Status(strings.ToUpper(strings.TrimSpace(req.InitialStatus))),
	}

	detail, err := h.appService.Create(r.Context(), cmd)
	if err != nil {
		handleDomainError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusCreated, toApplicationDetailResponse(detail))
}

// BackfillApplication handles POST /api/applications/backfill.
func (h *Handler) BackfillApplication(w http.ResponseWriter, r *http.Request) {
	var req BackfillApplicationRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	history := make([]ports.BackfillEvent, 0, len(req.History))
	for i, e := range req.History {
		t, err := parseTimestamp(e.ChangedAt)
		if err != nil {
			writeProblem(w, r, http.StatusBadRequest, "Bad Request",
				fmt.Sprintf("history[%d].changed_at must be an ISO-8601 date or datetime", i))
			return
		}
		history = append(history, ports.BackfillEvent{
			Status:    domain.Status(e.Status),
			Notes:     e.Notes,
			ChangedAt: t,
		})
	}

	cmd := ports.BackfillApplicationCommand{
		CompanyName:    req.CompanyName,
		RoleTitle:      req.RoleTitle,
		Location:       req.Location,
		JobURL:         req.JobURL,
		SalaryRange:    req.SalaryRange,
		Notes:          req.Notes,
		JobDescription: req.JobDescription,
		History:        history,
	}

	detail, err := h.appService.Backfill(r.Context(), cmd)
	if err != nil {
		handleDomainError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusCreated, toApplicationDetailResponse(detail))
}

// GetApplication handles GET /api/applications/{id}.
func (h *Handler) GetApplication(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parsePathID(w, r)
	if !ok {
		return
	}

	detail, err := h.appService.GetByID(r.Context(), id)
	if err != nil {
		handleDomainError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, toApplicationDetailResponse(detail))
}

// UpdateApplication handles PATCH /api/applications/{id}.
func (h *Handler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parsePathID(w, r)
	if !ok {
		return
	}

	var req UpdateApplicationRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	cmd := ports.UpdateApplicationCommand{
		CompanyName:    req.CompanyName,
		RoleTitle:      req.RoleTitle,
		Location:       req.Location,
		JobURL:         req.JobURL,
		SalaryRange:    req.SalaryRange,
		Notes:          req.Notes,
		JobDescription: req.JobDescription,
	}

	detail, err := h.appService.Update(r.Context(), id, cmd)
	if err != nil {
		handleDomainError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, toApplicationDetailResponse(detail))
}

// DeleteApplication handles DELETE /api/applications/{id}.
func (h *Handler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parsePathID(w, r)
	if !ok {
		return
	}

	if err := h.appService.Delete(r.Context(), id); err != nil {
		handleDomainError(w, r, h.logger, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddStatus handles POST /api/applications/{id}/status.
func (h *Handler) AddStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parsePathID(w, r)
	if !ok {
		return
	}

	var req StatusChangeRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	cmd := ports.AddStatusCommand{
		Status: domain.Status(strings.ToUpper(strings.TrimSpace(req.Status))),
		Notes:  req.Notes,
	}

	detail, err := h.appService.AddStatus(r.Context(), id, cmd)
	if err != nil {
		handleDomainError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusCreated, toApplicationDetailResponse(detail))
}

// GraphAnalytics handles GET /api/analytics/graph.
func (h *Handler) GraphAnalytics(w http.ResponseWriter, r *http.Request) {
	data, err := h.analyticsService.GetGraphAnalytics(r.Context())
	if err != nil {
		handleDomainError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, toGraphAnalyticsResponse(data))
}

// Health handles GET /api/health.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if h.healthChecker != nil {
		if err := h.healthChecker.Ping(r.Context()); err != nil {
			writeProblem(w, r, http.StatusServiceUnavailable, "Service Unavailable", "database unavailable")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) parsePathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid application id")
		return 0, false
	}
	return id, true
}
