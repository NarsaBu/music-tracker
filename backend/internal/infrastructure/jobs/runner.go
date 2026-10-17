// Package jobs runs background jobs from the queue: a pool of workers claims
// jobs, dispatches them by kind and retries failures with backoff (ADR-5).
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"music-tracker/backend/internal/usecase"
)

// Job is a claimed job; Attempts already includes the current one.
type Job struct {
	ID          int64
	Kind        string
	Payload     []byte
	Attempts    int
	MaxAttempts int
}

// Store is the worker side of the queue.
type Store interface {
	// Claim locks the next due job; ok is false when there is none.
	Claim(ctx context.Context) (job Job, ok bool, err error)
	Complete(ctx context.Context, id int64) error
	Retry(ctx context.Context, id int64, at time.Time, cause error) error
	// Bury keeps a job that gave up as dead, for inspection.
	Bury(ctx context.Context, id int64, cause error) error
	// ReleaseStale unlocks jobs of workers that died before finishing.
	ReleaseStale(ctx context.Context, lockedBefore time.Time) (int, error)
}

// Handler processes one kind of job. lastAttempt is true when a failure will not be retried.
// Returning *usecase.PermanentError buries the job at once.
type Handler interface {
	HandleJob(ctx context.Context, payload []byte, lastAttempt bool) error
}

type Config struct {
	Concurrency  int           // default 4
	PollInterval time.Duration // idle wait between claims, default 1s
	JobTimeout   time.Duration // default 10m
	BaseBackoff  time.Duration // doubled per attempt, default 30s
	MaxBackoff   time.Duration // default 1h
}

type Runner struct {
	store    Store
	handlers map[string]Handler
	cfg      Config
	log      *slog.Logger
	now      func() time.Time
}

func NewRunner(store Store, cfg Config, log *slog.Logger) *Runner {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.JobTimeout <= 0 {
		cfg.JobTimeout = 10 * time.Minute
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = 30 * time.Second
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = time.Hour
	}
	return &Runner{store: store, handlers: map[string]Handler{}, cfg: cfg, log: log, now: time.Now}
}

// Register must be called before Run.
func (r *Runner) Register(kind string, h Handler) { r.handlers[kind] = h }

// Run blocks until ctx is done and all workers have finished their current job.
func (r *Runner) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for range r.cfg.Concurrency {
		wg.Go(func() { r.work(ctx) })
	}
	wg.Go(func() { r.releaseStale(ctx) })
	wg.Wait()
}

func (r *Runner) work(ctx context.Context) {
	for ctx.Err() == nil {
		processed, err := r.RunOnce(ctx)
		if err != nil && ctx.Err() == nil {
			r.log.ErrorContext(ctx, "job queue", slog.String("error", err.Error()))
		}
		if !processed {
			wait(ctx, r.cfg.PollInterval)
		}
	}
}

// RunOnce claims and processes at most one job; processed is false when the queue is empty.
func (r *Runner) RunOnce(ctx context.Context) (processed bool, err error) {
	job, ok, err := r.store.Claim(ctx)
	if err != nil || !ok {
		return false, err
	}

	jobErr := r.handle(ctx, job)

	// Bookkeeping must survive shutdown: the job is already locked.
	bctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	log := r.log.With(slog.Int64("job_id", job.ID), slog.String("kind", job.Kind), slog.Int("attempt", job.Attempts))

	var permanent *usecase.PermanentError
	switch {
	case jobErr == nil:
		return true, r.store.Complete(bctx, job.ID)
	case ctx.Err() != nil:
		// Interrupted by shutdown: run again as soon as possible.
		log.InfoContext(bctx, "job interrupted")
		return true, r.store.Retry(bctx, job.ID, r.now(), jobErr)
	case errors.As(jobErr, &permanent) || job.Attempts >= job.MaxAttempts:
		log.ErrorContext(bctx, "job failed for good", slog.String("error", jobErr.Error()))
		return true, r.store.Bury(bctx, job.ID, jobErr)
	default:
		at := r.now().Add(r.backoff(job.Attempts))
		log.WarnContext(bctx, "job failed, will retry", slog.String("error", jobErr.Error()), slog.Time("retry_at", at))
		return true, r.store.Retry(bctx, job.ID, at, jobErr)
	}
}

func (r *Runner) handle(ctx context.Context, job Job) (err error) {
	h, ok := r.handlers[job.Kind]
	if !ok {
		return &usecase.PermanentError{Err: fmt.Errorf("no handler for job kind %q", job.Kind)}
	}
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("job panicked: %v", rec)
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, r.cfg.JobTimeout)
	defer cancel()
	return h.HandleJob(ctx, job.Payload, job.Attempts >= job.MaxAttempts)
}

// backoff is BaseBackoff·2^(attempts-1), capped by MaxBackoff.
func (r *Runner) backoff(attempts int) time.Duration {
	d := r.cfg.BaseBackoff
	for i := 1; i < attempts && d < r.cfg.MaxBackoff; i++ {
		d *= 2
	}
	return min(d, r.cfg.MaxBackoff)
}

// releaseStale periodically unlocks jobs locked for longer than a job may run.
func (r *Runner) releaseStale(ctx context.Context) {
	every := r.cfg.JobTimeout
	for {
		n, err := r.store.ReleaseStale(ctx, r.now().Add(-2*r.cfg.JobTimeout))
		switch {
		case err != nil && ctx.Err() == nil:
			r.log.ErrorContext(ctx, "release stale jobs", slog.String("error", err.Error()))
		case n > 0:
			r.log.WarnContext(ctx, "released stale jobs", slog.Int("count", n))
		}
		if !wait(ctx, every) {
			return
		}
	}
}

// wait sleeps for d; it returns false if ctx ended first.
func wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
