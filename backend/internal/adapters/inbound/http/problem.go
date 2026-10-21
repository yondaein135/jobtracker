package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"job-tracker/backend/internal/domain"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// ProblemDetails implements RFC 9457 Problem Details for HTTP APIs.
// The Error field provides backward compatibility with clients expecting { "error": "..." }.
type ProblemDetails struct {
	Type     string `json:"type,omitempty"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Error    string `json:"error"`
	Instance string `json:"instance,omitempty"`
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	prob := ProblemDetails{
		Title:    title,
		Status:   status,
		Detail:   detail,
		Error:    detail,
		Instance: r.URL.Path,
	}

	_ = json.NewEncoder(w).Encode(prob)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			writeProblem(w, r, http.StatusRequestEntityTooLarge, "Payload Too Large", "request body too large")
		case errors.Is(err, io.EOF):
			writeProblem(w, r, http.StatusBadRequest, "Bad Request", "request body is empty")
		default:
			writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid JSON: "+err.Error())
		}
		return false
	}
	return true
}

func handleDomainError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	if err == nil {
		return
	}

	var valErr *domain.ValidationError
	if errors.As(err, &valErr) {
		writeProblem(w, r, http.StatusBadRequest, "Validation Error", valErr.Message)
		return
	}

	if errors.Is(err, domain.ErrInvalidInput) {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	if errors.Is(err, domain.ErrNotFound) {
		writeProblem(w, r, http.StatusNotFound, "Not Found", "application not found")
		return
	}

	logger.ErrorContext(r.Context(), "internal server error",
		"path", r.URL.Path,
		"method", r.Method,
		"error", err,
	)
	writeProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "internal server error")
}

func parseTimestamp(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, strings.TrimSpace(s)); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised timestamp %q", s)
}
