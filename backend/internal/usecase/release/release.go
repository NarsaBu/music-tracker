// Package release holds use cases of the release page: GetReleaseGroup, ListReviews, GetDetails.
package release

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// ReviewsTTL: reviews of new releases come in over days, a daily refresh is enough.
const ReviewsTTL = 24 * time.Hour

// GetReleaseGroup returns a stored release group with its MusicBrainz rating.
type GetReleaseGroup struct {
	releases usecase.ReleaseGroupRepository
}

func NewGetReleaseGroup(releases usecase.ReleaseGroupRepository) *GetReleaseGroup {
	return &GetReleaseGroup{releases: releases}
}

func (uc *GetReleaseGroup) Execute(ctx context.Context, rawMBID string) (domain.ReleaseGroup, error) {
	mbid, err := domain.ParseMBID(rawMBID)
	if err != nil {
		return domain.ReleaseGroup{}, err
	}
	rg, err := uc.releases.Get(ctx, mbid)
	if err != nil {
		return domain.ReleaseGroup{}, fmt.Errorf("release group %s: %w", mbid, err)
	}
	return rg, nil
}

// ListReviews returns user reviews of a stored release group. Answers are cached
// for ReviewsTTL; when the source fails, a stale answer is better than none.
type ListReviews struct {
	releases usecase.ReleaseGroupRepository
	source   usecase.ReviewSource
	cache    usecase.ReviewCache
	clock    usecase.Clock
}

func NewListReviews(
	releases usecase.ReleaseGroupRepository,
	source usecase.ReviewSource,
	cache usecase.ReviewCache,
	clock usecase.Clock,
) *ListReviews {
	return &ListReviews{releases: releases, source: source, cache: cache, clock: clock}
}

func (uc *ListReviews) Execute(ctx context.Context, rawMBID string) (domain.Reviews, error) {
	mbid, err := domain.ParseMBID(rawMBID)
	if err != nil {
		return domain.Reviews{}, err
	}
	// Only releases the tracker knows: the cache is keyed by stored release groups.
	if _, err := uc.releases.Get(ctx, mbid); err != nil {
		return domain.Reviews{}, fmt.Errorf("reviews of %s: %w", mbid, err)
	}

	reviews, err := loadCached(ctx, mbid, uc.cache, uc.source.ReleaseGroupReviews, uc.clock.Now(), ReviewsTTL)
	if err != nil {
		return domain.Reviews{}, fmt.Errorf("reviews of %s: %w", mbid, err)
	}
	return reviews, nil
}

// DetailsTTL: tracklists and genres change rarely once a release is out.
const DetailsTTL = 7 * 24 * time.Hour

// DetailsView is the release page details with the subscription state of every
// credited artist, so the page links to the artists the tracker knows.
type DetailsView struct {
	domain.ReleaseDetails
	Subscribed map[domain.MBID]bool
}

// GetDetails returns genres, the tracklist and credited artists of a stored release
// group, cached for DetailsTTL; when MusicBrainz fails, a stale answer is better than none.
type GetDetails struct {
	releases usecase.ReleaseGroupRepository
	catalog  usecase.ReleaseCatalog
	cache    usecase.ReleaseDetailsCache
	subs     usecase.SubscriptionRepository
	clock    usecase.Clock
}

func NewGetDetails(
	releases usecase.ReleaseGroupRepository,
	catalog usecase.ReleaseCatalog,
	cache usecase.ReleaseDetailsCache,
	subs usecase.SubscriptionRepository,
	clock usecase.Clock,
) *GetDetails {
	return &GetDetails{releases: releases, catalog: catalog, cache: cache, subs: subs, clock: clock}
}

func (uc *GetDetails) Execute(ctx context.Context, rawMBID string) (DetailsView, error) {
	mbid, err := domain.ParseMBID(rawMBID)
	if err != nil {
		return DetailsView{}, err
	}
	if _, err := uc.releases.Get(ctx, mbid); err != nil {
		return DetailsView{}, fmt.Errorf("release details of %s: %w", mbid, err)
	}
	d, err := loadCached(ctx, mbid, uc.cache, uc.catalog.ReleaseGroupDetails, uc.clock.Now(), DetailsTTL)
	if err != nil {
		return DetailsView{}, fmt.Errorf("release details of %s: %w", mbid, err)
	}

	var artists []domain.MBID
	for _, a := range append(slices.Clone(d.Artists), d.Guests...) {
		artists = append(artists, a.ArtistMBID)
	}
	subscribed, err := uc.subs.SubscribedAmong(ctx, artists)
	if err != nil {
		return DetailsView{}, fmt.Errorf("check subscriptions: %w", err)
	}
	return DetailsView{ReleaseDetails: d, Subscribed: subscribed}, nil
}

// cache is a per-release-group answer store with the time of the answer.
type cache[T any] interface {
	Get(ctx context.Context, rg domain.MBID) (T, time.Time, error)
	Put(ctx context.Context, rg domain.MBID, v T, at time.Time) error
}

// loadCached serves a fresh cached answer, otherwise fetches and stores a new one;
// when the fetch fails, a stale answer is returned instead of the error.
func loadCached[T any](
	ctx context.Context,
	rg domain.MBID,
	c cache[T],
	fetch func(context.Context, domain.MBID) (T, error),
	now time.Time,
	ttl time.Duration,
) (T, error) {
	var zero T
	cached, fetchedAt, err := c.Get(ctx, rg)
	found := err == nil
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return zero, err
	}
	if found && now.Sub(fetchedAt) < ttl {
		return cached, nil
	}
	fresh, err := fetch(ctx, rg)
	if err != nil {
		if found {
			return cached, nil
		}
		return zero, err
	}
	if err := c.Put(ctx, rg, fresh, now); err != nil {
		return zero, err
	}
	return fresh, nil
}
