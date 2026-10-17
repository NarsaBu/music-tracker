package http

import (
	"context"
	"net/http"
	"time"
)

// Pinger checks that a dependency (the database) is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

type healthHandler struct {
	db Pinger
}

func newHealthHandler(db Pinger) *healthHandler { return &healthHandler{db: db} }

type healthResponse struct {
	Status string `json:"status"`
}

func (h *healthHandler) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (h *healthHandler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "Not ready", "database is unreachable")
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{Status: "ready"})
}
