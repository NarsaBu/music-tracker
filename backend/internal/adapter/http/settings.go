package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"music-tracker/backend/internal/domain"
	settingsuc "music-tracker/backend/internal/usecase/settings"
)

type (
	SettingsGetter interface {
		Execute(ctx context.Context) (settingsuc.View, error)
	}
	SettingsUpdater interface {
		Execute(ctx context.Context, in settingsuc.UpdateInput) (settingsuc.View, error)
	}
	FeedTokenRotator interface {
		Execute(ctx context.Context) (string, error)
	}
	TestEmailSender interface {
		Execute(ctx context.Context) error
	}
)

type SettingsUseCases struct {
	Get       SettingsGetter
	Update    SettingsUpdater
	Rotate    FeedTokenRotator
	TestEmail TestEmailSender
}

// settingsDTO is both the GET response and the PUT body; read-only fields
// (feed_url, last/next digest) are ignored on input.
type settingsDTO struct {
	Filters filtersDTO       `json:"filters"`
	Email   emailSettingsDTO `json:"email"`
	RSS     rssSettingsDTO   `json:"rss"`
}

type filtersDTO struct {
	PrimaryTypes           []domain.PrimaryType `json:"primary_types"`
	ExcludedSecondaryTypes []string             `json:"excluded_secondary_types"`
	IncludeFeatured        bool                 `json:"include_featured"`
	NotifyAnnouncements    bool                 `json:"notify_announcements"`
}

type emailSettingsDTO struct {
	Enabled      bool       `json:"enabled"`
	To           string     `json:"to"`
	Digest       digestDTO  `json:"digest"`
	LastDigestAt *time.Time `json:"last_digest_at"`
	NextDigestAt *time.Time `json:"next_digest_at"`
}

type digestDTO struct {
	Frequency domain.DigestFrequency `json:"frequency"`
	Weekday   int                    `json:"weekday"` // 0 = Sunday
	Time      string                 `json:"time"`    // HH:MM
	Timezone  string                 `json:"timezone"`
}

type rssSettingsDTO struct {
	Enabled bool   `json:"enabled"`
	FeedURL string `json:"feed_url"`
}

func toSettingsDTO(v settingsuc.View) settingsDTO {
	d := v.Email.Digest
	return settingsDTO{
		Filters: filtersDTO{
			PrimaryTypes:           nonNilSlice(v.Filter.PrimaryTypes),
			ExcludedSecondaryTypes: nonNilSlice(v.Filter.ExcludedSecondaryTypes),
			IncludeFeatured:        v.Filter.IncludeFeatured,
			NotifyAnnouncements:    v.NotifyAnnouncements,
		},
		Email: emailSettingsDTO{
			Enabled: v.Email.Enabled,
			To:      v.Email.To,
			Digest: digestDTO{
				Frequency: d.Frequency,
				Weekday:   int(d.Weekday),
				Time:      time.Date(0, 1, 1, d.Hour, d.Minute, 0, 0, time.UTC).Format("15:04"),
				Timezone:  d.Timezone,
			},
			LastDigestAt: v.LastDigestAt,
			NextDigestAt: v.NextDigestAt,
		},
		RSS: rssSettingsDTO{Enabled: v.RSS.Enabled, FeedURL: v.FeedURL},
	}
}

func (d settingsDTO) toInput() (settingsuc.UpdateInput, error) {
	var fields []domain.FieldError
	if d.Filters.PrimaryTypes == nil {
		fields = append(fields, domain.FieldError{Field: "filters.primary_types", Message: "required"})
	}
	at, err := time.Parse("15:04", d.Email.Digest.Time)
	if err != nil {
		fields = append(fields, domain.FieldError{Field: "email.digest.time", Message: "must be HH:MM"})
	}
	if len(fields) > 0 {
		return settingsuc.UpdateInput{}, &domain.ValidationError{Fields: fields}
	}
	return settingsuc.UpdateInput{
		Filter: domain.ReleaseFilter{
			PrimaryTypes:           d.Filters.PrimaryTypes,
			ExcludedSecondaryTypes: nonNilSlice(d.Filters.ExcludedSecondaryTypes),
			IncludeFeatured:        d.Filters.IncludeFeatured,
		},
		NotifyAnnouncements: d.Filters.NotifyAnnouncements,
		Email: domain.EmailSettings{
			Enabled: d.Email.Enabled,
			To:      d.Email.To,
			Digest: domain.DigestSchedule{
				Frequency: d.Email.Digest.Frequency,
				Weekday:   time.Weekday(d.Email.Digest.Weekday),
				Hour:      at.Hour(),
				Minute:    at.Minute(),
				Timezone:  d.Email.Digest.Timezone,
			},
		},
		RSSEnabled: d.RSS.Enabled,
	}, nil
}

type settingsHandler struct {
	uc  SettingsUseCases
	log *slog.Logger
}

// GET /settings
func (h *settingsHandler) get(w http.ResponseWriter, r *http.Request) {
	v, err := h.uc.Get.Execute(r.Context())
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSettingsDTO(v))
}

// PUT /settings
func (h *settingsHandler) put(w http.ResponseWriter, r *http.Request) {
	var req settingsDTO
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(h.log, w, r, err)
		return
	}
	in, err := req.toInput()
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	v, err := h.uc.Update.Execute(r.Context(), in)
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSettingsDTO(v))
}

// POST /settings/email/test → 204
func (h *settingsHandler) testEmail(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.TestEmail.Execute(r.Context()); err != nil {
		respondError(h.log, w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /settings/feed-token/rotate → {feed_url}
func (h *settingsHandler) rotate(w http.ResponseWriter, r *http.Request) {
	url, err := h.uc.Rotate.Execute(r.Context())
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		FeedURL string `json:"feed_url"`
	}{url})
}

// GET /meta/release-types
func releaseTypes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, struct {
		PrimaryTypes   []domain.PrimaryType `json:"primary_types"`
		SecondaryTypes []string             `json:"secondary_types"`
	}{domain.PrimaryTypes, domain.SecondaryTypes})
}

func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
