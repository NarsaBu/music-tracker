package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
	"music-tracker/backend/internal/domain"
)

// SyncRunRepository implements usecase.SyncRunRepository.
type SyncRunRepository struct {
	pool *pgxpool.Pool
}

func NewSyncRunRepository(pool *pgxpool.Pool) *SyncRunRepository {
	return &SyncRunRepository{pool: pool}
}

func (r *SyncRunRepository) Create(ctx context.Context, trigger domain.SyncTrigger, total int, at time.Time) (domain.SyncRun, error) {
	var finished *time.Time
	if total == 0 {
		finished = &at
	}
	row, err := queries(ctx, r.pool).CreateSyncRun(ctx, sqlc.CreateSyncRunParams{
		Trigger: string(trigger), StartedAt: at, ArtistsTotal: int32(total), FinishedAt: finished, //nolint:gosec // artist count
	})
	if err != nil {
		return domain.SyncRun{}, fmt.Errorf("create sync run: %w", err)
	}
	return syncRunFromRow(row), nil
}

func (r *SyncRunRepository) Running(ctx context.Context, since time.Time) (domain.SyncRun, error) {
	row, err := queries(ctx, r.pool).GetRunningSyncRun(ctx, since)
	return syncRunOrNotFound(row, err)
}

func (r *SyncRunRepository) Latest(ctx context.Context) (domain.SyncRun, error) {
	row, err := queries(ctx, r.pool).GetLatestSyncRun(ctx)
	return syncRunOrNotFound(row, err)
}

func (r *SyncRunRepository) ArtistDone(ctx context.Context, runID string, newReleases int, at time.Time) error {
	id, err := uuid.Parse(runID)
	if err != nil {
		return fmt.Errorf("sync run id %q: %w", runID, err)
	}
	err = queries(ctx, r.pool).SyncRunArtistDone(ctx, sqlc.SyncRunArtistDoneParams{
		ID: id, NewReleases: int32(newReleases), Now: at, //nolint:gosec // release count
	})
	if err != nil {
		return fmt.Errorf("advance sync run %s: %w", runID, err)
	}
	return nil
}

func syncRunOrNotFound(row sqlc.SyncRun, err error) (domain.SyncRun, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SyncRun{}, fmt.Errorf("sync run: %w", domain.ErrNotFound)
	}
	if err != nil {
		return domain.SyncRun{}, fmt.Errorf("get sync run: %w", err)
	}
	return syncRunFromRow(row), nil
}

func syncRunFromRow(r sqlc.SyncRun) domain.SyncRun {
	return domain.SyncRun{
		ID:           r.ID.String(),
		Trigger:      domain.SyncTrigger(r.Trigger),
		StartedAt:    r.StartedAt.UTC(),
		FinishedAt:   utcPtr(r.FinishedAt),
		ArtistsTotal: int(r.ArtistsTotal),
		ArtistsDone:  int(r.ArtistsDone),
		NewReleases:  int(r.NewReleases),
	}
}
