package artist

import (
	"context"
	"fmt"
	"strings"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

const (
	DefaultListLimit = 50
	MaxListLimit     = 200
)

// List filter values; an empty string means "any".
const (
	StatusSynced  = "synced"
	StatusPending = "pending"
	FilterOwn     = "own"
	FilterGlobal  = "global"
	SortName      = "name"
	SortAdded     = "added"
)

type ListInput struct {
	Search string
	// Status: StatusSynced or StatusPending.
	Status     string
	ArtistType string
	Country    string
	// Filter: FilterOwn (has a per-artist override) or FilterGlobal.
	Filter string
	// Sort: SortName (default) or SortAdded (newest subscriptions first).
	Sort   string
	Limit  int // 0 = DefaultListLimit
	Offset int
}

type ListResult struct {
	Items []SubscriptionView
	Total int
	// Facets list the countries and artist types to offer as filters.
	Facets usecase.SubscriptionFacets
}

// optionalFlag maps one of two allowed values to *bool; "" is nil.
func optionalFlag(v, yes, no, field string) (*bool, *domain.FieldError) {
	switch v {
	case "":
		return nil, nil
	case yes, no:
		b := v == yes
		return &b, nil
	}
	return nil, &domain.FieldError{Field: field, Message: fmt.Sprintf("must be %q or %q", yes, no)}
}

// ListSubscriptions returns subscriptions ordered by artist sort name.
type ListSubscriptions struct {
	subs     usecase.SubscriptionRepository
	defaults usecase.FilterDefaultsReader
}

func NewListSubscriptions(subs usecase.SubscriptionRepository, defaults usecase.FilterDefaultsReader) *ListSubscriptions {
	return &ListSubscriptions{subs: subs, defaults: defaults}
}

func (uc *ListSubscriptions) Execute(ctx context.Context, in ListInput) (ListResult, error) {
	limit := in.Limit
	if limit == 0 {
		limit = DefaultListLimit
	}
	var fields []domain.FieldError
	if limit < 1 || limit > MaxListLimit {
		fields = append(fields, domain.FieldError{Field: "limit", Message: fmt.Sprintf("must be between 1 and %d", MaxListLimit)})
	}
	if in.Offset < 0 {
		fields = append(fields, domain.FieldError{Field: "offset", Message: "must not be negative"})
	}
	synced, ferr := optionalFlag(in.Status, StatusSynced, StatusPending, "status")
	if ferr != nil {
		fields = append(fields, *ferr)
	}
	overridden, ferr := optionalFlag(in.Filter, FilterOwn, FilterGlobal, "filter")
	if ferr != nil {
		fields = append(fields, *ferr)
	}
	if in.Sort != "" && in.Sort != SortName && in.Sort != SortAdded {
		fields = append(fields, domain.FieldError{Field: "sort", Message: fmt.Sprintf("must be %q or %q", SortName, SortAdded)})
	}
	if len(fields) > 0 {
		return ListResult{}, &domain.ValidationError{Fields: fields}
	}

	subs, total, err := uc.subs.List(ctx, usecase.SubscriptionListQuery{
		Search:      strings.TrimSpace(in.Search),
		Synced:      synced,
		ArtistType:  strings.TrimSpace(in.ArtistType),
		Country:     strings.TrimSpace(in.Country),
		Overridden:  overridden,
		NewestFirst: in.Sort == SortAdded,
		Limit:       limit,
		Offset:      in.Offset,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("list subscriptions: %w", err)
	}
	facets, err := uc.subs.Facets(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("subscription facets: %w", err)
	}
	defaults, err := uc.defaults.FilterDefaults(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("read filter defaults: %w", err)
	}

	items := make([]SubscriptionView, len(subs))
	for i, s := range subs {
		items[i] = newView(s, defaults)
	}
	return ListResult{Items: items, Total: total, Facets: facets}, nil
}
