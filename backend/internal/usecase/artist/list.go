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

type ListInput struct {
	Search string
	Limit  int // 0 = DefaultListLimit
	Offset int
}

type ListResult struct {
	Items []SubscriptionView
	Total int
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
	if len(fields) > 0 {
		return ListResult{}, &domain.ValidationError{Fields: fields}
	}

	subs, total, err := uc.subs.List(ctx, usecase.SubscriptionListQuery{
		Search: strings.TrimSpace(in.Search), Limit: limit, Offset: in.Offset,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("list subscriptions: %w", err)
	}
	defaults, err := uc.defaults.FilterDefaults(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("read filter defaults: %w", err)
	}

	items := make([]SubscriptionView, len(subs))
	for i, s := range subs {
		items[i] = newView(s, defaults)
	}
	return ListResult{Items: items, Total: total}, nil
}
