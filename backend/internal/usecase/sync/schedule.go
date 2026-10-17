package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// abandonAfter: an unfinished run older than this no longer blocks a new one.
const abandonAfter = 12 * time.Hour

// ScheduleSyncRun starts a pass over all tracked artists: a run plus one
// sync_artist job per artist. While a run is in progress it is returned instead.
type ScheduleSyncRun struct {
	artists usecase.ArtistRepository
	runs    usecase.SyncRunRepository
	queue   usecase.JobQueue
	tx      usecase.TxManager
	clock   usecase.Clock
}

func NewScheduleSyncRun(
	artists usecase.ArtistRepository,
	runs usecase.SyncRunRepository,
	queue usecase.JobQueue,
	tx usecase.TxManager,
	clock usecase.Clock,
) *ScheduleSyncRun {
	return &ScheduleSyncRun{artists: artists, runs: runs, queue: queue, tx: tx, clock: clock}
}

// Execute reports started = false when an unfinished run was returned.
func (uc *ScheduleSyncRun) Execute(ctx context.Context, trigger domain.SyncTrigger) (run domain.SyncRun, started bool, err error) {
	now := uc.clock.Now()
	running, err := uc.runs.Running(ctx, now.Add(-abandonAfter))
	switch {
	case err == nil:
		return running, false, nil
	case !errors.Is(err, domain.ErrNotFound):
		return domain.SyncRun{}, false, fmt.Errorf("check running sync: %w", err)
	}

	tracked, err := uc.artists.ListTracked(ctx)
	if err != nil {
		return domain.SyncRun{}, false, fmt.Errorf("list tracked artists: %w", err)
	}
	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		run, err = uc.runs.Create(ctx, trigger, len(tracked), now)
		if err != nil {
			return err
		}
		for _, mbid := range tracked {
			if _, err := uc.queue.Enqueue(ctx, usecase.NewSyncArtistJob(string(mbid), run.ID, run.ID, now)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.SyncRun{}, false, fmt.Errorf("start sync run: %w", err)
	}
	return run, true, nil
}
