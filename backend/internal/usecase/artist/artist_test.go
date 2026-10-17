package artist

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
)

func TestSearchArtists_MarksSubscribed(t *testing.T) {
	store := newMemStore()
	store.subs[nirvana] = domain.Subscription{}
	catalog := &fakeCatalog{found: []domain.ArtistCandidate{
		{Artist: domain.Artist{MBID: nirvana, Name: "Nirvana"}, Score: 100},
		{Artist: domain.Artist{MBID: radiohead, Name: "Radiohead"}, Score: 40},
	}}

	got, err := NewSearchArtists(catalog, store).Execute(context.Background(), SearchInput{Query: "  nirvana "})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[0].Subscribed || got[1].Subscribed {
		t.Errorf("subscribed flags = %+v", got)
	}
	if catalog.lastLimit != DefaultSearchLimit {
		t.Errorf("limit = %d, want default %d", catalog.lastLimit, DefaultSearchLimit)
	}
}

func TestSearchArtists_Validation(t *testing.T) {
	tests := []struct {
		name string
		in   SearchInput
	}{
		{"empty query", SearchInput{Query: "   "}},
		{"query too long", SearchInput{Query: strings.Repeat("я", maxQueryLength+1)}},
		{"limit too big", SearchInput{Query: "x", Limit: MaxSearchLimit + 1}},
		{"negative limit", SearchInput{Query: "x", Limit: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSearchArtists(&fakeCatalog{}, newMemStore()).Execute(context.Background(), tt.in)
			if !errors.Is(err, domain.ErrValidation) {
				t.Errorf("error = %v, want ErrValidation", err)
			}
		})
	}
}

func TestSearchArtists_UpstreamErrorIsWrapped(t *testing.T) {
	catalog := &fakeCatalog{err: domain.ErrUpstreamUnavailable}
	_, err := NewSearchArtists(catalog, newMemStore()).Execute(context.Background(), SearchInput{Query: "x"})
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("error = %v, want ErrUpstreamUnavailable", err)
	}
}

func newSubscribe(catalog *fakeCatalog, store *memStore, now time.Time) *Subscribe {
	return NewSubscribe(catalog, store, store, defaultFilter, noTx{}, fixedClock(now))
}

func TestSubscribe_CreatesWithBaselineAndOverride(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	store := newMemStore()
	catalog := &fakeCatalog{artists: map[domain.MBID]domain.Artist{
		nirvana: {MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"},
	}}
	on := true

	res, err := newSubscribe(catalog, store, now).Execute(context.Background(), SubscribeInput{
		MBID:     strings.ToUpper(string(nirvana)),
		Override: domain.FilterOverride{IncludeFeatured: &on},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created {
		t.Error("Created = false, want true")
	}
	if res.Artist.Name != "Nirvana" || !res.BaselineAt.Equal(now) {
		t.Errorf("subscription = %+v", res.Subscription)
	}
	if f := res.Filter; !f.IncludeFeatured.Overridden || !f.IncludeFeatured.Value || f.PrimaryTypes.Overridden {
		t.Errorf("filter = %+v", f)
	}
	if _, ok := store.artists[nirvana]; !ok {
		t.Error("artist was not stored")
	}
}

func TestSubscribe_RepeatIsIdempotentAndSkipsCatalog(t *testing.T) {
	store := newMemStore()
	catalog := &fakeCatalog{artists: map[domain.MBID]domain.Artist{nirvana: {MBID: nirvana, Name: "Nirvana"}}}
	uc := newSubscribe(catalog, store, time.Now())

	if _, err := uc.Execute(context.Background(), SubscribeInput{MBID: string(nirvana)}); err != nil {
		t.Fatal(err)
	}
	res, err := uc.Execute(context.Background(), SubscribeInput{MBID: string(nirvana)})
	if err != nil {
		t.Fatal(err)
	}
	if res.Created {
		t.Error("second subscribe: Created = true, want false")
	}
	if catalog.getCalls != 1 {
		t.Errorf("catalog calls = %d, want 1", catalog.getCalls)
	}
}

func TestSubscribe_Errors(t *testing.T) {
	tests := []struct {
		name    string
		in      SubscribeInput
		catalog *fakeCatalog
		want    error
	}{
		{"bad mbid", SubscribeInput{MBID: "nope"}, &fakeCatalog{}, domain.ErrValidation},
		{"bad override", SubscribeInput{MBID: string(nirvana), Override: domain.FilterOverride{
			PrimaryTypes: []domain.PrimaryType{"lp"},
		}}, &fakeCatalog{}, domain.ErrValidation},
		{"unknown artist", SubscribeInput{MBID: string(nirvana)}, &fakeCatalog{}, domain.ErrNotFound},
		{"musicbrainz down", SubscribeInput{MBID: string(nirvana)}, &fakeCatalog{err: domain.ErrUpstreamUnavailable}, domain.ErrUpstreamUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newSubscribe(tt.catalog, newMemStore(), time.Now()).Execute(context.Background(), tt.in)
			if !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestUnsubscribe(t *testing.T) {
	store := newMemStore()
	store.subs[nirvana] = domain.Subscription{}
	uc := NewUnsubscribe(store)

	if err := uc.Execute(context.Background(), string(nirvana)); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.subs[nirvana]; ok {
		t.Error("subscription still exists")
	}
	if err := uc.Execute(context.Background(), string(nirvana)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("repeat: error = %v, want ErrNotFound", err)
	}
	if err := uc.Execute(context.Background(), "nope"); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("bad mbid: error = %v, want ErrValidation", err)
	}
}

func TestListSubscriptions(t *testing.T) {
	store := newMemStore()
	for _, a := range []domain.Artist{
		{MBID: radiohead, Name: "Radiohead", SortName: "Radiohead"},
		{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"},
	} {
		store.artists[a.MBID] = a
		store.subs[a.MBID] = domain.Subscription{}
	}
	uc := NewListSubscriptions(store, defaultFilter)

	res, err := uc.Execute(context.Background(), ListInput{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 || len(res.Items) != 1 || res.Items[0].Artist.Name != "Nirvana" {
		t.Errorf("page = %+v", res)
	}
	if got := res.Items[0].Filter.PrimaryTypes; got.Overridden || len(got.Value) != 3 {
		t.Errorf("filter should fall back to defaults: %+v", got)
	}

	if _, err := uc.Execute(context.Background(), ListInput{Offset: -1}); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("negative offset: error = %v, want ErrValidation", err)
	}
}
