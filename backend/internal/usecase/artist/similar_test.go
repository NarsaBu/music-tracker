package artist

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/memory"
)

type fakeSimilarSource struct {
	items []domain.SimilarArtist
	err   error
	calls int
}

func (f *fakeSimilarSource) SimilarArtists(context.Context, domain.MBID) ([]domain.SimilarArtist, error) {
	f.calls++
	return f.items, f.err
}

type cachedSimilar struct {
	items []domain.SimilarArtist
	at    time.Time
}

type fakeSimilarCache map[domain.MBID]cachedSimilar

func (c fakeSimilarCache) Get(_ context.Context, a domain.MBID) ([]domain.SimilarArtist, time.Time, error) {
	e, ok := c[a]
	if !ok {
		return nil, time.Time{}, domain.ErrNotFound
	}
	return e.items, e.at, nil
}

func (c fakeSimilarCache) Put(_ context.Context, a domain.MBID, items []domain.SimilarArtist, at time.Time) error {
	c[a] = cachedSimilar{items, at}
	return nil
}

var similarNow = time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

func newListSimilar(source *fakeSimilarSource, cache fakeSimilarCache, store *memory.Store) *ListSimilar {
	return NewListSimilar(source, cache, store.SubscriptionRepo(), &memory.Clock{T: similarNow})
}

func TestListSimilar_FetchesCachesAndMarksSubscribed(t *testing.T) {
	store := memory.NewStore()
	store.Subs[radiohead] = domain.Subscription{}
	source := &fakeSimilarSource{items: []domain.SimilarArtist{
		{MBID: radiohead, Name: "Radiohead", Score: 900},
		{MBID: nirvana, Name: "Nirvana", Score: 800}, // the artist itself is dropped
	}}
	cache := fakeSimilarCache{}
	uc := newListSimilar(source, cache, store)

	got, err := uc.Execute(context.Background(), string(nirvana))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].MBID != radiohead || !got[0].Subscribed {
		t.Errorf("got %+v, want only Radiohead, subscribed", got)
	}
	if e := cache[nirvana]; len(e.items) != 2 || !e.at.Equal(similarNow) {
		t.Errorf("cache = %+v, want the full answer at now", e)
	}

	// A fresh cache answers without the source.
	if _, err := uc.Execute(context.Background(), string(nirvana)); err != nil || source.calls != 1 {
		t.Errorf("second call: err = %v, source calls = %d, want 1", err, source.calls)
	}
}

func TestListSimilar_RefreshesStaleCache(t *testing.T) {
	cache := fakeSimilarCache{nirvana: {
		items: []domain.SimilarArtist{{MBID: radiohead, Name: "Old"}},
		at:    similarNow.Add(-SimilarTTL - time.Minute),
	}}
	source := &fakeSimilarSource{items: []domain.SimilarArtist{{MBID: radiohead, Name: "New"}}}

	got, err := newListSimilar(source, cache, memory.NewStore()).Execute(context.Background(), string(nirvana))
	if err != nil || len(got) != 1 || got[0].Name != "New" || !cache[nirvana].at.Equal(similarNow) {
		t.Errorf("got %+v, err = %v; want the refreshed answer cached", got, err)
	}
}

func TestListSimilar_SourceDown(t *testing.T) {
	down := &fakeSimilarSource{err: fmt.Errorf("%w: labs", domain.ErrUpstreamUnavailable)}

	// Stale cache beats an error.
	cache := fakeSimilarCache{nirvana: {
		items: []domain.SimilarArtist{{MBID: radiohead, Name: "Radiohead"}},
		at:    similarNow.Add(-30 * 24 * time.Hour),
	}}
	got, err := newListSimilar(down, cache, memory.NewStore()).Execute(context.Background(), string(nirvana))
	if err != nil || len(got) != 1 {
		t.Errorf("with stale cache: got %+v, err = %v", got, err)
	}

	// Nothing cached: the error goes up.
	_, err = newListSimilar(down, fakeSimilarCache{}, memory.NewStore()).Execute(context.Background(), string(nirvana))
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("without cache: err = %v, want ErrUpstreamUnavailable", err)
	}
}

func TestListSimilar_LimitsAndValidates(t *testing.T) {
	var many []domain.SimilarArtist
	for i := range SimilarLimit + 5 {
		many = append(many, domain.SimilarArtist{MBID: domain.MBID(fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1))})
	}
	uc := newListSimilar(&fakeSimilarSource{items: many}, fakeSimilarCache{}, memory.NewStore())
	if got, err := uc.Execute(context.Background(), string(nirvana)); err != nil || len(got) != SimilarLimit {
		t.Errorf("got %d items, err = %v, want %d", len(got), err, SimilarLimit)
	}
	if _, err := uc.Execute(context.Background(), "not-an-mbid"); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("bad mbid: %v, want ErrValidation", err)
	}
}
