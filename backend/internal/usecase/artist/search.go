// Package artist holds use cases around artists and subscriptions:
// SearchArtists, Subscribe, Unsubscribe, ListSubscriptions.
package artist

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

const (
	DefaultSearchLimit = 10
	MaxSearchLimit     = 25
	maxQueryLength     = 200
)

type SearchInput struct {
	Query string
	Limit int // 0 = DefaultSearchLimit
}

// Candidate is a search result marked with the subscription state.
type Candidate struct {
	domain.ArtistCandidate
	Subscribed bool
}

// SearchArtists looks artists up in the music catalog (US-1).
type SearchArtists struct {
	catalog usecase.MusicCatalog
	subs    usecase.SubscriptionRepository
}

func NewSearchArtists(catalog usecase.MusicCatalog, subs usecase.SubscriptionRepository) *SearchArtists {
	return &SearchArtists{catalog: catalog, subs: subs}
}

func (uc *SearchArtists) Execute(ctx context.Context, in SearchInput) ([]Candidate, error) {
	query := strings.TrimSpace(in.Query)
	limit := in.Limit
	if limit == 0 {
		limit = DefaultSearchLimit
	}
	var fields []domain.FieldError
	switch {
	case query == "":
		fields = append(fields, domain.FieldError{Field: "q", Message: "required"})
	case utf8.RuneCountInString(query) > maxQueryLength:
		fields = append(fields, domain.FieldError{Field: "q", Message: fmt.Sprintf("at most %d characters", maxQueryLength)})
	}
	if limit < 1 || limit > MaxSearchLimit {
		fields = append(fields, domain.FieldError{Field: "limit", Message: fmt.Sprintf("must be between 1 and %d", MaxSearchLimit)})
	}
	if len(fields) > 0 {
		return nil, &domain.ValidationError{Fields: fields}
	}

	found, err := uc.catalog.SearchArtists(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search artists %q: %w", query, err)
	}
	mbids := make([]domain.MBID, len(found))
	for i, a := range found {
		mbids[i] = a.MBID
	}
	subscribed, err := uc.subs.SubscribedAmong(ctx, mbids)
	if err != nil {
		return nil, fmt.Errorf("check subscriptions: %w", err)
	}

	out := make([]Candidate, len(found))
	for i, a := range found {
		out[i] = Candidate{ArtistCandidate: a, Subscribed: subscribed[a.MBID]}
	}
	return out, nil
}
