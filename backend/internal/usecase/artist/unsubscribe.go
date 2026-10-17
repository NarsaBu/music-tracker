package artist

import (
	"context"
	"fmt"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// Unsubscribe stops tracking an artist. The artist and its catalog stay in the
// database for a later re-subscription (docs/05-data-model.md §4).
type Unsubscribe struct {
	subs usecase.SubscriptionRepository
}

func NewUnsubscribe(subs usecase.SubscriptionRepository) *Unsubscribe {
	return &Unsubscribe{subs: subs}
}

func (uc *Unsubscribe) Execute(ctx context.Context, rawMBID string) error {
	mbid, err := domain.ParseMBID(rawMBID)
	if err != nil {
		return err
	}
	if err := uc.subs.Delete(ctx, mbid); err != nil {
		return fmt.Errorf("unsubscribe %s: %w", mbid, err)
	}
	return nil
}
