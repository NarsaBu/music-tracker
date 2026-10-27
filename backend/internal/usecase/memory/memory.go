// Package memory provides in-memory implementations of the use case ports for
// tests. They follow the same contracts as the PostgreSQL adapters (LSP).
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
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
	Settings domain.Settings
	Notifs   []StoredNotification
	Imports  []domain.ImportJob
	Items    []domain.ImportItem
}

// StoredNotification is a notification row as the outbox keeps it.
type StoredNotification struct {
	domain.Notification
	ID       string
	Attempts int
}

func NewStore() *Store {
	return &Store{
		Artists:  map[domain.MBID]domain.Artist{},
		Subs:     map[domain.MBID]domain.Subscription{},
		Releases: map[domain.MBID]domain.ReleaseGroup{},
		Links:    map[domain.MBID][]domain.ArtistReleaseGroup{},
		Settings: domain.Settings{
			Filter: domain.ReleaseFilter{
				PrimaryTypes:           []domain.PrimaryType{domain.PrimaryAlbum, domain.PrimaryEP, domain.PrimarySingle},
				ExcludedSecondaryTypes: []string{"Compilation"},
			},
			NotifyAnnouncements: true,
			NewReleaseDays:      90,
			Email: domain.EmailSettings{Digest: domain.DigestSchedule{
				Frequency: domain.DigestDaily, Hour: 9, Timezone: "UTC",
			}},
			RSS: domain.RSSSettings{Enabled: true, FeedToken: "token"},
		},
	}
}

func (s *Store) ArtistRepo() *ArtistRepo             { return &ArtistRepo{s} }
func (s *Store) SubscriptionRepo() *SubscriptionRepo { return &SubscriptionRepo{s} }
func (s *Store) ReleaseRepo() *ReleaseRepo           { return &ReleaseRepo{s} }
func (s *Store) RunRepo() *RunRepo                   { return &RunRepo{s} }
func (s *Store) Queue() *Queue                       { return &Queue{s} }
func (s *Store) SettingsRepo() *SettingsRepo         { return &SettingsRepo{s} }
func (s *Store) NotificationRepo() *NotificationRepo { return &NotificationRepo{s} }
func (s *Store) ImportRepo() *ImportRepo             { return &ImportRepo{s} }

// FilterDefaults implements usecase.FilterDefaultsReader.
func (s *Store) FilterDefaults(context.Context) (domain.ReleaseFilter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Settings.Filter, nil
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
		if matchesListQuery(sub, q) {
			all = append(all, sub)
		}
	}
	slices.SortFunc(all, func(a, b domain.Subscription) int {
		if q.NewestFirst {
			if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
				return c
			}
		}
		return strings.Compare(strings.ToLower(a.Artist.SortName), strings.ToLower(b.Artist.SortName))
	})
	page := all[min(q.Offset, len(all)):min(q.Offset+q.Limit, len(all))]
	return page, len(all), nil
}

func matchesListQuery(sub domain.Subscription, q usecase.SubscriptionListQuery) bool {
	a := sub.Artist
	o := sub.Override
	overridden := o.PrimaryTypes != nil || o.ExcludedSecondaryTypes != nil || o.IncludeFeatured != nil
	return strings.Contains(strings.ToLower(a.Name), strings.ToLower(q.Search)) &&
		(q.Synced == nil || *q.Synced == (a.LastSyncedAt != nil)) &&
		(q.ArtistType == "" || a.Type == q.ArtistType) &&
		(q.Country == "" || a.Country == q.Country) &&
		(q.Overridden == nil || *q.Overridden == overridden)
}

func (r *SubscriptionRepo) Facets(_ context.Context) (usecase.SubscriptionFacets, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	countries, types := map[string]bool{}, map[string]bool{}
	for mbid := range r.s.Subs {
		a := r.s.Artists[mbid]
		if a.Country != "" {
			countries[a.Country] = true
		}
		if a.Type != "" {
			types[a.Type] = true
		}
	}
	return usecase.SubscriptionFacets{
		Countries:   slices.Sorted(maps.Keys(countries)),
		ArtistTypes: slices.Sorted(maps.Keys(types)),
	}, nil
}

func (r *SubscriptionRepo) UpdateOverride(_ context.Context, mbid domain.MBID, o domain.FilterOverride) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	sub, ok := r.s.Subs[mbid]
	if !ok {
		return domain.ErrNotFound
	}
	sub.Override = o
	r.s.Subs[mbid] = sub
	return nil
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

func (r *ReleaseRepo) Get(_ context.Context, mbid domain.MBID) (domain.ReleaseGroup, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	rg, ok := r.s.Releases[mbid]
	if !ok {
		return domain.ReleaseGroup{}, domain.ErrNotFound
	}
	return rg, nil
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

func (q *Queue) EnqueueMany(ctx context.Context, jobs []usecase.Job) error {
	for _, job := range jobs {
		if _, err := q.Enqueue(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

// TxManager runs fn directly: the in-memory store has no rollback.
type TxManager struct{}

func (TxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// Clock is a settable clock.
type Clock struct{ T time.Time }

func (c *Clock) Now() time.Time { return c.T }

// --- settings ---

type SettingsRepo struct{ s *Store }

func (r *SettingsRepo) FilterDefaults(ctx context.Context) (domain.ReleaseFilter, error) {
	return r.s.FilterDefaults(ctx)
}

func (r *SettingsRepo) Get(context.Context) (domain.Settings, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	return r.s.Settings, nil
}

func (r *SettingsRepo) Save(_ context.Context, st domain.Settings) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	st.RSS.FeedToken = r.s.Settings.RSS.FeedToken
	st.LastDigestAt = r.s.Settings.LastDigestAt
	r.s.Settings = st
	return nil
}

func (r *SettingsRepo) SetFeedToken(_ context.Context, token string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.Settings.RSS.FeedToken = token
	return nil
}

func (r *SettingsRepo) SetDigestTimes(_ context.Context, last, next *time.Time) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if last != nil {
		r.s.Settings.LastDigestAt = last
	}
	r.s.Settings.NextDigestAt = next
	return nil
}

// --- notifications ---

type NotificationRepo struct{ s *Store }

func (r *NotificationRepo) Insert(_ context.Context, n domain.Notification) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for i, old := range r.s.Notifs {
		if old.ReleaseGroup == n.ReleaseGroup && old.Channel == n.Channel {
			if old.Role != domain.RoleMain && n.Role == domain.RoleMain {
				r.s.Notifs[i].Artist, r.s.Notifs[i].Role = n.Artist, n.Role
			}
			return nil
		}
	}
	id := fmt.Sprintf("00000000-0000-0000-0000-%012d", len(r.s.Notifs)+1)
	r.s.Notifs = append(r.s.Notifs, StoredNotification{Notification: n, ID: id})
	return nil
}

func (r *NotificationRepo) item(n StoredNotification) domain.NotificationItem {
	return domain.NotificationItem{
		ID: n.ID, ReleaseGroup: r.s.Releases[n.ReleaseGroup], FoundVia: r.s.Artists[n.Artist],
		Role: n.Role, CreatedAt: n.At,
	}
}

func (r *NotificationRepo) ClaimPendingEmail(context.Context) ([]domain.NotificationItem, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	var out []domain.NotificationItem
	for _, n := range r.s.Notifs {
		if n.Channel == domain.ChannelEmail && n.Status == domain.StatusPending {
			out = append(out, r.item(n))
		}
	}
	return out, nil
}

func (r *NotificationRepo) update(ids []string, f func(*StoredNotification)) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for i := range r.s.Notifs {
		if slices.Contains(ids, r.s.Notifs[i].ID) {
			f(&r.s.Notifs[i])
		}
	}
}

func (r *NotificationRepo) MarkSent(_ context.Context, ids []string, _ time.Time) error {
	r.update(ids, func(n *StoredNotification) { n.Status = domain.StatusSent })
	return nil
}

func (r *NotificationRepo) MarkFailed(_ context.Context, ids []string, _ error) error {
	r.update(ids, func(n *StoredNotification) {
		n.Attempts++
		if n.Attempts >= domain.MaxEmailAttempts {
			n.Status = domain.StatusFailed
		}
	})
	return nil
}

func (r *NotificationRepo) ListFeed(_ context.Context, q usecase.FeedQuery) ([]domain.NotificationItem, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	var rows []StoredNotification
	for _, n := range r.s.Notifs {
		if n.Channel != domain.ChannelRSS || (q.Role != "" && n.Role != q.Role) {
			continue
		}
		if q.Type != "" && r.s.Releases[n.ReleaseGroup].PrimaryType != q.Type {
			continue
		}
		rows = append(rows, n)
	}
	// Newest first; the id breaks ties like in PostgreSQL.
	key := func(n StoredNotification) time.Time {
		if q.Sort != usecase.FeedByRelease {
			return n.At
		}
		if d := r.s.Releases[n.ReleaseGroup].FirstRelease.Day; d != nil {
			return *d
		}
		return time.Time{} // undated last
	}
	slices.SortFunc(rows, func(a, b StoredNotification) int {
		if c := key(b).Compare(key(a)); c != 0 {
			return c
		}
		return strings.Compare(b.ID, a.ID)
	})
	var out []domain.NotificationItem
	for _, n := range rows {
		if q.Before != nil && (key(n).After(q.Before.At) || key(n).Equal(q.Before.At) && n.ID >= q.Before.ID) {
			continue
		}
		if len(out) == q.Limit {
			break
		}
		out = append(out, r.item(n))
	}
	return out, nil
}

// --- imports ---

type ImportRepo struct{ s *Store }

func (r *ImportRepo) Create(_ context.Context, filename string, lines []domain.ImportLine, at time.Time) (domain.ImportJob, []domain.ImportItem, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	job := domain.ImportJob{ID: fmt.Sprintf("import-%d", len(r.s.Imports)+1), Filename: filename, Total: len(lines), CreatedAt: at}
	r.s.Imports = append(r.s.Imports, job)
	var items []domain.ImportItem
	for _, l := range lines {
		it := domain.ImportItem{
			ID: fmt.Sprintf("item-%d", len(r.s.Items)+1), JobID: job.ID, LineNo: l.LineNo, RawValue: l.Value,
			InputMBID: l.MBID, IncludeFeatured: l.IncludeFeatured, Status: domain.ImportPending,
		}
		r.s.Items = append(r.s.Items, it)
		items = append(items, it)
	}
	return r.withCounts(job), items, nil
}

func (r *ImportRepo) withCounts(job domain.ImportJob) domain.ImportJob {
	job.Counts = map[domain.ImportItemStatus]int{}
	for _, it := range r.s.Items {
		if it.JobID == job.ID {
			job.Counts[it.Status]++
		}
	}
	return job
}

func (r *ImportRepo) Get(_ context.Context, id string) (domain.ImportJob, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, job := range r.s.Imports {
		if job.ID == id {
			return r.withCounts(job), nil
		}
	}
	return domain.ImportJob{}, domain.ErrNotFound
}

func (r *ImportRepo) List(_ context.Context, limit int) ([]domain.ImportJob, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	var out []domain.ImportJob
	for i := len(r.s.Imports) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.withCounts(r.s.Imports[i]))
	}
	return out, nil
}

func (r *ImportRepo) GetItem(_ context.Context, jobID, itemID string) (domain.ImportItem, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, it := range r.s.Items {
		if it.ID == itemID && (jobID == "" || it.JobID == jobID) {
			return it, nil
		}
	}
	return domain.ImportItem{}, domain.ErrNotFound
}

func (r *ImportRepo) ListItems(_ context.Context, jobID string, statuses []domain.ImportItemStatus) ([]domain.ImportItem, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	var out []domain.ImportItem
	for _, it := range r.s.Items {
		if it.JobID == jobID && (len(statuses) == 0 || slices.Contains(statuses, it.Status)) {
			out = append(out, it)
		}
	}
	return out, nil
}

func (r *ImportRepo) UpdateItem(_ context.Context, item domain.ImportItem) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for i, it := range r.s.Items {
		if it.ID == item.ID {
			if item.Artist != nil {
				// Like the database: the item references the artist, the name comes from the artists table.
				item.Artist = &domain.Artist{MBID: item.Artist.MBID, Name: r.s.Artists[item.Artist.MBID].Name}
			}
			r.s.Items[i] = item
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *ImportRepo) FinishIfDone(_ context.Context, jobID string, at time.Time) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for i, job := range r.s.Imports {
		if job.ID != jobID || job.Finished() {
			continue
		}
		if r.withCounts(job).Counts[domain.ImportPending] == 0 {
			r.s.Imports[i].FinishedAt = &at
		}
	}
	return nil
}
