package sync

import (
	"context"
	"errors"
	"fmt"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// RequestArtist queues a check of one subscribed artist ("Проверить сейчас", US-10).
type RequestArtist struct {
	subs  usecase.SubscriptionRepository
	queue usecase.JobQueue
	clock usecase.Clock
}

func NewRequestArtist(subs usecase.SubscriptionRepository, queue usecase.JobQueue, clock usecase.Clock) *RequestArtist {
	return &RequestArtist{subs: subs, queue: queue, clock: clock}
}

// Execute is idempotent: while a manual check is queued, another one is not added.
func (uc *RequestArtist) Execute(ctx context.Context, rawMBID string) error {
	mbid, err := domain.ParseMBID(rawMBID)
	if err != nil {
		return err
	}
	if _, err := uc.subs.Get(ctx, mbid); err != nil {
		return fmt.Errorf("sync subscription %s: %w", mbid, err)
	}
	if _, err := uc.queue.Enqueue(ctx, usecase.NewSyncArtistJob(string(mbid), "", "manual", uc.clock.Now())); err != nil {
		return fmt.Errorf("sync subscription %s: enqueue: %w", mbid, err)
	}
	return nil
}

// LatestSyncRun returns the most recent run, or nil before the first one.
type LatestSyncRun struct {
	runs usecase.SyncRunRepository
}

func NewLatestSyncRun(runs usecase.SyncRunRepository) *LatestSyncRun {
	return &LatestSyncRun{runs: runs}
}

func (uc *LatestSyncRun) Execute(ctx context.Context) (*domain.SyncRun, error) {
	run, err := uc.runs.Latest(ctx)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest sync run: %w", err)
	}
	return &run, nil
}
