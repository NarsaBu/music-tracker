// Package scheduler runs periodic tasks on cron schedules (UTC).
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron *cron.Cron
	log  *slog.Logger
}

func New(log *slog.Logger) *Scheduler {
	return &Scheduler{cron: cron.New(cron.WithLocation(time.UTC)), log: log}
}

// Add registers task under a standard 5-field cron spec. A run still in progress
// when the next tick comes is not started twice.
func (s *Scheduler) Add(ctx context.Context, name, spec string, task func(ctx context.Context) error) error {
	job := cron.NewChain(cron.SkipIfStillRunning(cron.DiscardLogger)).Then(cron.FuncJob(func() {
		if err := task(ctx); err != nil {
			s.log.ErrorContext(ctx, "scheduled task failed", slog.String("task", name), slog.String("error", err.Error()))
		}
	}))
	if _, err := s.cron.AddJob(spec, job); err != nil {
		return fmt.Errorf("schedule %s %q: %w", name, spec, err)
	}
	return nil
}

// Run blocks until ctx is done, then waits for running tasks.
func (s *Scheduler) Run(ctx context.Context) {
	s.cron.Start()
	<-ctx.Done()
	<-s.cron.Stop().Done()
}
