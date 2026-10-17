package artist

import (
	"context"
	"fmt"
	"slices"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// GetSubscription returns one subscription with its resolved filter.
type GetSubscription struct {
	subs     usecase.SubscriptionRepository
	defaults usecase.FilterDefaultsReader
}

func NewGetSubscription(subs usecase.SubscriptionRepository, defaults usecase.FilterDefaultsReader) *GetSubscription {
	return &GetSubscription{subs: subs, defaults: defaults}
}

func (uc *GetSubscription) Execute(ctx context.Context, rawMBID string) (SubscriptionView, error) {
	mbid, err := domain.ParseMBID(rawMBID)
	if err != nil {
		return SubscriptionView{}, err
	}
	sub, err := uc.subs.Get(ctx, mbid)
	if err != nil {
		return SubscriptionView{}, fmt.Errorf("get subscription %s: %w", mbid, err)
	}
	defaults, err := uc.defaults.FilterDefaults(ctx)
	if err != nil {
		return SubscriptionView{}, fmt.Errorf("read filter defaults: %w", err)
	}
	return newView(sub, defaults), nil
}

// ListReleaseGroups returns an artist's stored catalog, including featured releases.
type ListReleaseGroups struct {
	artists  usecase.ArtistRepository
	releases usecase.ReleaseGroupRepository
}

func NewListReleaseGroups(artists usecase.ArtistRepository, releases usecase.ReleaseGroupRepository) *ListReleaseGroups {
	return &ListReleaseGroups{artists: artists, releases: releases}
}

// Execute filters by role when role is not empty.
func (uc *ListReleaseGroups) Execute(ctx context.Context, rawMBID, role string) ([]domain.ArtistReleaseGroup, error) {
	mbid, err := domain.ParseMBID(rawMBID)
	if err != nil {
		return nil, err
	}
	r := domain.CreditRole(role)
	if r != "" && !slices.Contains(domain.CreditRoles, r) {
		return nil, &domain.ValidationError{Fields: []domain.FieldError{{Field: "role", Message: fmt.Sprintf("unknown value %q", role)}}}
	}
	if _, err := uc.artists.Get(ctx, mbid); err != nil {
		return nil, fmt.Errorf("artist %s: %w", mbid, err)
	}
	items, err := uc.releases.ListByArtist(ctx, mbid, r)
	if err != nil {
		return nil, fmt.Errorf("list release groups of %s: %w", mbid, err)
	}
	return items, nil
}
