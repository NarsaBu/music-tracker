package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"music-tracker/backend/internal/usecase/artist"
)

// Use case ports of the transport (declared at the consumer, ADR-7).
type (
	ArtistSearcher interface {
		Execute(ctx context.Context, in artist.SearchInput) ([]artist.Candidate, error)
	}
	Subscriber interface {
		Execute(ctx context.Context, in artist.SubscribeInput) (artist.SubscribeResult, error)
	}
	Unsubscriber interface {
		Execute(ctx context.Context, mbid string) error
	}
	SubscriptionLister interface {
		Execute(ctx context.Context, in artist.ListInput) (artist.ListResult, error)
	}
)

// ArtistUseCases groups the catalog search and subscription operations.
type ArtistUseCases struct {
	Search      ArtistSearcher
	Subscribe   Subscriber
	Unsubscribe Unsubscriber
	List        SubscriptionLister
}

type artistHandler struct {
	uc  ArtistUseCases
	log *slog.Logger
}

// GET /catalog/artists?q=&limit=
func (h *artistHandler) search(w http.ResponseWriter, r *http.Request) {
	limit, err := queryInt(r, "limit")
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	found, err := h.uc.Search.Execute(r.Context(), artist.SearchInput{Query: r.URL.Query().Get("q"), Limit: limit})
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, itemsDTO[candidateDTO]{Items: mapSlice(found, toCandidateDTO)})
}

// GET /subscriptions?q=&limit=&offset=
func (h *artistHandler) list(w http.ResponseWriter, r *http.Request) {
	limit, err := queryInt(r, "limit")
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	offset, err := queryInt(r, "offset")
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	res, err := h.uc.List.Execute(r.Context(), artist.ListInput{
		Search: r.URL.Query().Get("q"), Limit: limit, Offset: offset,
	})
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, pageDTO[subscriptionDTO]{Items: mapSlice(res.Items, toSubscriptionDTO), Total: res.Total})
}

// POST /subscriptions → 201 when created, 200 when the artist was already subscribed.
func (h *artistHandler) subscribe(w http.ResponseWriter, r *http.Request) {
	var req subscribeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(h.log, w, r, err)
		return
	}
	res, err := h.uc.Subscribe.Execute(r.Context(), req.toInput())
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	status := http.StatusOK
	if res.Created {
		status = http.StatusCreated
	}
	writeJSON(w, status, toSubscriptionDTO(res.SubscriptionView))
}

// DELETE /subscriptions/{mbid}
func (h *artistHandler) unsubscribe(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Unsubscribe.Execute(r.Context(), chi.URLParam(r, "mbid")); err != nil {
		respondError(h.log, w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
