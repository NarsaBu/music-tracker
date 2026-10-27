package release

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/memory"
)

const (
	okComputer domain.MBID = "b1392450-e666-3926-a536-22c65f834433"
	unknown    domain.MBID = "00000000-0000-0000-0000-000000000009"
)

var now = time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

type fakeSource struct {
	reviews domain.Reviews
	err     error
	calls   int
}

func (f *fakeSource) ReleaseGroupReviews(context.Context, domain.MBID) (domain.Reviews, error) {
	f.calls++
	return f.reviews, f.err
}

type cached struct {
	reviews domain.Reviews
	at      time.Time
}

type fakeCache map[domain.MBID]cached

func (c fakeCache) Get(_ context.Context, rg domain.MBID) (domain.Reviews, time.Time, error) {
	e, ok := c[rg]
	if !ok {
		return domain.Reviews{}, time.Time{}, domain.ErrNotFound
	}
	return e.reviews, e.at, nil
}

func (c fakeCache) Put(_ context.Context, rg domain.MBID, r domain.Reviews, at time.Time) error {
	c[rg] = cached{r, at}
	return nil
}

func storeWithRelease() *memory.Store {
	s := memory.NewStore()
	s.Releases[okComputer] = domain.ReleaseGroup{
		MBID: okComputer, Title: "OK Computer", Rating: domain.Rating{Value: 4.55, Votes: 89},
	}
	return s
}

func reviews(author string) domain.Reviews {
	return domain.Reviews{Items: []domain.Review{{ID: "r1", Author: author, Rating: 5}}, Total: 1, Average: 5, Rated: 1}
}

func TestGetReleaseGroup(t *testing.T) {
	uc := NewGetReleaseGroup(storeWithRelease().ReleaseRepo())
	rg, err := uc.Execute(context.Background(), string(okComputer))
	if err != nil || rg.Rating.Votes != 89 {
		t.Errorf("rg = %+v, err = %v", rg, err)
	}
	if _, err := uc.Execute(context.Background(), string(unknown)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown: %v, want ErrNotFound", err)
	}
	if _, err := uc.Execute(context.Background(), "nope"); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("bad mbid: %v, want ErrValidation", err)
	}
}

func TestListReviews_FetchesThenServesCache(t *testing.T) {
	source := &fakeSource{reviews: reviews("listener")}
	cache := fakeCache{}
	uc := NewListReviews(storeWithRelease().ReleaseRepo(), source, cache, &memory.Clock{T: now})

	for range 2 {
		got, err := uc.Execute(context.Background(), string(okComputer))
		if err != nil || len(got.Items) != 1 || got.Items[0].Author != "listener" {
			t.Fatalf("got %+v, err = %v", got, err)
		}
	}
	if source.calls != 1 || !cache[okComputer].at.Equal(now) {
		t.Errorf("source calls = %d, cache = %+v; want one fetch, cached at now", source.calls, cache[okComputer])
	}
}

func TestListReviews_StaleCache(t *testing.T) {
	old := fakeCache{okComputer: {reviews("old"), now.Add(-ReviewsTTL - time.Minute)}}
	uc := NewListReviews(storeWithRelease().ReleaseRepo(), &fakeSource{reviews: reviews("new")}, old, &memory.Clock{T: now})
	if got, _ := uc.Execute(context.Background(), string(okComputer)); got.Items[0].Author != "new" {
		t.Errorf("stale cache not refreshed: %+v", got)
	}

	down := &fakeSource{err: fmt.Errorf("%w: cb", domain.ErrUpstreamUnavailable)}
	old = fakeCache{okComputer: {reviews("old"), now.Add(-30 * 24 * time.Hour)}}
	uc = NewListReviews(storeWithRelease().ReleaseRepo(), down, old, &memory.Clock{T: now})
	if got, err := uc.Execute(context.Background(), string(okComputer)); err != nil || got.Items[0].Author != "old" {
		t.Errorf("source down with stale cache: %+v, %v", got, err)
	}

	uc = NewListReviews(storeWithRelease().ReleaseRepo(), down, fakeCache{}, &memory.Clock{T: now})
	if _, err := uc.Execute(context.Background(), string(okComputer)); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("source down, no cache: %v, want ErrUpstreamUnavailable", err)
	}
}

func TestListReviews_UnknownReleaseSkipsSource(t *testing.T) {
	source := &fakeSource{}
	uc := NewListReviews(storeWithRelease().ReleaseRepo(), source, fakeCache{}, &memory.Clock{T: now})
	if _, err := uc.Execute(context.Background(), string(unknown)); !errors.Is(err, domain.ErrNotFound) || source.calls != 0 {
		t.Errorf("err = %v, source calls = %d; want ErrNotFound without a fetch", err, source.calls)
	}
}

type fakeCatalog struct {
	details domain.ReleaseDetails
	err     error
	calls   int
}

func (f *fakeCatalog) ReleaseGroupDetails(context.Context, domain.MBID) (domain.ReleaseDetails, error) {
	f.calls++
	return f.details, f.err
}

type cachedDetails struct {
	d  domain.ReleaseDetails
	at time.Time
}

type fakeDetailsCache map[domain.MBID]cachedDetails

func (c fakeDetailsCache) Get(_ context.Context, rg domain.MBID) (domain.ReleaseDetails, time.Time, error) {
	e, ok := c[rg]
	if !ok {
		return domain.ReleaseDetails{}, time.Time{}, domain.ErrNotFound
	}
	return e.d, e.at, nil
}

func (c fakeDetailsCache) Put(_ context.Context, rg domain.MBID, d domain.ReleaseDetails, at time.Time) error {
	c[rg] = cachedDetails{d, at}
	return nil
}

func TestGetDetails_MarksSubscribedArtistsAndCaches(t *testing.T) {
	const radiohead, guest domain.MBID = "a74b1b7f-71a5-4011-9441-d0b5e4122711", "f4fdbb4c-e4b7-47a0-b83b-d91bbfcfa387"
	s := storeWithRelease()
	s.Subs[radiohead] = domain.Subscription{}
	catalog := &fakeCatalog{details: domain.ReleaseDetails{
		Genres:  []string{"alternative rock"},
		Artists: []domain.CreditName{{ArtistMBID: radiohead, Name: "Radiohead"}},
		Guests:  []domain.CreditName{{ArtistMBID: guest, Name: "Guest"}},
	}}
	cache := fakeDetailsCache{}
	uc := NewGetDetails(s.ReleaseRepo(), catalog, cache, s.SubscriptionRepo(), &memory.Clock{T: now})

	for range 2 {
		got, err := uc.Execute(context.Background(), string(okComputer))
		if err != nil {
			t.Fatal(err)
		}
		if !got.Subscribed[radiohead] || got.Subscribed[guest] || got.Genres[0] != "alternative rock" {
			t.Errorf("got %+v", got)
		}
	}
	if catalog.calls != 1 {
		t.Errorf("catalog calls = %d, want 1 (then cached)", catalog.calls)
	}

	down := &fakeCatalog{err: fmt.Errorf("%w: mb", domain.ErrUpstreamUnavailable)}
	uc = NewGetDetails(s.ReleaseRepo(), down, fakeDetailsCache{}, s.SubscriptionRepo(), &memory.Clock{T: now})
	if _, err := uc.Execute(context.Background(), string(okComputer)); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("MusicBrainz down, no cache: %v", err)
	}
	if _, err := uc.Execute(context.Background(), string(unknown)); !errors.Is(err, domain.ErrNotFound) || down.calls != 1 {
		t.Errorf("unknown release: %v, calls = %d; want ErrNotFound without a fetch", err, down.calls)
	}
}
