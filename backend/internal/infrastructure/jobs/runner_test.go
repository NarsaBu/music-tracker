package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"music-tracker/backend/internal/usecase"
)

type outcome struct {
	kind  string // complete | retry | bury
	at    time.Time
	cause error
}

type fakeStore struct {
	mu       sync.Mutex
	queue    []Job
	outcomes map[int64]outcome
}

func (s *fakeStore) Claim(context.Context) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.queue) == 0 {
		return Job{}, false, nil
	}
	j := s.queue[0]
	s.queue = s.queue[1:]
	j.Attempts++
	return j, true, nil
}

func (s *fakeStore) record(id int64, o outcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.outcomes == nil {
		s.outcomes = map[int64]outcome{}
	}
	s.outcomes[id] = o
	return nil
}

func (s *fakeStore) Complete(_ context.Context, id int64) error {
	return s.record(id, outcome{kind: "complete"})
}

func (s *fakeStore) Retry(_ context.Context, id int64, at time.Time, cause error) error {
	return s.record(id, outcome{kind: "retry", at: at, cause: cause})
}

func (s *fakeStore) Bury(_ context.Context, id int64, cause error) error {
	return s.record(id, outcome{kind: "bury", cause: cause})
}
func (s *fakeStore) ReleaseStale(context.Context, time.Time) (int, error) { return 0, nil }

type handlerFunc func(ctx context.Context, payload []byte, last bool) error

func (f handlerFunc) HandleJob(ctx context.Context, payload []byte, last bool) error {
	return f(ctx, payload, last)
}

var t0 = time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)

func newRunner(store *fakeStore) *Runner {
	r := NewRunner(store, Config{BaseBackoff: time.Minute, MaxBackoff: 10 * time.Minute}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.now = func() time.Time { return t0 }
	return r
}

func TestRunOnce_Outcomes(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name     string
		job      Job
		handler  handlerFunc
		want     string
		wantLast bool
		wantAt   time.Time
	}{
		{"success", Job{ID: 1, Kind: "k", MaxAttempts: 5}, func(context.Context, []byte, bool) error { return nil }, "complete", false, time.Time{}},
		{"first failure retries after base backoff", Job{ID: 2, Kind: "k", MaxAttempts: 5},
			func(context.Context, []byte, bool) error { return boom }, "retry", false, t0.Add(time.Minute)},
		{"third failure backs off 4x", Job{ID: 3, Kind: "k", Attempts: 2, MaxAttempts: 5},
			func(context.Context, []byte, bool) error { return boom }, "retry", false, t0.Add(4 * time.Minute)},
		{"backoff is capped", Job{ID: 4, Kind: "k", Attempts: 8, MaxAttempts: 20},
			func(context.Context, []byte, bool) error { return boom }, "retry", false, t0.Add(10 * time.Minute)},
		{"last attempt buries", Job{ID: 5, Kind: "k", Attempts: 4, MaxAttempts: 5},
			func(context.Context, []byte, bool) error { return boom }, "bury", true, time.Time{}},
		{"permanent error buries at once", Job{ID: 6, Kind: "k", MaxAttempts: 5},
			func(context.Context, []byte, bool) error { return &usecase.PermanentError{Err: boom} }, "bury", false, time.Time{}},
		{"panic is a failure", Job{ID: 7, Kind: "k", MaxAttempts: 5},
			func(context.Context, []byte, bool) error { panic("oops") }, "retry", false, t0.Add(time.Minute)},
		{"unknown kind buries", Job{ID: 8, Kind: "nope", MaxAttempts: 5}, nil, "bury", false, time.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{queue: []Job{tt.job}}
			r := newRunner(store)
			var gotLast bool
			if tt.handler != nil {
				r.Register("k", handlerFunc(func(ctx context.Context, p []byte, last bool) error {
					gotLast = last
					return tt.handler(ctx, p, last)
				}))
			}

			processed, err := r.RunOnce(context.Background())
			if err != nil || !processed {
				t.Fatalf("processed %t, err %v", processed, err)
			}
			got := store.outcomes[tt.job.ID]
			if got.kind != tt.want || !got.at.Equal(tt.wantAt) || gotLast != tt.wantLast {
				t.Errorf("outcome = %+v, last %t; want %s at %v, last %t", got, gotLast, tt.want, tt.wantAt, tt.wantLast)
			}
		})
	}
}

func TestRunOnce_EmptyQueue(t *testing.T) {
	processed, err := newRunner(&fakeStore{}).RunOnce(context.Background())
	if processed || err != nil {
		t.Errorf("processed %t, err %v", processed, err)
	}
}

func TestRun_ProcessesQueueAndStops(t *testing.T) {
	store := &fakeStore{}
	for i := range 10 {
		store.queue = append(store.queue, Job{ID: int64(i), Kind: "k", MaxAttempts: 1})
	}
	r := newRunner(store)
	r.cfg.PollInterval = 5 * time.Millisecond
	r.Register("k", handlerFunc(func(context.Context, []byte, bool) error { return nil }))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { r.Run(ctx); close(done) }()

	deadline := time.Now().Add(2 * time.Second)
	for {
		store.mu.Lock()
		n := len(store.outcomes)
		store.mu.Unlock()
		if n == 10 || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after cancel")
	}
	if len(store.outcomes) != 10 {
		t.Errorf("processed %d jobs, want 10", len(store.outcomes))
	}
}
