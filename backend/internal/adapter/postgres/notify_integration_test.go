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

func TestSettingsRepository_SaveAndGet(t *testing.T) {
	ctx := context.Background()
	repo := NewSettingsRepository(testPool)
	orig, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Save(ctx, orig) })

	if orig.Email.Digest.Hour != 9 || orig.Email.Digest.Timezone != "UTC" || orig.Email.Digest.Weekday != time.Friday || len(orig.RSS.FeedToken) != 43 {
		t.Errorf("defaults = %+v", orig)
	}

	next := time.Date(2026, 10, 23, 6, 30, 0, 0, time.UTC)
	s := orig
	s.Filter = domain.ReleaseFilter{PrimaryTypes: []domain.PrimaryType{domain.PrimaryAlbum}, ExcludedSecondaryTypes: []string{}, IncludeFeatured: true}
	s.NotifyAnnouncements = false
	s.Email = domain.EmailSettings{Enabled: true, To: "me@example.com", Digest: domain.DigestSchedule{
		Frequency: domain.DigestWeekly, Weekday: time.Monday, Hour: 7, Minute: 45, Timezone: "Europe/Moscow",
	}}
	s.RSS.Enabled = false
	s.NextDigestAt = &next
	if err := repo.Save(ctx, s); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != s.Email || got.NotifyAnnouncements || got.RSS.Enabled || !got.Filter.IncludeFeatured ||
		got.Filter.ExcludedSecondaryTypes == nil || got.NextDigestAt == nil || !got.NextDigestAt.Equal(next) {
		t.Errorf("got %+v", got)
	}

	last := next.Add(time.Minute)
	if err := repo.SetDigestTimes(ctx, &last, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetDigestTimes(ctx, nil, &next); err != nil { // nil last keeps the previous one
		t.Fatal(err)
	}
	if got, _ = repo.Get(ctx); got.LastDigestAt == nil || !got.LastDigestAt.Equal(last) || !got.NextDigestAt.Equal(next) {
		t.Errorf("digest times = %v / %v", got.LastDigestAt, got.NextDigestAt)
	}

	if err := repo.SetFeedToken(ctx, "new-token"); err != nil {
		t.Fatal(err)
	}
	if got, _ = repo.Get(ctx); got.RSS.FeedToken != "new-token" {
		t.Errorf("token = %q", got.RSS.FeedToken)
	}
}

// seedReleases stores artists and release groups the notification tests point at.
func seedReleases(t *testing.T, at time.Time) {
	t.Helper()
	ctx := context.Background()
	for _, a := range []domain.Artist{
		{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"},
		{MBID: radiohead, Name: "Radiohead", SortName: "Radiohead"},
	} {
		if err := NewArtistRepository(testPool).Upsert(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	rgs := []domain.ReleaseGroup{
		{MBID: rgOld, Title: "Album", PrimaryType: domain.PrimaryAlbum, ArtistCredit: "Nirvana", FirstRelease: domain.ParseReleaseDate("1991-09-24")},
		{MBID: rgNew, Title: "Single", PrimaryType: domain.PrimarySingle, ArtistCredit: "Radiohead feat. Nirvana", FirstRelease: domain.ParseReleaseDate("2026-10")},
		{MBID: rgNoDay, Title: "EP", PrimaryType: domain.PrimaryEP, ArtistCredit: "Nirvana"},
	}
	if err := NewReleaseGroupRepository(testPool).UpsertMany(ctx, rgs, at); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationRepository_InsertDedupAndFoundVia(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)
	seedReleases(t, at)
	repo := NewNotificationRepository(testPool)

	insert := func(artist domain.MBID, role domain.CreditRole) {
		t.Helper()
		err := repo.Insert(ctx, domain.Notification{ReleaseGroup: rgNew, Artist: artist, Role: role, Channel: domain.ChannelRSS, Status: domain.StatusSent, At: at})
		if err != nil {
			t.Fatal(err)
		}
	}
	insert(nirvana, domain.RoleFeatured)
	insert(radiohead, domain.RoleMain)   // promotes found-via to the main artist
	insert(nirvana, domain.RoleFeatured) // does not demote it back

	items, err := repo.ListFeed(ctx, usecase.FeedQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].FoundVia.MBID != radiohead || items[0].Role != domain.RoleMain || items[0].ReleaseGroup.Title != "Single" {
		t.Errorf("items = %+v", items)
	}
}

func TestNotificationRepository_EmailLifecycle(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)
	seedReleases(t, at)
	repo := NewNotificationRepository(testPool)
	for _, rg := range []domain.MBID{rgOld, rgNew} {
		if err := repo.Insert(ctx, domain.Notification{ReleaseGroup: rg, Artist: nirvana, Role: domain.RoleMain, Channel: domain.ChannelEmail, Status: domain.StatusPending, At: at}); err != nil {
			t.Fatal(err)
		}
	}

	// A digest in progress locks its items: a concurrent one sees nothing.
	tx := NewTxManager(testPool)
	var ids []string
	err := tx.WithinTx(ctx, func(ctx context.Context) error {
		items, err := repo.ClaimPendingEmail(ctx)
		if err != nil || len(items) != 2 {
			t.Fatalf("claimed %d, %v", len(items), err)
		}
		for _, it := range items {
			ids = append(ids, it.ID)
		}
		other, err := repo.ClaimPendingEmail(context.Background()) // outside the transaction
		if err != nil || len(other) != 0 {
			t.Errorf("concurrent claim got %d, %v; want 0", len(other), err)
		}
		return repo.MarkFailed(ctx, ids, errors.New("smtp down"))
	})
	if err != nil {
		t.Fatal(err)
	}

	for range domain.MaxEmailAttempts - 1 {
		if err := repo.MarkFailed(ctx, ids[:1], errors.New("smtp down")); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.MarkSent(ctx, ids[1:], at); err != nil {
		t.Fatal(err)
	}
	left, err := repo.ClaimPendingEmail(ctx)
	if err != nil || len(left) != 0 {
		t.Errorf("pending = %d, %v; want none (one failed for good, one sent)", len(left), err)
	}
	var failed int
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE status = 'failed' AND attempts = $1", domain.MaxEmailAttempts).Scan(&failed); err != nil || failed != 1 {
		t.Errorf("failed = %d, %v", failed, err)
	}
}

func TestNotificationRepository_FeedPagingAndFilters(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)
	seedReleases(t, at)
	repo := NewNotificationRepository(testPool)
	// rgOld and rgNew share a timestamp: the id must break the tie without losing rows.
	for _, n := range []domain.Notification{
		{ReleaseGroup: rgOld, Artist: nirvana, Role: domain.RoleMain, At: at},
		{ReleaseGroup: rgNew, Artist: nirvana, Role: domain.RoleFeatured, At: at},
		{ReleaseGroup: rgNoDay, Artist: nirvana, Role: domain.RoleMain, At: at.Add(time.Hour)},
	} {
		n.Channel, n.Status = domain.ChannelRSS, domain.StatusSent
		if err := repo.Insert(ctx, n); err != nil {
			t.Fatal(err)
		}
	}

	var titles []string
	var before *usecase.FeedCursor
	for range 5 {
		page, err := repo.ListFeed(ctx, usecase.FeedQuery{Limit: 1, Before: before})
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		titles = append(titles, page[0].ReleaseGroup.Title)
		before = &usecase.FeedCursor{At: page[0].CreatedAt, ID: page[0].ID}
	}
	if len(titles) != 3 || titles[0] != "EP" {
		t.Errorf("pages = %v, want 3 items, newest (EP) first", titles)
	}

	// By release date: newest first, the undated EP last.
	titles, before = nil, nil
	for range 5 {
		page, err := repo.ListFeed(ctx, usecase.FeedQuery{Sort: usecase.FeedByRelease, Limit: 1, Before: before})
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		titles = append(titles, page[0].ReleaseGroup.Title)
		before = &usecase.FeedCursor{ID: page[0].ID}
		if d := page[0].ReleaseGroup.FirstRelease.Day; d != nil {
			before.At = *d
		}
	}
	if want := []string{"Single", "Album", "EP"}; !slices.Equal(titles, want) {
		t.Errorf("pages by release = %v, want %v", titles, want)
	}

	singles, _ := repo.ListFeed(ctx, usecase.FeedQuery{Limit: 10, Type: domain.PrimarySingle})
	featured, _ := repo.ListFeed(ctx, usecase.FeedQuery{Limit: 10, Role: domain.RoleFeatured})
	if len(singles) != 1 || len(featured) != 1 || singles[0].ID != featured[0].ID {
		t.Errorf("singles = %d, featured = %d", len(singles), len(featured))
	}
}

func TestSubscriptionRepository_UpdateOverride(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	seedSubscription(t, domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"}, domain.FilterOverride{})
	repo := NewSubscriptionRepository(testPool)
	off := false
	o := domain.FilterOverride{PrimaryTypes: []domain.PrimaryType{domain.PrimaryEP}, IncludeFeatured: &off}
	if err := repo.UpdateOverride(ctx, nirvana, o); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.Get(ctx, nirvana)
	if len(got.Override.PrimaryTypes) != 1 || got.Override.ExcludedSecondaryTypes != nil || got.Override.IncludeFeatured == nil || *got.Override.IncludeFeatured {
		t.Errorf("override = %+v", got.Override)
	}
	if err := repo.UpdateOverride(ctx, radiohead, o); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("not subscribed: %v", err)
	}
}
