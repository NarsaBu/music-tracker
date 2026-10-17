package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"music-tracker/backend/internal/adapter/feed"
	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/notify"
)

type (
	ReleasesLister interface {
		Execute(ctx context.Context, in notify.ListReleasesInput) (notify.ReleasesPage, error)
	}
	FeedBuilder interface {
		Execute(ctx context.Context, token string) ([]domain.NotificationItem, error)
	}
)

type foundViaDTO struct {
	MBID       string            `json:"mbid"`
	Name       string            `json:"name"`
	CreditRole domain.CreditRole `json:"credit_role"`
}

// releaseDTO is a feed entry (docs/06-api.md §6).
type releaseDTO struct {
	ID               string              `json:"id"`
	MBID             string              `json:"mbid"`
	Title            string              `json:"title"`
	ArtistCredit     string              `json:"artist_credit"`
	FoundVia         foundViaDTO         `json:"found_via"`
	PrimaryType      *domain.PrimaryType `json:"primary_type"`
	SecondaryTypes   []string            `json:"secondary_types"`
	FirstReleaseDate *string             `json:"first_release_date"`
	IsAnnouncement   bool                `json:"is_announcement"`
	CoverURL         string              `json:"cover_url"`
	MusicBrainzURL   string              `json:"musicbrainz_url"`
	DiscoveredAt     time.Time           `json:"discovered_at"`
}

func toReleaseDTO(now time.Time) func(domain.NotificationItem) releaseDTO {
	return func(it domain.NotificationItem) releaseDTO {
		rg := it.ReleaseGroup
		var pt *domain.PrimaryType
		if rg.PrimaryType != "" {
			pt = &rg.PrimaryType
		}
		return releaseDTO{
			ID:               it.ID,
			MBID:             string(rg.MBID),
			Title:            rg.Title,
			ArtistCredit:     rg.ArtistCredit,
			FoundVia:         foundViaDTO{MBID: string(it.FoundVia.MBID), Name: it.FoundVia.Name, CreditRole: it.Role},
			PrimaryType:      pt,
			SecondaryTypes:   nonNilSlice(rg.SecondaryTypes),
			FirstReleaseDate: nullIfEmpty(rg.FirstRelease.Raw),
			IsAnnouncement:   rg.FirstRelease.IsAnnouncement(now),
			CoverURL:         rg.CoverURL(),
			MusicBrainzURL:   rg.MusicBrainzURL(),
			DiscoveredAt:     it.CreatedAt,
		}
	}
}

type releasesHandler struct {
	list   ReleasesLister
	feeds  FeedBuilder
	appURL string
	log    *slog.Logger
}

// GET /releases?limit=&cursor=&type=&role=
func (h *releasesHandler) listReleases(w http.ResponseWriter, r *http.Request) {
	limit, err := queryInt(r, "limit")
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	q := r.URL.Query()
	page, err := h.list.Execute(r.Context(), notify.ListReleasesInput{
		Cursor: q.Get("cursor"), Type: q.Get("type"), Role: q.Get("role"), Limit: limit,
	})
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Items      []releaseDTO `json:"items"`
		NextCursor *string      `json:"next_cursor"`
	}{mapSlice(page.Items, toReleaseDTO(time.Now().UTC())), nullIfEmpty(page.NextCursor)})
}

// GET /feeds/{token}.atom|.rss — public, protected by the token only (FR-6.3).
func (h *releasesHandler) serveFeed(w http.ResponseWriter, r *http.Request) {
	token, format, ok := feed.Parse(chi.URLParam(r, "file"))
	if !ok {
		writeProblem(w, r, http.StatusNotFound, "Not found", "")
		return
	}
	items, err := h.feeds.Execute(r.Context(), token)
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}

	etag := feed.ETag(items, format)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "max-age=900")
	if lm := feed.LastModified(items); !lm.IsZero() {
		w.Header().Set("Last-Modified", lm.UTC().Format(http.TimeFormat))
	}
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	doc, err := feed.Render(items, format, h.appURL)
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	w.Header().Set("Content-Type", format.ContentType())
	_, _ = w.Write(doc)
}
