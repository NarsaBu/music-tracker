//go:build integration

package postgres

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

const (
	nirvana   domain.MBID = "5b11f4ce-a62d-471e-81fc-a69a8278c7da"
	radiohead domain.MBID = "a74b1b7f-71a5-4011-9441-d0b5e4122711"
	sigurRos  domain.MBID = "f6f2326f-6b25-4170-b89d-e235b25508e8"
)

func seedSubscription(t *testing.T, a domain.Artist, o domain.FilterOverride) {
	t.Helper()
	ctx := context.Background()
	if err := NewArtistRepository(testPool).Upsert(ctx, a); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	created, err := NewSubscriptionRepository(testPool).Create(ctx, domain.Subscription{
		Artist: a, Override: o, BaselineAt: now, CreatedAt: now,
	})
	if err != nil || !created {
		t.Fatalf("create subscription %s: created=%t err=%v", a.Name, created, err)
	}
}

func TestSubscriptionRepository_RoundTrip(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	repo := NewSubscriptionRepository(testPool)
	on := true
	override := domain.FilterOverride{
		PrimaryTypes:           []domain.PrimaryType{domain.PrimaryAlbum, domain.PrimaryEP},
		ExcludedSecondaryTypes: []string{},
		IncludeFeatured:        &on,
	}
	seedSubscription(t, domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana", Type: "Group", Country: "US"}, override)

	got, err := repo.Get(ctx, nirvana)
	if err != nil {
		t.Fatal(err)
	}
	if got.Artist.Name != "Nirvana" || got.Artist.Country != "US" || got.Artist.LastSyncedAt != nil {
		t.Errorf("artist = %+v", got.Artist)
	}
	if got.CreatedAt.Location() != time.UTC {
		t.Errorf("created_at zone = %v, want UTC", got.CreatedAt.Location())
	}
	if !slices.Equal(got.Override.PrimaryTypes, override.PrimaryTypes) {
		t.Errorf("primary types = %v", got.Override.PrimaryTypes)
	}
	if got.Override.ExcludedSecondaryTypes == nil {
		t.Error("an empty exclusion list must stay an override, not become NULL")
	}
	if got.Override.IncludeFeatured == nil || !*got.Override.IncludeFeatured {
		t.Errorf("include_featured = %v", got.Override.IncludeFeatured)
	}

	created, err := repo.Create(ctx, got)
	if err != nil || created {
		t.Errorf("duplicate create: created=%t err=%v, want false, nil", created, err)
	}

	if err := repo.Delete(ctx, nirvana); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, nirvana); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("get after delete: %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, nirvana); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second delete: %v, want ErrNotFound", err)
	}
}

func TestSubscriptionRepository_NullOverrides(t *testing.T) {
	resetTables(t)
	seedSubscription(t, domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"}, domain.FilterOverride{})

	got, err := NewSubscriptionRepository(testPool).Get(context.Background(), nirvana)
	if err != nil {
		t.Fatal(err)
	}
	o := got.Override
	if o.PrimaryTypes != nil || o.ExcludedSecondaryTypes != nil || o.IncludeFeatured != nil {
		t.Errorf("override = %+v, want all nil", o)
	}
	if got.Artist.Type != "" || got.Artist.Country != "" {
		t.Errorf("empty type/country should round-trip as empty: %+v", got.Artist)
	}
}

func TestSubscriptionRepository_ListSearchAndPaging(t *testing.T) {
	resetTables(t)
	for _, a := range []domain.Artist{
		{MBID: radiohead, Name: "Radiohead", SortName: "Radiohead"},
		{MBID: sigurRos, Name: "Sigur Rós", SortName: "Sigur Rós"},
		{MBID: nirvana, Name: "Nirvana", SortName: "nirvana"},
	} {
		seedSubscription(t, a, domain.FilterOverride{})
	}
	repo := NewSubscriptionRepository(testPool)
	ctx := context.Background()

	names := func(subs []domain.Subscription) []string {
		var out []string
		for _, s := range subs {
			out = append(out, s.Artist.Name)
		}
		return out
	}

	tests := []struct {
		name      string
		q         usecase.SubscriptionListQuery
		want      []string
		wantTotal int
	}{
		{"all, case-insensitive sort", usecase.SubscriptionListQuery{Limit: 10}, []string{"Nirvana", "Radiohead", "Sigur Rós"}, 3},
		{"page", usecase.SubscriptionListQuery{Limit: 1, Offset: 1}, []string{"Radiohead"}, 3},
		{"search is case-insensitive", usecase.SubscriptionListQuery{Search: "RÓS", Limit: 10}, []string{"Sigur Rós"}, 1},
		{"LIKE wildcards are literal", usecase.SubscriptionListQuery{Search: "%", Limit: 10}, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := repo.List(ctx, tt.q)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(names(got), tt.want) || total != tt.wantTotal {
				t.Errorf("got %v (total %d), want %v (total %d)", names(got), total, tt.want, tt.wantTotal)
			}
		})
	}
}

func TestSubscriptionRepository_SubscribedAmong(t *testing.T) {
	resetTables(t)
	seedSubscription(t, domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"}, domain.FilterOverride{})

	got, err := NewSubscriptionRepository(testPool).SubscribedAmong(context.Background(), []domain.MBID{nirvana, radiohead})
	if err != nil {
		t.Fatal(err)
	}
	if !got[nirvana] || got[radiohead] || len(got) != 1 {
		t.Errorf("subscribed = %v", got)
	}
}

func TestArtistRepository_UpsertKeepsSyncState(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	repo := NewArtistRepository(testPool)
	if err := repo.Upsert(ctx, domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"}); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(ctx, "UPDATE artists SET last_synced_at = now() WHERE mbid = $1", string(nirvana)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(ctx, domain.Artist{MBID: nirvana, Name: "Nirvana (renamed)", SortName: "Nirvana"}); err != nil {
		t.Fatal(err)
	}

	var name string
	var synced bool
	err := testPool.QueryRow(ctx, "SELECT name, last_synced_at IS NOT NULL FROM artists WHERE mbid = $1", string(nirvana)).Scan(&name, &synced)
	if err != nil {
		t.Fatal(err)
	}
	if name != "Nirvana (renamed)" || !synced {
		t.Errorf("name = %q, synced = %t; want renamed and still synced", name, synced)
	}
}

func TestSettingsRepository_FilterDefaults(t *testing.T) {
	got, err := NewSettingsRepository(testPool).FilterDefaults(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.PrimaryType{domain.PrimaryAlbum, domain.PrimaryEP, domain.PrimarySingle}
	if !slices.Equal(got.PrimaryTypes, want) || got.IncludeFeatured || !slices.Contains(got.ExcludedSecondaryTypes, "Audio drama") {
		t.Errorf("defaults = %+v", got)
	}
}

func TestTxManager_RollsBack(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	boom := errors.New("boom")
	err := NewTxManager(testPool).WithinTx(ctx, func(ctx context.Context) error {
		if err := NewArtistRepository(testPool).Upsert(ctx, domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	var n int
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM artists").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("artists = %d after rollback, want 0", n)
	}
}
