package artist

import (
	"context"
	"slices"
	"strings"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

const (
	nirvana   domain.MBID = "5b11f4ce-a62d-471e-81fc-a69a8278c7da"
	radiohead domain.MBID = "a74b1b7f-71a5-4011-9441-d0b5e4122711"
)

type fakeCatalog struct {
	artists   map[domain.MBID]domain.Artist
	found     []domain.ArtistCandidate
	err       error
	getCalls  int
	lastLimit int
}

func (f *fakeCatalog) SearchArtists(_ context.Context, _ string, limit int) ([]domain.ArtistCandidate, error) {
	f.lastLimit = limit
	return f.found, f.err
}

func (f *fakeCatalog) GetArtist(_ context.Context, mbid domain.MBID) (domain.Artist, error) {
	f.getCalls++
	if f.err != nil {
		return domain.Artist{}, f.err
	}
	a, ok := f.artists[mbid]
	if !ok {
		return domain.Artist{}, domain.ErrNotFound
	}
	return a, nil
}

// memStore implements ArtistRepository and SubscriptionRepository in memory.
type memStore struct {
	artists map[domain.MBID]domain.Artist
	subs    map[domain.MBID]domain.Subscription
}

func newMemStore() *memStore {
	return &memStore{artists: map[domain.MBID]domain.Artist{}, subs: map[domain.MBID]domain.Subscription{}}
}

func (m *memStore) Upsert(_ context.Context, a domain.Artist) error {
	if old, ok := m.artists[a.MBID]; ok {
		a.LastSyncedAt = old.LastSyncedAt
	}
	m.artists[a.MBID] = a
	return nil
}

func (m *memStore) Create(_ context.Context, s domain.Subscription) (bool, error) {
	if _, ok := m.subs[s.Artist.MBID]; ok {
		return false, nil
	}
	m.subs[s.Artist.MBID] = s
	return true, nil
}

func (m *memStore) Get(_ context.Context, mbid domain.MBID) (domain.Subscription, error) {
	s, ok := m.subs[mbid]
	if !ok {
		return domain.Subscription{}, domain.ErrNotFound
	}
	s.Artist = m.artists[mbid]
	return s, nil
}

func (m *memStore) Delete(_ context.Context, mbid domain.MBID) error {
	if _, ok := m.subs[mbid]; !ok {
		return domain.ErrNotFound
	}
	delete(m.subs, mbid)
	return nil
}

func (m *memStore) List(_ context.Context, q usecase.SubscriptionListQuery) ([]domain.Subscription, int, error) {
	var all []domain.Subscription
	for mbid, s := range m.subs {
		s.Artist = m.artists[mbid]
		if strings.Contains(strings.ToLower(s.Artist.Name), strings.ToLower(q.Search)) {
			all = append(all, s)
		}
	}
	slices.SortFunc(all, func(a, b domain.Subscription) int { return strings.Compare(a.Artist.SortName, b.Artist.SortName) })
	page := all[min(q.Offset, len(all)):min(q.Offset+q.Limit, len(all))]
	return page, len(all), nil
}

func (m *memStore) SubscribedAmong(_ context.Context, mbids []domain.MBID) (map[domain.MBID]bool, error) {
	out := map[domain.MBID]bool{}
	for _, id := range mbids {
		if _, ok := m.subs[id]; ok {
			out[id] = true
		}
	}
	return out, nil
}

type fixedDefaults domain.ReleaseFilter

func (f fixedDefaults) FilterDefaults(context.Context) (domain.ReleaseFilter, error) {
	return domain.ReleaseFilter(f), nil
}

type noTx struct{}

func (noTx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type fixedClock time.Time

func (c fixedClock) Now() time.Time { return time.Time(c) }

var defaultFilter = fixedDefaults{
	PrimaryTypes:           []domain.PrimaryType{domain.PrimaryAlbum, domain.PrimaryEP, domain.PrimarySingle},
	ExcludedSecondaryTypes: []string{"Compilation"},
}
