// Package http is the HTTP transport adapter: routing, handlers, DTOs and middleware.
package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Deps are the collaborators the router needs. New handlers add their use cases here.
type Deps struct {
	Logger    *slog.Logger
	Readiness Pinger
	Artists   ArtistUseCases
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger(d.Logger))
	r.Use(recoverer(d.Logger))

	health := newHealthHandler(d.Readiness)
	r.Get("/healthz", health.live)
	r.Get("/readyz", health.ready)

	artists := &artistHandler{uc: d.Artists, log: d.Logger}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/catalog/artists", artists.search)
		r.Route("/subscriptions", func(r chi.Router) {
			r.Get("/", artists.list)
			r.Post("/", artists.subscribe)
			r.Delete("/{mbid}", artists.unsubscribe)
		})

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeProblem(w, r, http.StatusNotFound, "Not found", "")
		})
	})

	return r
}
