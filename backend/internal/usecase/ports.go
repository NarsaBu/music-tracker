// Package usecase contains application business operations. Ports (interfaces
// to the outside world) are declared here, next to their consumers; adapters
// implement them.
package usecase

import (
	"context"
	"time"

	"music-tracker/backend/internal/domain"
)

// MusicCatalog is the external music database (MusicBrainz).
type MusicCatalog interface {
	SearchArtists(ctx context.Context, query string, limit int) ([]domain.ArtistCandidate, error)
	// GetArtist returns domain.ErrNotFound for an unknown MBID.
	GetArtist(ctx context.Context, mbid domain.MBID) (domain.Artist, error)
}

type ArtistRepository interface {
	// Upsert stores descriptive fields; sync state (last_synced_at) is left intact.
	Upsert(ctx context.Context, a domain.Artist) error
}

// SubscriptionListQuery selects a page of subscriptions; Search matches artist names.
type SubscriptionListQuery struct {
	Search string
	Limit  int
	Offset int
}

type SubscriptionRepository interface {
	// Create reports false if a subscription for the artist already exists.
	Create(ctx context.Context, s domain.Subscription) (created bool, err error)
	// Get returns domain.ErrNotFound when the artist is not subscribed.
	Get(ctx context.Context, artist domain.MBID) (domain.Subscription, error)
	// Delete returns domain.ErrNotFound when the artist is not subscribed.
	Delete(ctx context.Context, artist domain.MBID) error
	List(ctx context.Context, q SubscriptionListQuery) (items []domain.Subscription, total int, err error)
	// SubscribedAmong returns the subset of artists that have a subscription.
	SubscribedAmong(ctx context.Context, artists []domain.MBID) (map[domain.MBID]bool, error)
}

// FilterDefaultsReader reads the global release filter from settings.
type FilterDefaultsReader interface {
	FilterDefaults(ctx context.Context) (domain.ReleaseFilter, error)
}

// TxManager runs fn inside a single database transaction. Repositories pick the
// transaction up from ctx, so use cases stay unaware of the storage details.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Clock abstracts time for deterministic tests.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production Clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
