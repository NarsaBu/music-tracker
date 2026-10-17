package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"

	"music-tracker/backend/internal/domain"
)

// problem is an RFC 9457 Problem Details response.
type problem struct {
	Type   string         `json:"type"`
	Title  string         `json:"title"`
	Status int            `json:"status"`
	Detail string         `json:"detail,omitempty"`
	Errors []problemField `json:"errors,omitempty"`
}

type problemField struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

const problemTypeBase = "https://music-tracker/errors/"

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, _ *http.Request, status int, title, detail string) {
	writeProblemValue(w, problem{
		Type:   problemTypeBase + slug(title),
		Title:  title,
		Status: status,
		Detail: detail,
	})
}

func writeProblemValue(w http.ResponseWriter, p problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// writeError maps domain errors to HTTP responses. It is the only place that
// knows about this mapping (see docs/06-api.md).
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *domain.ValidationError
	switch {
	case errors.As(err, &verr):
		p := problem{Type: problemTypeBase + "validation", Title: "Validation failed",
			Status: http.StatusUnprocessableEntity, Detail: err.Error()}
		for _, f := range verr.Fields {
			p.Errors = append(p.Errors, problemField(f))
		}
		writeProblemValue(w, p)
	case errors.Is(err, domain.ErrValidation):
		writeProblem(w, r, http.StatusUnprocessableEntity, "Validation failed", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "Not found", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeProblem(w, r, http.StatusConflict, "Already exists", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "")
	case errors.Is(err, domain.ErrUpstreamUnavailable):
		writeProblem(w, r, http.StatusServiceUnavailable, "Upstream unavailable", err.Error())
	default:
		// Internal details are logged by the caller, never exposed.
		writeProblem(w, r, http.StatusInternalServerError, "Internal server error", "")
	}
}

// respondError logs failures that are not the client's fault and writes the problem response.
func respondError(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	attrs := []slog.Attr{
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("error", err.Error()),
	}
	switch {
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrNotFound),
		errors.Is(err, domain.ErrAlreadyExists), errors.Is(err, domain.ErrUnauthorized):
	case errors.Is(err, domain.ErrUpstreamUnavailable), errors.Is(err, context.Canceled):
		log.LogAttrs(r.Context(), slog.LevelWarn, "request failed", attrs...)
	default:
		log.LogAttrs(r.Context(), slog.LevelError, "request failed", attrs...)
	}
	writeError(w, r, err)
}

func slug(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), " ", "-")
}
