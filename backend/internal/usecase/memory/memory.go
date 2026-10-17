// Package memory provides in-memory implementations of the use case ports for
// tests. They follow the same contracts as the PostgreSQL adapters (LSP).
package memory

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// Store is the shared state; its accessor methods return the individual repositories.
type Store struct {
	mu       sync.Mutex
	Artists  map[domain.MBID]domain.Artist
	Subs     map[domain.MBID]domain.Subscription
	Releases map[domain.MBID]domain.ReleaseGroup
	Links    map[domain.MBID][]domain.ArtistReleaseGroup // by artist
	Runs     []domain.SyncRun
	Jobs     []usecase.Job
	Defaults domain.ReleaseFilter
}

func NewStore() *Store {
	return &Store{
		Artists:  map[domain.MBID]domain.Artist{},
		Subs:     map[domain.MBID]domain.Subscription{},
		Releases: map[domain.MBID]domain.ReleaseGroup{},
		Links:    map[domain.MBID][]domain.ArtistReleaseGroup{},
		Defaults: domain.ReleaseFilter{
			PrimaryTypes:           []domain.PrimaryType{domain.PrimaryAlbum, domain.PrimaryEP, domain.PrimarySingle},
			ExcludedSecondaryTypes: []string{"Compilation"},
		},
	}
}

func (s *Store) ArtistRepo() *ArtistRepo             { return &ArtistRepo{s} }
func (s *Store) SubscriptionRepo() *SubscriptionRepo { return &SubscriptionRepo{s} }
func (s *Store) ReleaseRepo() *ReleaseRepo           { return &ReleaseRepo{s} }
func (s *Store) RunRepo() *RunRepo                   { return &RunRepo{s} }
func (s *Store) Queue() *Queue                       { return &Queue{s} }

// FilterDefaults implements usecase.FilterDefaultsReader.
func (s *Store) FilterDefaults(context.Context) (domain.ReleaseFilter, error) {
	return s.Defaults, nil
}

// --- artists ---

type ArtistRepo struct{ s *Store }

func (r *ArtistRepo) Upsert(_ context.Context, a domain.Artist) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if old, ok := r.s.Artists[a.MBID]; ok {
		a.LastSyncedAt = old.LastSyncedAt
	}
	r.s.Artists[a.MBID] = a
	return nil
}

func (r *ArtistRepo) Get(_ context.Context, mbid domain.MBID) (domain.Artist, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	a, ok := r.s.Artists[mbid]
	if !ok {
		return domain.Artist{}, domain.ErrNotFound
	}
	return a, nil
}

func (r *ArtistRepo) ListTracked(context.Context) ([]domain.MBID, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	var out []domain.MBID
	for mbid := range r.s.Subs {
		out = append(out, mbid)
	}
	slices.Sort(out)
	return out, nil
}

func (r *ArtistRepo) MarkSynced(_ context.Context, mbid domain.MBID, at time.Time) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	a := r.s.Artists[mbid]
	a.LastSyncedAt = &at
	r.s.Artists[mbid] = a
	return nil
}

// --- subscriptions ---

type SubscriptionRepo struct{ s *Store }

func (r *SubscriptionRepo) Create(_ context.Context, sub domain.Subscription) (bool, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if _, ok := r.s.Subs[sub.Artist.MBID]; ok {
		return false, nil
	}
	r.s.Subs[sub.Artist.MBID] = sub
	return true, nil
}

func (r *SubscriptionRepo) Get(_ context.Context, mbid domain.MBID) (domain.Subscription, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	sub, ok := r.s.Subs[mbid]
	if !ok {
		return domain.Subscription{}, domain.ErrNotFound
	}
	sub.Artist = r.s.Artists[mbid]
	return sub, nil
}

func (r *SubscriptionRepo) Delete(_ context.Context, mbid domain.MBID) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if _, ok := r.s.Subs[mbid]; !ok {
		return domain.ErrNotFound
	}
	delete(r.s.Subs, mbid)
	return nil
}

func (r *SubscriptionRepo) List(_ context.Context, q usecase.SubscriptionListQuery) ([]domain.Subscription, int, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	var all []domain.Subscription
	for mbid, sub := range r.s.Subs {
		sub.Artist = r.s.Artists[mbid]
		if strings.Contains(strings.ToLower(sub.Artist.Name), strings.ToLower(q.Search)) {
			all = append(all, sub)
		}
	}
	slices.SortFunc(all, func(a, b domain.Subscription) int {
		return strings.Compare(strings.ToLower(a.Artist.SortName), strings.ToLower(b.Artist.SortName))
	})
	page := all[min(q.Offset, len(all)):min(q.Offset+q.Limit, len(all))]
	return page, len(all), nil
}

func (r *SubscriptionRepo) SubscribedAmong(_ context.Context, mbids []domain.MBID) (map[domain.MBID]bool, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	out := map[domain.MBID]bool{}
	for _, id := range mbids {
		if _, ok := r.s.Subs[id]; ok {
			out[id] = true
		}
	}
	return out, nil
}

// --- release groups ---

type ReleaseRepo struct{ s *Store }

func (r *ReleaseRepo) KnownMBIDs(_ context.Context, artist domain.MBID) (map[domain.MBID]bool, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	out := map[domain.MBID]bool{}
	for _, l := range r.s.Links[artist] {
		out[l.ReleaseGroup.MBID] = true
	}
	return out, nil
}

func (r *ReleaseRepo) UpsertMany(_ context.Context, rgs []domain.ReleaseGroup, _ time.Time) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, rg := range rgs {
		rg.Credits = nil // not stored, like in PostgreSQL
		r.s.Releases[rg.MBID] = rg
	}
	return nil
}

func (r *ReleaseRepo) LinkMany(_ context.Context, links []domain.ArtistReleaseGroup) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, l := range links {
		if !slices.ContainsFunc(r.s.Links[l.ArtistMBID], func(x domain.ArtistReleaseGroup) bool {
			return x.ReleaseGroup.MBID == l.ReleaseGroup.MBID
		}) {
			r.s.Links[l.ArtistMBID] = append(r.s.Links[l.ArtistMBID], l)
		}
	}
	return nil
}

func (r *ReleaseRepo) ListByArtist(_ context.Context, artist domain.MBID, role domain.CreditRole) ([]domain.ArtistReleaseGroup, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	var out []domain.ArtistReleaseGroup
	for _, l := range r.s.Links[artist] {
		if role == "" || l.Role == role {
			l.ReleaseGroup = r.s.Releases[l.ReleaseGroup.MBID]
			out = append(out, l)
		}
	}
	return out, nil
}

// --- sync runs ---

type RunRepo struct{ s *Store }

func (r *RunRepo) Create(_ context.Context, trigger domain.SyncTrigger, total int, at time.Time) (domain.SyncRun, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	run := domain.SyncRun{ID: "run-" + string(rune('a'+len(r.s.Runs))), Trigger: trigger, StartedAt: at, ArtistsTotal: total}
	if total == 0 {
		run.FinishedAt = &at
	}
	r.s.Runs = append(r.s.Runs, run)
	return run, nil
}

func (r *RunRepo) Running(_ context.Context, since time.Time) (domain.SyncRun, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for i := len(r.s.Runs) - 1; i >= 0; i-- {
		if run := r.s.Runs[i]; !run.Finished() && run.StartedAt.After(since) {
			return run, nil
		}
	}
	return domain.SyncRun{}, domain.ErrNotFound
}

func (r *RunRepo) Latest(context.Context) (domain.SyncRun, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if len(r.s.Runs) == 0 {
		return domain.SyncRun{}, domain.ErrNotFound
	}
	return r.s.Runs[len(r.s.Runs)-1], nil
}

func (r *RunRepo) ArtistDone(_ context.Context, id string, newReleases int, at time.Time) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for i, run := range r.s.Runs {
		if run.ID != id || run.Finished() {
			continue
		}
		run.ArtistsDone++
		run.NewReleases += newReleases
		if run.ArtistsDone >= run.ArtistsTotal {
			run.FinishedAt = &at
		}
		r.s.Runs[i] = run
	}
	return nil
}

// --- job queue ---

type Queue struct{ s *Store }

func (q *Queue) Enqueue(_ context.Context, job usecase.Job) (bool, error) {
	q.s.mu.Lock()
	defer q.s.mu.Unlock()
	if _, err := json.Marshal(job.Payload); err != nil {
		return false, err
	}
	if job.DedupKey != "" && slices.ContainsFunc(q.s.Jobs, func(j usecase.Job) bool { return j.DedupKey == job.DedupKey }) {
		return false, nil
	}
	q.s.Jobs = append(q.s.Jobs, job)
	return true, nil
}

// TxManager runs fn directly: the in-memory store has no rollback.
type TxManager struct{}

func (TxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// Clock is a settable clock.
type Clock struct{ T time.Time }

func (c *Clock) Now() time.Time { return c.T }
