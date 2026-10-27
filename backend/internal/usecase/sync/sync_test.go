package sync

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
	"music-tracker/backend/internal/usecase/memory"
	"music-tracker/backend/internal/usecase/notify"
)

const (
	artistA domain.MBID = "5b11f4ce-a62d-471e-81fc-a69a8278c7da"
	artistB domain.MBID = "a74b1b7f-71a5-4011-9441-d0b5e4122711"
	rg1     domain.MBID = "00000000-0000-0000-0000-000000000001"
	rg2     domain.MBID = "00000000-0000-0000-0000-000000000002"
	rg3     domain.MBID = "00000000-0000-0000-0000-000000000003"
)

var now = time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)

type fakeCatalog struct {
	groups map[domain.MBID][]domain.ReleaseGroup
	err    error
}

func (f *fakeCatalog) SearchArtists(context.Context, string, int) ([]domain.ArtistCandidate, error) {
	return nil, nil
}

func (f *fakeCatalog) GetArtist(context.Context, domain.MBID) (domain.Artist, error) {
	return domain.Artist{}, nil
}

func (f *fakeCatalog) ListReleaseGroups(_ context.Context, a domain.MBID) ([]domain.ReleaseGroup, error) {
	return f.groups[a], f.err
}

func rg(mbid domain.MBID, date string, credits ...domain.CreditName) domain.ReleaseGroup {
	return domain.ReleaseGroup{
		MBID: mbid, Title: string(mbid), PrimaryType: domain.PrimaryAlbum,
		FirstRelease: domain.ParseReleaseDate(date), Credits: credits,
	}
}

func solo(a domain.MBID) domain.CreditName { return domain.CreditName{ArtistMBID: a, Name: "A"} }

func newSyncArtist(store *memory.Store, catalog *fakeCatalog) *Artist {
	clock := &memory.Clock{T: now}
	fanout := notify.NewFanOut(store.SettingsRepo(), store.SubscriptionRepo(), store.NotificationRepo(), clock)
	return NewArtist(catalog, store.ArtistRepo(), store.SubscriptionRepo(), store.ReleaseRepo(), store.RunRepo(),
		fanout, memory.TxManager{}, clock, Config{StaleThreshold: 90 * 24 * time.Hour})
}

func TestSyncArtist_BaselineThenNews(t *testing.T) {
	store := memory.NewStore()
	store.Artists[artistA] = domain.Artist{MBID: artistA}
	store.Subs[artistA] = domain.Subscription{BaselineAt: now.Add(-time.Hour)}
	catalog := &fakeCatalog{groups: map[domain.MBID][]domain.ReleaseGroup{
		artistA: {rg(rg1, "2001", solo(artistA))},
	}}
	uc := newSyncArtist(store, catalog)

	res, err := uc.Execute(context.Background(), artistA, "")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Baseline || res.Discovered != 1 || res.New != 0 {
		t.Errorf("baseline result = %+v", res)
	}
	if l := store.Links[artistA]; len(l) != 1 || !l[0].IsBaseline || l[0].Role != domain.RoleMain {
		t.Errorf("baseline links = %+v", l)
	}
	if store.Artists[artistA].LastSyncedAt == nil {
		t.Error("artist not marked as synced")
	}

	// Second sync: a fresh single, an old album found late, a featuring.
	catalog.groups[artistA] = append(catalog.groups[artistA],
		rg(rg2, "2026-10-10", solo(artistA)),
		rg(rg3, "2019", domain.CreditName{ArtistMBID: artistB, JoinPhrase: " feat. "}, solo(artistA)),
	)
	res, err = uc.Execute(context.Background(), artistA, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Baseline || res.Discovered != 2 || res.New != 1 {
		t.Errorf("second result = %+v, want 2 discovered, 1 new (the old one is stale)", res)
	}
	roles := map[domain.MBID]domain.CreditRole{}
	for _, l := range store.Links[artistA] {
		roles[l.ReleaseGroup.MBID] = l.Role
		if l.ReleaseGroup.MBID != rg1 && l.IsBaseline {
			t.Errorf("%s must not be baseline", l.ReleaseGroup.MBID)
		}
	}
	if roles[rg3] != domain.RoleFeatured {
		t.Errorf("role of rg3 = %s, want featured", roles[rg3])
	}
	if len(store.Notifs) != 1 || store.Notifs[0].ReleaseGroup != rg2 || store.Notifs[0].Channel != domain.ChannelRSS {
		t.Errorf("notifications = %+v, want one rss entry for rg2", store.Notifs)
	}
}

func TestSyncArtist_ResubscriptionIsBaseline(t *testing.T) {
	store := memory.NewStore()
	lastSync := now.Add(-30 * 24 * time.Hour)
	store.Artists[artistA] = domain.Artist{MBID: artistA, LastSyncedAt: &lastSync}
	// Unsubscribed for a while, subscribed again a minute ago.
	store.Subs[artistA] = domain.Subscription{BaselineAt: now.Add(-time.Minute)}
	catalog := &fakeCatalog{groups: map[domain.MBID][]domain.ReleaseGroup{
		artistA: {rg(rg1, "2025-01-01", solo(artistA))},
	}}

	res, err := newSyncArtist(store, catalog).Execute(context.Background(), artistA, "")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Baseline || res.New != 0 || len(store.Notifs) != 0 {
		t.Errorf("result = %+v, notifications = %d; an old release from the unsubscribed period is history", res, len(store.Notifs))
	}
}

func TestSyncArtist_BaselineNotifiesReleasesInsideStaleWindow(t *testing.T) {
	store := memory.NewStore()
	store.Artists[artistA] = domain.Artist{MBID: artistA}
	store.Subs[artistA] = domain.Subscription{BaselineAt: now.Add(-time.Hour)}
	// A recent album, an old one, an undated one.
	catalog := &fakeCatalog{groups: map[domain.MBID][]domain.ReleaseGroup{
		artistA: {rg(rg1, "2026-10-02", solo(artistA)), rg(rg2, "2024", solo(artistA)), rg(rg3, "", solo(artistA))},
	}}

	res, err := newSyncArtist(store, catalog).Execute(context.Background(), artistA, "")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Baseline || res.Discovered != 3 || res.New != 1 {
		t.Errorf("result = %+v, want 3 discovered, 1 new", res)
	}
	for _, l := range store.Links[artistA] {
		if want := l.ReleaseGroup.MBID != rg1; l.IsBaseline != want {
			t.Errorf("%s: is_baseline = %v, want %v", l.ReleaseGroup.MBID, l.IsBaseline, want)
		}
	}
	if len(store.Notifs) != 1 || store.Notifs[0].ReleaseGroup != rg1 {
		t.Errorf("notifications = %+v, want one entry for rg1", store.Notifs)
	}
}

func TestSyncArtist_CatalogErrorWritesNothing(t *testing.T) {
	store := memory.NewStore()
	store.Artists[artistA] = domain.Artist{MBID: artistA}
	uc := newSyncArtist(store, &fakeCatalog{err: domain.ErrUpstreamUnavailable})

	_, err := uc.Execute(context.Background(), artistA, "")
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("error = %v", err)
	}
	if store.Artists[artistA].LastSyncedAt != nil || len(store.Releases) != 0 {
		t.Error("nothing must be stored after a failed fetch")
	}
}

func payload(t *testing.T, p usecase.SyncArtistPayload) []byte {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSyncArtist_HandleJob(t *testing.T) {
	t.Run("advances the run", func(t *testing.T) {
		store := memory.NewStore()
		store.Artists[artistA] = domain.Artist{MBID: artistA}
		run, _ := store.RunRepo().Create(context.Background(), domain.TriggerManual, 1, now)
		uc := newSyncArtist(store, &fakeCatalog{})

		if err := uc.HandleJob(context.Background(), payload(t, usecase.SyncArtistPayload{ArtistMBID: string(artistA), RunID: run.ID}), false); err != nil {
			t.Fatal(err)
		}
		if got := store.Runs[0]; got.ArtistsDone != 1 || !got.Finished() {
			t.Errorf("run = %+v, want finished", got)
		}
	})

	t.Run("transient failure is retried, the last attempt still counts", func(t *testing.T) {
		store := memory.NewStore()
		store.Artists[artistA] = domain.Artist{MBID: artistA}
		run, _ := store.RunRepo().Create(context.Background(), domain.TriggerManual, 1, now)
		uc := newSyncArtist(store, &fakeCatalog{err: domain.ErrUpstreamUnavailable})
		p := payload(t, usecase.SyncArtistPayload{ArtistMBID: string(artistA), RunID: run.ID})

		var permanent *usecase.PermanentError
		if err := uc.HandleJob(context.Background(), p, false); err == nil || errors.As(err, &permanent) {
			t.Errorf("error = %v, want a retryable error", err)
		}
		if store.Runs[0].ArtistsDone != 0 {
			t.Error("a retried job must not advance the run")
		}
		_ = uc.HandleJob(context.Background(), p, true)
		if !store.Runs[0].Finished() {
			t.Error("the last attempt must advance the run")
		}
	})

	t.Run("unknown artist is permanent", func(t *testing.T) {
		store := memory.NewStore()
		uc := newSyncArtist(store, &fakeCatalog{})
		err := uc.HandleJob(context.Background(), payload(t, usecase.SyncArtistPayload{ArtistMBID: string(artistA)}), false)
		var permanent *usecase.PermanentError
		if !errors.As(err, &permanent) || !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("error = %v, want permanent ErrNotFound", err)
		}
	})

	t.Run("bad payload is permanent", func(t *testing.T) {
		uc := newSyncArtist(memory.NewStore(), &fakeCatalog{})
		var permanent *usecase.PermanentError
		if err := uc.HandleJob(context.Background(), []byte(`{"artist_mbid":"x"}`), false); !errors.As(err, &permanent) {
			t.Errorf("error = %v, want permanent", err)
		}
	})
}

func newSchedule(store *memory.Store, at time.Time) *ScheduleSyncRun {
	return NewScheduleSyncRun(store.ArtistRepo(), store.RunRepo(), store.Queue(), memory.TxManager{}, &memory.Clock{T: at})
}

func TestScheduleSyncRun(t *testing.T) {
	store := memory.NewStore()
	store.Subs[artistA] = domain.Subscription{}
	store.Subs[artistB] = domain.Subscription{}

	run, started, err := newSchedule(store, now).Execute(context.Background(), domain.TriggerManual)
	if err != nil || !started {
		t.Fatalf("started = %t, err = %v", started, err)
	}
	if run.ArtistsTotal != 2 || len(store.Jobs) != 2 {
		t.Errorf("run = %+v, jobs = %d", run, len(store.Jobs))
	}
	if p := store.Jobs[0].Payload.(usecase.SyncArtistPayload); p.RunID != run.ID {
		t.Errorf("job payload = %+v", p)
	}

	again, started, err := newSchedule(store, now.Add(time.Minute)).Execute(context.Background(), domain.TriggerSchedule)
	if err != nil || started || again.ID != run.ID {
		t.Errorf("second call: run %s started %t err %v, want the running one", again.ID, started, err)
	}

	_, started, _ = newSchedule(store, now.Add(13*time.Hour)).Execute(context.Background(), domain.TriggerSchedule)
	if !started {
		t.Error("an abandoned run must not block a new one")
	}
}

func TestScheduleSyncRun_NoArtists(t *testing.T) {
	store := memory.NewStore()
	run, started, err := newSchedule(store, now).Execute(context.Background(), domain.TriggerSchedule)
	if err != nil || !started || !run.Finished() {
		t.Errorf("run = %+v, started %t, err %v; want an already finished run", run, started, err)
	}
}

func TestRequestArtist(t *testing.T) {
	store := memory.NewStore()
	store.Subs[artistA] = domain.Subscription{}
	uc := NewRequestArtist(store.SubscriptionRepo(), store.Queue(), &memory.Clock{T: now})

	for range 2 {
		if err := uc.Execute(context.Background(), string(artistA)); err != nil {
			t.Fatal(err)
		}
	}
	if len(store.Jobs) != 1 {
		t.Errorf("jobs = %d, want 1 (deduplicated)", len(store.Jobs))
	}
	if err := uc.Execute(context.Background(), string(artistB)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("not subscribed: %v, want ErrNotFound", err)
	}
}

func TestLatestSyncRun(t *testing.T) {
	store := memory.NewStore()
	uc := NewLatestSyncRun(store.RunRepo())
	if run, err := uc.Execute(context.Background()); run != nil || err != nil {
		t.Errorf("empty: %v, %v", run, err)
	}
	_, _ = store.RunRepo().Create(context.Background(), domain.TriggerManual, 3, now)
	if run, _ := uc.Execute(context.Background()); run == nil || run.ArtistsTotal != 3 {
		t.Errorf("latest = %+v", run)
	}
}
