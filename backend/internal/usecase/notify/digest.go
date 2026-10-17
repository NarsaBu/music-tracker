package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// ScheduleDigest is polled every minute: when the digest is due it queues a
// send_digest job, deduplicated by the due time.
type ScheduleDigest struct {
	settings usecase.SettingsRepository
	queue    usecase.JobQueue
	clock    usecase.Clock
}

func NewScheduleDigest(settings usecase.SettingsRepository, queue usecase.JobQueue, clock usecase.Clock) *ScheduleDigest {
	return &ScheduleDigest{settings: settings, queue: queue, clock: clock}
}

func (uc *ScheduleDigest) Execute(ctx context.Context) error {
	s, err := uc.settings.Get(ctx)
	if err != nil {
		return fmt.Errorf("schedule digest: %w", err)
	}
	now := uc.clock.Now()
	if !s.EmailActive() || s.NextDigestAt == nil || now.Before(*s.NextDigestAt) {
		return nil
	}
	if _, err := uc.queue.Enqueue(ctx, usecase.NewSendDigestJob(*s.NextDigestAt, now)); err != nil {
		return fmt.Errorf("schedule digest: %w", err)
	}
	return nil
}

// SendDigest sends one email with every pending email notification, then moves
// the schedule forward. No news → no email, the schedule still moves (FR-5.4).
type SendDigest struct {
	settings      usecase.SettingsRepository
	notifications usecase.NotificationRepository
	mailer        usecase.DigestMailer
	tx            usecase.TxManager
	clock         usecase.Clock
}

func NewSendDigest(
	settings usecase.SettingsRepository,
	notifications usecase.NotificationRepository,
	mailer usecase.DigestMailer,
	tx usecase.TxManager,
	clock usecase.Clock,
) *SendDigest {
	return &SendDigest{settings: settings, notifications: notifications, mailer: mailer, tx: tx, clock: clock}
}

// HandleJob runs a usecase.JobSendDigest job.
func (uc *SendDigest) HandleJob(ctx context.Context, payload []byte, lastAttempt bool) error {
	var p usecase.SendDigestPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return &usecase.PermanentError{Err: fmt.Errorf("decode payload: %w", err)}
	}
	_, err := uc.Execute(ctx, p.Due, lastAttempt)
	return err
}

// Execute sends the digest due at `due` and reports how many items went out.
// A job for an outdated slot (settings changed, digest already sent) does nothing.
func (uc *SendDigest) Execute(ctx context.Context, due time.Time, lastAttempt bool) (int, error) {
	s, err := uc.settings.Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("send digest: %w", err)
	}
	if !s.EmailActive() || s.NextDigestAt == nil || !s.NextDigestAt.Equal(due) {
		return 0, nil
	}
	now := uc.clock.Now()
	next, err := s.Email.Digest.Next(now)
	if err != nil {
		return 0, &usecase.PermanentError{Err: err}
	}

	var ids []string
	var sendErr error
	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		items, err := uc.notifications.ClaimPendingEmail(ctx)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return uc.settings.SetDigestTimes(ctx, nil, &next)
		}
		ids = make([]string, len(items))
		for i, it := range items {
			ids[i] = it.ID
		}
		// Sent before commit: a crash right after SMTP may repeat the email (at-least-once).
		if sendErr = uc.mailer.SendDigest(ctx, s.Email.To, domain.NewDigest(s.Email.Digest.Frequency, items, now)); sendErr != nil {
			return sendErr
		}
		if err := uc.notifications.MarkSent(ctx, ids, now); err != nil {
			return err
		}
		return uc.settings.SetDigestTimes(ctx, &now, &next)
	})
	if sendErr == nil {
		if err != nil {
			return 0, fmt.Errorf("send digest: %w", err)
		}
		return len(ids), nil
	}

	// SMTP failed: count the attempt; when the job gives up, move on to the next slot
	// (the items stay pending and go out with the next digest).
	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.notifications.MarkFailed(ctx, ids, sendErr); err != nil {
			return err
		}
		if lastAttempt {
			return uc.settings.SetDigestTimes(ctx, nil, &next)
		}
		return nil
	})
	return 0, errors.Join(fmt.Errorf("send digest: %w: %w", domain.ErrUpstreamUnavailable, sendErr), err)
}

// SendTestEmail sends a test message to the configured address (FR-5.5).
type SendTestEmail struct {
	settings usecase.SettingsRepository
	mailer   usecase.DigestMailer
}

func NewSendTestEmail(settings usecase.SettingsRepository, mailer usecase.DigestMailer) *SendTestEmail {
	return &SendTestEmail{settings: settings, mailer: mailer}
}

func (uc *SendTestEmail) Execute(ctx context.Context) error {
	s, err := uc.settings.Get(ctx)
	if err != nil {
		return fmt.Errorf("send test email: %w", err)
	}
	if s.Email.To == "" {
		return &domain.ValidationError{Fields: []domain.FieldError{{Field: "email.to", Message: "save an address first"}}}
	}
	if err := uc.mailer.SendTest(ctx, s.Email.To); err != nil {
		return fmt.Errorf("send test email: %w: %w", domain.ErrUpstreamUnavailable, err)
	}
	return nil
}
