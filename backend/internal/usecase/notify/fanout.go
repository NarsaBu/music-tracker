// Package notify turns discovered releases into notifications and delivers them:
// FanOut, ScheduleDigest, SendDigest, SendTestEmail (docs/08-notifications.md).
package notify

import (
	"context"
	"errors"
	"fmt"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// FanOut creates notifications for releases newly found for a subscribed artist.
// It runs inside the sync transaction, so discovery and outbox commit together (ADR-4).
type FanOut struct {
	settings      usecase.SettingsRepository
	subs          usecase.SubscriptionRepository
	notifications usecase.NotificationRepository
	clock         usecase.Clock
}

func NewFanOut(
	settings usecase.SettingsRepository,
	subs usecase.SubscriptionRepository,
	notifications usecase.NotificationRepository,
	clock usecase.Clock,
) *FanOut {
	return &FanOut{settings: settings, subs: subs, notifications: notifications, clock: clock}
}

// Execute returns how many release groups passed the filters. links must be
// fresh discoveries (news, not stale); unsubscribed artists notify nothing.
func (uc *FanOut) Execute(ctx context.Context, artist domain.MBID, links []domain.ArtistReleaseGroup) (int, error) {
	if len(links) == 0 {
		return 0, nil
	}
	sub, err := uc.subs.Get(ctx, artist)
	if errors.Is(err, domain.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("fan out %s: %w", artist, err)
	}
	s, err := uc.settings.Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("fan out %s: %w", artist, err)
	}
	filter := sub.Override.Resolve(s.Filter).Effective()
	now := uc.clock.Now()

	matched := 0
	for _, l := range links {
		rg := l.ReleaseGroup
		if !filter.Matches(rg, l.Role) {
			continue
		}
		if rg.FirstRelease.IsAnnouncement(now) && !s.NotifyAnnouncements {
			continue
		}
		matched++
		base := domain.Notification{ReleaseGroup: rg.MBID, Artist: artist, Role: l.Role, At: now}

		// The rss record is the feed itself: delivered on creation, always written.
		rss := base
		rss.Channel, rss.Status = domain.ChannelRSS, domain.StatusSent
		if err := uc.notifications.Insert(ctx, rss); err != nil {
			return 0, fmt.Errorf("fan out %s: %w", artist, err)
		}
		if s.EmailActive() {
			email := base
			email.Channel, email.Status = domain.ChannelEmail, domain.StatusPending
			if err := uc.notifications.Insert(ctx, email); err != nil {
				return 0, fmt.Errorf("fan out %s: %w", artist, err)
			}
		}
	}
	return matched, nil
}
