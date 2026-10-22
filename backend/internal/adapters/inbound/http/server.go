package http

import (
	"log/slog"
	"net/http"
)

// NewRouter registers all application routes, middleware, and SPA file serving.
func NewRouter(h *Handler, allowedOrigins []string, staticDir string, logger *slog.Logger) http.Handler {
	apiMux := http.NewServeMux()

	apiMux.HandleFunc("GET /api/health", h.Health)

	apiMux.HandleFunc("GET /api/applications", h.ListApplications)
	apiMux.HandleFunc("POST /api/applications", h.CreateApplication)
	apiMux.HandleFunc("POST /api/applications/backfill", h.BackfillApplication)
	apiMux.HandleFunc("GET /api/applications/{id}", h.GetApplication)
	apiMux.HandleFunc("PATCH /api/applications/{id}", h.UpdateApplication)
	apiMux.HandleFunc("DELETE /api/applications/{id}", h.DeleteApplication)
	apiMux.HandleFunc("POST /api/applications/{id}/status", h.AddStatus)

	apiMux.HandleFunc("GET /api/analytics/graph", h.GraphAnalytics)

	// Unknown API paths get a JSON 404 rather than falling through to the SPA.
	apiMux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, r, http.StatusNotFound, "Not Found", "not found")
	})

	root := http.NewServeMux()
	root.Handle("/api/", apiMux)
	if staticDir != "" {
		root.Handle("/", SPAHandler(staticDir))
	}

	handler := SecurityHeaders(root)
	handler = CORS(allowedOrigins, handler)
	handler = StructuredLogging(logger, handler)

	return handler
}
