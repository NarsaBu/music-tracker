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
	// ListReleaseGroups returns every release group the artist is credited on (all pages),
	// with structured credits so roles can be computed.
	ListReleaseGroups(ctx context.Context, artist domain.MBID) ([]domain.ReleaseGroup, error)
}

type ArtistRepository interface {
	// Upsert stores descriptive fields; sync state (last_synced_at) is left intact.
	Upsert(ctx context.Context, a domain.Artist) error
	// Get returns domain.ErrNotFound for an unknown artist.
	Get(ctx context.Context, mbid domain.MBID) (domain.Artist, error)
	// ListTracked returns artists that have a subscription.
	ListTracked(ctx context.Context) ([]domain.MBID, error)
	MarkSynced(ctx context.Context, mbid domain.MBID, at time.Time) error
}

type ReleaseGroupRepository interface {
	KnownMBIDs(ctx context.Context, artist domain.MBID) (map[domain.MBID]bool, error)
	// UpsertMany stores release groups, refreshing title, types, date and credit of known ones.
	UpsertMany(ctx context.Context, rgs []domain.ReleaseGroup, at time.Time) error
	// LinkMany ignores links that already exist.
	LinkMany(ctx context.Context, links []domain.ArtistReleaseGroup) error
	// ListByArtist returns the artist's catalog, newest first; role "" means any role.
	ListByArtist(ctx context.Context, artist domain.MBID, role domain.CreditRole) ([]domain.ArtistReleaseGroup, error)
}

type SyncRunRepository interface {
	// Create starts a run; a run with zero artists is created already finished.
	Create(ctx context.Context, trigger domain.SyncTrigger, total int, at time.Time) (domain.SyncRun, error)
	// Running returns the unfinished run started after since, or domain.ErrNotFound.
	Running(ctx context.Context, since time.Time) (domain.SyncRun, error)
	// Latest returns the most recent run, or domain.ErrNotFound.
	Latest(ctx context.Context) (domain.SyncRun, error)
	// ArtistDone advances the run and finishes it after the last artist.
	ArtistDone(ctx context.Context, runID string, newReleases int, at time.Time) error
}

// JobQueue puts background work into the queue (ADR-5).
type JobQueue interface {
	// Enqueue reports false when a live job with the same dedup key already exists.
	Enqueue(ctx context.Context, job Job) (bool, error)
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
	// UpdateOverride replaces the filter override; domain.ErrNotFound when not subscribed.
	UpdateOverride(ctx context.Context, artist domain.MBID, o domain.FilterOverride) error
	// SubscribedAmong returns the subset of artists that have a subscription.
	SubscribedAmong(ctx context.Context, artists []domain.MBID) (map[domain.MBID]bool, error)
}

// FilterDefaultsReader reads the global release filter from settings.
type FilterDefaultsReader interface {
	FilterDefaults(ctx context.Context) (domain.ReleaseFilter, error)
}

type SettingsRepository interface {
	FilterDefaultsReader
	Get(ctx context.Context) (domain.Settings, error)
	// Save stores the user-editable fields and NextDigestAt.
	Save(ctx context.Context, s domain.Settings) error
	SetFeedToken(ctx context.Context, token string) error
	// SetDigestTimes moves the schedule; a nil last keeps the previous value.
	SetDigestTimes(ctx context.Context, last, next *time.Time) error
}

// FeedQuery selects a page of the release feed, newest first.
type FeedQuery struct {
	Before *FeedCursor
	Type   domain.PrimaryType // "" = any
	Role   domain.CreditRole  // "" = any
	Limit  int
}

// FeedCursor is the position after the last item of a page.
type FeedCursor struct {
	CreatedAt time.Time
	ID        string
}

type NotificationRepository interface {
	// Insert ignores a duplicate release group in the same channel (FR-5.8).
	Insert(ctx context.Context, n domain.Notification) error
	// ClaimPendingEmail locks pending email notifications for the current transaction.
	ClaimPendingEmail(ctx context.Context) ([]domain.NotificationItem, error)
	MarkSent(ctx context.Context, ids []string, at time.Time) error
	// MarkFailed counts a failed attempt; after domain.MaxEmailAttempts the items are failed.
	MarkFailed(ctx context.Context, ids []string, cause error) error
	ListFeed(ctx context.Context, q FeedQuery) ([]domain.NotificationItem, error)
}

// DigestMailer delivers email (docs/08-notifications.md §2.1).
type DigestMailer interface {
	SendDigest(ctx context.Context, to string, d domain.Digest) error
	SendTest(ctx context.Context, to string) error
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
