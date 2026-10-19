package api

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"
)

// NewRouter registers all /api routes using Go 1.22+ method/path patterns and
// wraps them in logging and CORS middleware.
func NewRouter(db *sql.DB, allowedOrigins []string) http.Handler {
	h := &Handler{DB: db}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.Health)

	mux.HandleFunc("GET /api/applications", h.ListApplications)
	mux.HandleFunc("POST /api/applications", h.CreateApplication)
	mux.HandleFunc("GET /api/applications/{id}", h.GetApplication)
	mux.HandleFunc("PATCH /api/applications/{id}", h.UpdateApplication)
	mux.HandleFunc("DELETE /api/applications/{id}", h.DeleteApplication)
	mux.HandleFunc("POST /api/applications/{id}/status", h.AddStatus)

	mux.HandleFunc("GET /api/analytics/graph", h.GraphAnalytics)

	// Unknown API paths get a JSON 404 rather than falling through to the SPA.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})

	return logRequests(cors(allowedOrigins, mux))
}

// cors allows cross-origin calls from an explicit allowlist only (used by the
// Vite dev server). In production the SPA is same-origin and needs no CORS.
func cors(allowed []string, next http.Handler) http.Handler {
	allow := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		if o = strings.TrimSpace(o); o != "" {
			allow[o] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allow[origin] {
			hdr := w.Header()
			hdr.Set("Access-Control-Allow-Origin", origin)
			hdr.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			hdr.Set("Access-Control-Allow-Headers", "Content-Type")
			hdr.Set("Access-Control-Max-Age", "600")
			hdr.Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Microsecond))
	})
}
