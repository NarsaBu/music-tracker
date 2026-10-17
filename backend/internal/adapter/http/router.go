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
	Sync      SyncUseCases
	Settings  SettingsUseCases
	Releases  ReleasesLister
	Feeds     FeedBuilder
	Imports   ImportUseCases
	// AppURL is the public address of the app, linked from feeds.
	AppURL string
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
	sync := &syncHandler{uc: d.Sync, log: d.Logger}
	settings := &settingsHandler{uc: d.Settings, log: d.Logger}
	imports := &importHandler{uc: d.Imports, log: d.Logger}
	releases := &releasesHandler{list: d.Releases, feeds: d.Feeds, appURL: d.AppURL, log: d.Logger}

	r.Get("/feeds/{file}", releases.serveFeed)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/catalog/artists", artists.search)
		r.Route("/subscriptions", func(r chi.Router) {
			r.Get("/", artists.list)
			r.Post("/", artists.subscribe)
			r.Get("/{mbid}", artists.get)
			r.Patch("/{mbid}", artists.update)
			r.Delete("/{mbid}", artists.unsubscribe)
			r.Post("/{mbid}/sync", artists.sync)
		})
		r.Get("/artists/{mbid}/release-groups", artists.releaseGroups)
		r.Post("/sync", sync.start)
		r.Get("/sync/runs/latest", sync.latest)
		r.Get("/releases", releases.listReleases)
		r.Route("/imports", func(r chi.Router) {
			r.Get("/", imports.list)
			r.Post("/", imports.create)
			r.Get("/{id}", imports.get)
			r.Get("/{id}/items", imports.items)
			r.Post("/{id}/items/{itemId}/resolve", imports.resolve)
		})
		r.Get("/settings", settings.get)
		r.Put("/settings", settings.put)
		r.Post("/settings/email/test", settings.testEmail)
		r.Post("/settings/feed-token/rotate", settings.rotate)
		r.Get("/meta/release-types", releaseTypes)

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeProblem(w, r, http.StatusNotFound, "Not found", "")
		})
	})

	return r
}
