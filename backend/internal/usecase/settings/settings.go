// Package settings reads and changes the instance settings: Get, Update, RotateFeedToken.
package settings

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

type Config struct {
	// PublicBaseURL prefixes links handed to the user (feed URL).
	PublicBaseURL string
}

// View is the settings as shown to the user.
type View struct {
	domain.Settings
	FeedURL string // Atom; the RSS 2.0 variant ends with .rss
}

type Get struct {
	repo usecase.SettingsRepository
	cfg  Config
}

func NewGet(repo usecase.SettingsRepository, cfg Config) *Get { return &Get{repo: repo, cfg: cfg} }

func (uc *Get) Execute(ctx context.Context) (View, error) {
	s, err := uc.repo.Get(ctx)
	if err != nil {
		return View{}, fmt.Errorf("get settings: %w", err)
	}
	return View{Settings: s, FeedURL: feedURL(uc.cfg, s.RSS.FeedToken)}, nil
}

// UpdateInput holds the user-editable settings.
type UpdateInput struct {
	Filter              domain.ReleaseFilter
	NotifyAnnouncements bool
	NewReleaseDays      int
	Email               domain.EmailSettings
	RSSEnabled          bool
}

// Update validates and stores settings. The digest is rescheduled when the
// schedule changes or email is turned on; turning email off clears it.
type Update struct {
	repo  usecase.SettingsRepository
	clock usecase.Clock
	cfg   Config
}

func NewUpdate(repo usecase.SettingsRepository, clock usecase.Clock, cfg Config) *Update {
	return &Update{repo: repo, clock: clock, cfg: cfg}
}

func (uc *Update) Execute(ctx context.Context, in UpdateInput) (View, error) {
	cur, err := uc.repo.Get(ctx)
	if err != nil {
		return View{}, fmt.Errorf("update settings: %w", err)
	}
	next := cur
	next.Filter = in.Filter
	next.NotifyAnnouncements = in.NotifyAnnouncements
	next.NewReleaseDays = in.NewReleaseDays
	next.Email = in.Email
	next.Email.To = strings.TrimSpace(next.Email.To)
	next.RSS.Enabled = in.RSSEnabled
	if err := next.Validate(); err != nil {
		return View{}, err
	}

	switch {
	case !next.EmailActive():
		next.NextDigestAt = nil
	case !cur.EmailActive() || cur.NextDigestAt == nil || cur.Email.Digest != next.Email.Digest:
		at, err := next.Email.Digest.Next(uc.clock.Now())
		if err != nil {
			return View{}, err
		}
		next.NextDigestAt = &at
	}
	if err := uc.repo.Save(ctx, next); err != nil {
		return View{}, fmt.Errorf("update settings: %w", err)
	}
	return View{Settings: next, FeedURL: feedURL(uc.cfg, next.RSS.FeedToken)}, nil
}

// RotateFeedToken issues a new feed token; the old link stops working.
type RotateFeedToken struct {
	repo usecase.SettingsRepository
	cfg  Config
}

func NewRotateFeedToken(repo usecase.SettingsRepository, cfg Config) *RotateFeedToken {
	return &RotateFeedToken{repo: repo, cfg: cfg}
}

// Execute returns the new feed URL.
func (uc *RotateFeedToken) Execute(ctx context.Context) (string, error) {
	b := make([]byte, 32) // 256 bits, NFR-5 asks for ≥ 128
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rotate feed token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	if err := uc.repo.SetFeedToken(ctx, token); err != nil {
		return "", fmt.Errorf("rotate feed token: %w", err)
	}
	return feedURL(uc.cfg, token), nil
}

func feedURL(cfg Config, token string) string {
	return strings.TrimRight(cfg.PublicBaseURL, "/") + "/feeds/" + token + ".atom"
}
