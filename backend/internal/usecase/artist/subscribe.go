package artist

import (
	"context"
	"errors"
	"fmt"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// SubscriptionView is a subscription with its filter resolved against the global settings.
type SubscriptionView struct {
	domain.Subscription
	Filter domain.ResolvedFilter
}

type SubscribeInput struct {
	MBID     string
	Override domain.FilterOverride
}

type SubscribeResult struct {
	SubscriptionView
	// Created is false when the artist was already subscribed (the request is idempotent).
	Created bool
}

// Subscribe starts tracking an artist (US-1). The artist card is fetched from
// the catalog and stored locally; the subscription baseline is "now" (ADR-3).
type Subscribe struct {
	catalog  usecase.MusicCatalog
	artists  usecase.ArtistRepository
	subs     usecase.SubscriptionRepository
	defaults usecase.FilterDefaultsReader
	tx       usecase.TxManager
	clock    usecase.Clock
}

func NewSubscribe(
	catalog usecase.MusicCatalog,
	artists usecase.ArtistRepository,
	subs usecase.SubscriptionRepository,
	defaults usecase.FilterDefaultsReader,
	tx usecase.TxManager,
	clock usecase.Clock,
) *Subscribe {
	return &Subscribe{catalog: catalog, artists: artists, subs: subs, defaults: defaults, tx: tx, clock: clock}
}

func (uc *Subscribe) Execute(ctx context.Context, in SubscribeInput) (SubscribeResult, error) {
	mbid, err := domain.ParseMBID(in.MBID)
	if err != nil {
		return SubscribeResult{}, err
	}
	if err := in.Override.Validate(); err != nil {
		return SubscribeResult{}, err
	}

	// Already subscribed: answer from the database without calling MusicBrainz.
	sub, err := uc.subs.Get(ctx, mbid)
	switch {
	case err == nil:
		return uc.result(ctx, sub, false)
	case !errors.Is(err, domain.ErrNotFound):
		return SubscribeResult{}, fmt.Errorf("get subscription %s: %w", mbid, err)
	}

	artist, err := uc.catalog.GetArtist(ctx, mbid)
	if err != nil {
		return SubscribeResult{}, fmt.Errorf("get artist %s: %w", mbid, err)
	}

	now := uc.clock.Now()
	var created bool
	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.artists.Upsert(ctx, artist); err != nil {
			return fmt.Errorf("upsert artist: %w", err)
		}
		created, err = uc.subs.Create(ctx, domain.Subscription{
			Artist: artist, Override: in.Override, BaselineAt: now, CreatedAt: now,
		})
		if err != nil {
			return fmt.Errorf("create subscription: %w", err)
		}
		// Read back: the stored artist keeps its sync state, a concurrent request may have won.
		sub, err = uc.subs.Get(ctx, artist.MBID)
		return err
	})
	if err != nil {
		return SubscribeResult{}, fmt.Errorf("subscribe %s: %w", mbid, err)
	}
	return uc.result(ctx, sub, created)
}

func (uc *Subscribe) result(ctx context.Context, sub domain.Subscription, created bool) (SubscribeResult, error) {
	defaults, err := uc.defaults.FilterDefaults(ctx)
	if err != nil {
		return SubscribeResult{}, fmt.Errorf("read filter defaults: %w", err)
	}
	return SubscribeResult{SubscriptionView: newView(sub, defaults), Created: created}, nil
}

func newView(sub domain.Subscription, defaults domain.ReleaseFilter) SubscriptionView {
	return SubscriptionView{Subscription: sub, Filter: sub.Override.Resolve(defaults)}
}
