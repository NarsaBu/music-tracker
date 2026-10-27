package artist

import (
	"context"
	"errors"
	"fmt"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

const (
	// SimilarLimit is how many similar artists the artist page shows.
	SimilarLimit = 20
	// SimilarTTL: listening statistics change slowly, and ListenBrainz Labs has no SLA.
	SimilarTTL = 7 * 24 * time.Hour
)

// Similar is a similar artist marked with the subscription state.
type Similar struct {
	domain.SimilarArtist
	Subscribed bool
}

// ListSimilar returns artists similar to the given one. Answers are cached for
// SimilarTTL; when the source fails, a stale answer is better than none.
type ListSimilar struct {
	source usecase.SimilarArtistsSource
	cache  usecase.SimilarArtistsCache
	subs   usecase.SubscriptionRepository
	clock  usecase.Clock
}

func NewListSimilar(
	source usecase.SimilarArtistsSource,
	cache usecase.SimilarArtistsCache,
	subs usecase.SubscriptionRepository,
	clock usecase.Clock,
) *ListSimilar {
	return &ListSimilar{source: source, cache: cache, subs: subs, clock: clock}
}

func (uc *ListSimilar) Execute(ctx context.Context, rawMBID string) ([]Similar, error) {
	mbid, err := domain.ParseMBID(rawMBID)
	if err != nil {
		return nil, err
	}
	items, err := uc.load(ctx, mbid)
	if err != nil {
		return nil, fmt.Errorf("similar artists of %s: %w", mbid, err)
	}

	mbids := make([]domain.MBID, 0, len(items))
	for _, a := range items {
		mbids = append(mbids, a.MBID)
	}
	subscribed, err := uc.subs.SubscribedAmong(ctx, mbids)
	if err != nil {
		return nil, fmt.Errorf("check subscriptions: %w", err)
	}
	out := make([]Similar, 0, min(len(items), SimilarLimit))
	for _, a := range items {
		if a.MBID == mbid {
			continue
		}
		if len(out) == SimilarLimit {
			break
		}
		out = append(out, Similar{SimilarArtist: a, Subscribed: subscribed[a.MBID]})
	}
	return out, nil
}

func (uc *ListSimilar) load(ctx context.Context, mbid domain.MBID) ([]domain.SimilarArtist, error) {
	now := uc.clock.Now()
	cached, fetchedAt, err := uc.cache.Get(ctx, mbid)
	found := err == nil
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if found && now.Sub(fetchedAt) < SimilarTTL {
		return cached, nil
	}

	fresh, err := uc.source.SimilarArtists(ctx, mbid)
	if err != nil {
		if found {
			return cached, nil
		}
		return nil, err
	}
	if err := uc.cache.Put(ctx, mbid, fresh, now); err != nil {
		return nil, err
	}
	return fresh, nil
}
