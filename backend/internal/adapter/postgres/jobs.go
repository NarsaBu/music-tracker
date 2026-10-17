package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
	"music-tracker/backend/internal/infrastructure/jobs"
	"music-tracker/backend/internal/usecase"
)

// JobQueue implements usecase.JobQueue (producer side) and jobs.Store (worker side)
// on the jobs table with FOR UPDATE SKIP LOCKED (ADR-5).
type JobQueue struct {
	pool *pgxpool.Pool
}

func NewJobQueue(pool *pgxpool.Pool) *JobQueue { return &JobQueue{pool: pool} }

func (q *JobQueue) Enqueue(ctx context.Context, job usecase.Job) (bool, error) {
	payload, err := json.Marshal(job.Payload)
	if err != nil {
		return false, fmt.Errorf("enqueue %s: marshal payload: %w", job.Kind, err)
	}
	n, err := queries(ctx, q.pool).EnqueueJob(ctx, sqlc.EnqueueJobParams{
		Kind: job.Kind, Payload: payload, DedupKey: optionalText(job.DedupKey), RunAt: job.RunAt,
	})
	if err != nil {
		return false, fmt.Errorf("enqueue %s: %w", job.Kind, err)
	}
	return n == 1, nil
}

func (q *JobQueue) Claim(ctx context.Context) (jobs.Job, bool, error) {
	row, err := queries(ctx, q.pool).ClaimJob(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return jobs.Job{}, false, nil
	}
	if err != nil {
		return jobs.Job{}, false, fmt.Errorf("claim job: %w", err)
	}
	return jobs.Job{
		ID: row.ID, Kind: row.Kind, Payload: row.Payload,
		Attempts: int(row.Attempts), MaxAttempts: int(row.MaxAttempts),
	}, true, nil
}

func (q *JobQueue) Complete(ctx context.Context, id int64) error {
	return queries(ctx, q.pool).DeleteJob(ctx, id)
}

func (q *JobQueue) Retry(ctx context.Context, id int64, at time.Time, cause error) error {
	return queries(ctx, q.pool).RetryJob(ctx, sqlc.RetryJobParams{ID: id, RunAt: at, LastError: errorText(cause)})
}

func (q *JobQueue) Bury(ctx context.Context, id int64, cause error) error {
	return queries(ctx, q.pool).BuryJob(ctx, sqlc.BuryJobParams{ID: id, LastError: errorText(cause)})
}

func (q *JobQueue) ReleaseStale(ctx context.Context, lockedBefore time.Time) (int, error) {
	n, err := queries(ctx, q.pool).ReleaseStaleJobs(ctx, &lockedBefore)
	return int(n), err
}

func errorText(err error) pgtype.Text { return pgtype.Text{String: err.Error(), Valid: true} }
