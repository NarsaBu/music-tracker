package http

import (
	"context"
	"log/slog"
	"net/http"

	"music-tracker/backend/internal/domain"
)

type (
	SyncStarter interface {
		Execute(ctx context.Context, trigger domain.SyncTrigger) (domain.SyncRun, bool, error)
	}
	LatestSyncRunGetter interface {
		Execute(ctx context.Context) (*domain.SyncRun, error)
	}
)

// SyncUseCases groups whole-library sync operations.
type SyncUseCases struct {
	Start  SyncStarter
	Latest LatestSyncRunGetter
}

type syncHandler struct {
	uc  SyncUseCases
	log *slog.Logger
}

// POST /sync → 202 with the run; a run already in progress is returned as is.
func (h *syncHandler) start(w http.ResponseWriter, r *http.Request) {
	run, _, err := h.uc.Start.Execute(r.Context(), domain.TriggerManual)
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, toSyncRunDTO(run))
}

// GET /sync/runs/latest → the run, or null before the first one.
func (h *syncHandler) latest(w http.ResponseWriter, r *http.Request) {
	run, err := h.uc.Latest.Execute(r.Context())
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	if run == nil {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	writeJSON(w, http.StatusOK, toSyncRunDTO(*run))
}
