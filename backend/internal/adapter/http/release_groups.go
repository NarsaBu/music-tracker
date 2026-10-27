package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/release"
)

type (
	ReleaseGroupGetter interface {
		Execute(ctx context.Context, mbid string) (domain.ReleaseGroup, error)
	}
	ReviewLister interface {
		Execute(ctx context.Context, mbid string) (domain.Reviews, error)
	}
	ReleaseDetailsGetter interface {
		Execute(ctx context.Context, mbid string) (release.DetailsView, error)
	}
)

// ReleaseGroupUseCases back the release page.
type ReleaseGroupUseCases struct {
	Get     ReleaseGroupGetter
	Reviews ReviewLister
	Details ReleaseDetailsGetter
}

type releaseGroupHandler struct {
	uc  ReleaseGroupUseCases
	log *slog.Logger
}

type releaseDetailDTO struct {
	MBID             string              `json:"mbid"`
	Title            string              `json:"title"`
	ArtistCredit     string              `json:"artist_credit"`
	PrimaryType      *domain.PrimaryType `json:"primary_type"`
	SecondaryTypes   []string            `json:"secondary_types"`
	FirstReleaseDate *string             `json:"first_release_date"`
	IsAnnouncement   bool                `json:"is_announcement"`
	CoverURL         string              `json:"cover_url"`
	MusicBrainzURL   string              `json:"musicbrainz_url"`
	Rating           *ratingDTO          `json:"rating"`
}

type reviewDTO struct {
	ID          string     `json:"id"`
	Author      string     `json:"author"`
	Rating      *int       `json:"rating"` // 1–5
	Text        string     `json:"text"`
	Language    string     `json:"language"`
	PublishedAt *time.Time `json:"published_at"`
	License     string     `json:"license"`
	URL         string     `json:"url"`
}

type reviewsDTO struct {
	Items []reviewDTO `json:"items"`
	Total int         `json:"total"`
	// Average is the mean of the rated reviews; null when none is rated.
	Average *ratingDTO `json:"average"`
	URL     string     `json:"url"`
}

// GET /release-groups/{mbid} — a release the tracker knows, with its MusicBrainz rating.
func (h *releaseGroupHandler) get(w http.ResponseWriter, r *http.Request) {
	rg, err := h.uc.Get.Execute(r.Context(), chi.URLParam(r, "mbid"))
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	var pt *domain.PrimaryType
	if rg.PrimaryType != "" {
		pt = &rg.PrimaryType
	}
	writeJSON(w, http.StatusOK, releaseDetailDTO{
		MBID:             string(rg.MBID),
		Title:            rg.Title,
		ArtistCredit:     rg.ArtistCredit,
		PrimaryType:      pt,
		SecondaryTypes:   nonNilSlice(rg.SecondaryTypes),
		FirstReleaseDate: nullIfEmpty(rg.FirstRelease.Raw),
		IsAnnouncement:   rg.FirstRelease.IsAnnouncement(time.Now().UTC()),
		CoverURL:         rg.CoverURL(),
		MusicBrainzURL:   rg.MusicBrainzURL(),
		Rating:           toRatingDTO(rg.Rating),
	})
}

// GET /release-groups/{mbid}/reviews — CritiqueBrainz, 503 when it is down and nothing is cached.
func (h *releaseGroupHandler) reviews(w http.ResponseWriter, r *http.Request) {
	mbid := chi.URLParam(r, "mbid")
	rv, err := h.uc.Reviews.Execute(r.Context(), mbid)
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	out := reviewsDTO{
		Items: make([]reviewDTO, len(rv.Items)),
		Total: rv.Total,
		URL:   "https://critiquebrainz.org/release-group/" + mbid,
	}
	if rv.Rated > 0 {
		out.Average = &ratingDTO{Value: rv.Average, Votes: rv.Rated}
	}
	for i, it := range rv.Items {
		d := reviewDTO{
			ID: it.ID, Author: it.Author, Text: it.Text, Language: it.Language, License: it.License,
			URL: "https://critiquebrainz.org/review/" + it.ID,
		}
		if it.Rating > 0 {
			d.Rating = &it.Rating
		}
		if !it.PublishedAt.IsZero() {
			d.PublishedAt = &it.PublishedAt
		}
		out.Items[i] = d
	}
	writeJSON(w, http.StatusOK, out)
}

type creditedArtistDTO struct {
	MBID       string `json:"mbid"`
	Name       string `json:"name"`
	Subscribed bool   `json:"subscribed"`
}

type trackDTO struct {
	Number       string `json:"number"`
	Title        string `json:"title"`
	LengthMS     *int64 `json:"length_ms"`
	ArtistCredit string `json:"artist_credit"`
}

type mediumDTO struct {
	Position int        `json:"position"`
	Format   string     `json:"format"`
	Title    string     `json:"title"`
	Tracks   []trackDTO `json:"tracks"`
}

type editionDTO struct {
	MBID           string  `json:"mbid"`
	Title          string  `json:"title"`
	Date           *string `json:"date"`
	Country        string  `json:"country"`
	Status         string  `json:"status"`
	MusicBrainzURL string  `json:"musicbrainz_url"`
}

type releaseDetailsDTO struct {
	Genres []string `json:"genres"`
	// Edition is null when the release group has no release yet (an announcement).
	Edition *editionDTO         `json:"edition"`
	Media   []mediumDTO         `json:"media"`
	Artists []creditedArtistDTO `json:"artists"`
	Guests  []creditedArtistDTO `json:"guests"`
}

// GET /release-groups/{mbid}/details — genres, tracklist of the earliest official
// edition and credited artists; 503 when MusicBrainz is down and nothing is cached.
func (h *releaseGroupHandler) details(w http.ResponseWriter, r *http.Request) {
	v, err := h.uc.Details.Execute(r.Context(), chi.URLParam(r, "mbid"))
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	credited := func(names []domain.CreditName) []creditedArtistDTO {
		out := make([]creditedArtistDTO, len(names))
		for i, n := range names {
			out[i] = creditedArtistDTO{MBID: string(n.ArtistMBID), Name: n.Name, Subscribed: v.Subscribed[n.ArtistMBID]}
		}
		return out
	}
	out := releaseDetailsDTO{
		Genres:  nonNilSlice(v.Genres),
		Media:   make([]mediumDTO, len(v.Media)),
		Artists: credited(v.Artists),
		Guests:  credited(v.Guests),
	}
	if e := v.Edition; e.MBID != "" {
		out.Edition = &editionDTO{
			MBID: string(e.MBID), Title: e.Title, Date: nullIfEmpty(e.Date.Raw), Country: e.Country, Status: e.Status,
			MusicBrainzURL: "https://musicbrainz.org/release/" + string(e.MBID),
		}
	}
	for i, m := range v.Media {
		md := mediumDTO{Position: m.Position, Format: m.Format, Title: m.Title, Tracks: make([]trackDTO, len(m.Tracks))}
		for j, t := range m.Tracks {
			td := trackDTO{Number: t.Number, Title: t.Title, ArtistCredit: t.ArtistCredit}
			if t.Length > 0 {
				ms := t.Length.Milliseconds()
				td.LengthMS = &ms
			}
			md.Tracks[j] = td
		}
		out.Media[i] = md
	}
	writeJSON(w, http.StatusOK, out)
}
