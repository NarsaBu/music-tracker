package http

import (
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/artist"
)

// HTTP DTOs are separate from domain entities (docs/03-backend.md §4); empty
// optional strings are sent as null.

type artistDTO struct {
	MBID           string  `json:"mbid"`
	Name           string  `json:"name"`
	SortName       string  `json:"sort_name"`
	Disambiguation string  `json:"disambiguation"`
	Type           *string `json:"type"`
	Country        *string `json:"country"`
}

type lifeSpanDTO struct {
	Begin *string `json:"begin"`
	End   *string `json:"end"`
	Ended bool    `json:"ended"`
}

type candidateDTO struct {
	artistDTO
	LifeSpan   lifeSpanDTO `json:"life_span"`
	Score      int         `json:"score"`
	Subscribed bool        `json:"subscribed"`
}

type overridableDTO[T any] struct {
	Value      T    `json:"value"`
	Overridden bool `json:"overridden"`
}

type filterDTO struct {
	PrimaryTypes           overridableDTO[[]domain.PrimaryType] `json:"primary_types"`
	ExcludedSecondaryTypes overridableDTO[[]string]             `json:"excluded_secondary_types"`
	IncludeFeatured        overridableDTO[bool]                 `json:"include_featured"`
}

type subscriptionDTO struct {
	Artist       artistDTO  `json:"artist"`
	Filter       filterDTO  `json:"filter"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
	BaselineAt   time.Time  `json:"baseline_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type itemsDTO[T any] struct {
	Items []T `json:"items"`
}

type pageDTO[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

// subscribeRequest: a missing or null override field means "use the global setting".
type subscribeRequest struct {
	MBID                   string                `json:"mbid"`
	PrimaryTypes           *[]domain.PrimaryType `json:"primary_types"`
	ExcludedSecondaryTypes *[]string             `json:"excluded_secondary_types"`
	IncludeFeatured        *bool                 `json:"include_featured"`
}

func (r subscribeRequest) toInput() artist.SubscribeInput {
	in := artist.SubscribeInput{MBID: r.MBID, Override: domain.FilterOverride{IncludeFeatured: r.IncludeFeatured}}
	if r.PrimaryTypes != nil {
		in.Override.PrimaryTypes = append([]domain.PrimaryType{}, *r.PrimaryTypes...)
	}
	if r.ExcludedSecondaryTypes != nil {
		in.Override.ExcludedSecondaryTypes = append([]string{}, *r.ExcludedSecondaryTypes...)
	}
	return in
}

func toArtistDTO(a domain.Artist) artistDTO {
	return artistDTO{
		MBID:           string(a.MBID),
		Name:           a.Name,
		SortName:       a.SortName,
		Disambiguation: a.Disambiguation,
		Type:           nullIfEmpty(a.Type),
		Country:        nullIfEmpty(a.Country),
	}
}

func toCandidateDTO(c artist.Candidate) candidateDTO {
	return candidateDTO{
		artistDTO: toArtistDTO(c.Artist),
		LifeSpan: lifeSpanDTO{
			Begin: nullIfEmpty(c.LifeSpan.Begin),
			End:   nullIfEmpty(c.LifeSpan.End),
			Ended: c.LifeSpan.Ended,
		},
		Score:      c.Score,
		Subscribed: c.Subscribed,
	}
}

func toSubscriptionDTO(v artist.SubscriptionView) subscriptionDTO {
	f := v.Filter
	return subscriptionDTO{
		Artist: toArtistDTO(v.Artist),
		Filter: filterDTO{
			PrimaryTypes:           overridableDTO[[]domain.PrimaryType](f.PrimaryTypes),
			ExcludedSecondaryTypes: overridableDTO[[]string](f.ExcludedSecondaryTypes),
			IncludeFeatured:        overridableDTO[bool](f.IncludeFeatured),
		},
		LastSyncedAt: v.Artist.LastSyncedAt,
		BaselineAt:   v.BaselineAt,
		CreatedAt:    v.CreatedAt,
	}
}

func mapSlice[T, R any](in []T, f func(T) R) []R {
	out := make([]R, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type releaseGroupDTO struct {
	MBID             string              `json:"mbid"`
	Title            string              `json:"title"`
	ArtistCredit     string              `json:"artist_credit"`
	CreditRole       domain.CreditRole   `json:"credit_role"`
	PrimaryType      *domain.PrimaryType `json:"primary_type"`
	SecondaryTypes   []string            `json:"secondary_types"`
	FirstReleaseDate *string             `json:"first_release_date"`
	IsBaseline       bool                `json:"is_baseline"`
	IsAnnouncement   bool                `json:"is_announcement"`
	CoverURL         string              `json:"cover_url"`
	MusicBrainzURL   string              `json:"musicbrainz_url"`
	DiscoveredAt     time.Time           `json:"discovered_at"`
}

// toReleaseGroupDTO needs now to tell announcements (future dates) apart.
func toReleaseGroupDTO(now time.Time) func(domain.ArtistReleaseGroup) releaseGroupDTO {
	return func(l domain.ArtistReleaseGroup) releaseGroupDTO {
		rg := l.ReleaseGroup
		var pt *domain.PrimaryType
		if rg.PrimaryType != "" {
			pt = &rg.PrimaryType
		}
		return releaseGroupDTO{
			MBID:             string(rg.MBID),
			Title:            rg.Title,
			ArtistCredit:     rg.ArtistCredit,
			CreditRole:       l.Role,
			PrimaryType:      pt,
			SecondaryTypes:   rg.SecondaryTypes,
			FirstReleaseDate: nullIfEmpty(rg.FirstRelease.Raw),
			IsBaseline:       l.IsBaseline,
			IsAnnouncement:   rg.FirstRelease.IsAnnouncement(now),
			CoverURL:         rg.CoverURL(),
			MusicBrainzURL:   rg.MusicBrainzURL(),
			DiscoveredAt:     l.DiscoveredAt,
		}
	}
}

type syncRunDTO struct {
	ID           string     `json:"id"`
	Trigger      string     `json:"trigger"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	ArtistsTotal int        `json:"artists_total"`
	ArtistsDone  int        `json:"artists_done"`
	NewReleases  int        `json:"new_releases"`
}

func toSyncRunDTO(r domain.SyncRun) syncRunDTO {
	return syncRunDTO{
		ID: r.ID, Trigger: string(r.Trigger), StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
		ArtistsTotal: r.ArtistsTotal, ArtistsDone: r.ArtistsDone, NewReleases: r.NewReleases,
	}
}
