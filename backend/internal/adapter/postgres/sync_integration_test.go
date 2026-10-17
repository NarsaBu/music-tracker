//go:build integration

package postgres

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

const (
	rgOld   domain.MBID = "00000000-0000-0000-0000-000000000001"
	rgNew   domain.MBID = "00000000-0000-0000-0000-000000000002"
	rgNoDay domain.MBID = "00000000-0000-0000-0000-000000000003"
)

func TestReleaseGroupRepository_RoundTrip(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	if err := NewArtistRepository(testPool).Upsert(ctx, domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"}); err != nil {
		t.Fatal(err)
	}
	repo := NewReleaseGroupRepository(testPool)
	at := time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)
	rgs := []domain.ReleaseGroup{
		{MBID: rgOld, Title: "Bleach", PrimaryType: domain.PrimaryAlbum, FirstRelease: domain.ParseReleaseDate("1989-06"), ArtistCredit: "Nirvana"},
		{MBID: rgNew, Title: "Live", PrimaryType: domain.PrimaryAlbum, SecondaryTypes: []string{"Live"}, FirstRelease: domain.ParseReleaseDate("2026-09-01"), ArtistCredit: "Nirvana feat. X"},
		{MBID: rgNoDay, Title: "Untitled", ArtistCredit: "Nirvana"},
	}
	if err := repo.UpsertMany(ctx, rgs, at); err != nil {
		t.Fatal(err)
	}
	links := []domain.ArtistReleaseGroup{
		{ArtistMBID: nirvana, ReleaseGroup: rgs[0], Role: domain.RoleMain, IsBaseline: true, DiscoveredAt: at},
		{ArtistMBID: nirvana, ReleaseGroup: rgs[1], Role: domain.RoleFeatured, DiscoveredAt: at},
		{ArtistMBID: nirvana, ReleaseGroup: rgs[2], Role: domain.RoleMain, DiscoveredAt: at},
	}
	if err := repo.LinkMany(ctx, links); err != nil {
		t.Fatal(err)
	}
	// Re-linking is a no-op, re-upserting refreshes fields.
	if err := repo.LinkMany(ctx, links[:1]); err != nil {
		t.Fatal(err)
	}
	rgs[0].Title = "Bleach (Deluxe)"
	if err := repo.UpsertMany(ctx, rgs[:1], at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	known, err := repo.KnownMBIDs(ctx, nirvana)
	if err != nil || len(known) != 3 {
		t.Fatalf("known = %v, %v", known, err)
	}

	all, err := repo.ListByArtist(ctx, nirvana, "")
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, l := range all {
		titles = append(titles, l.ReleaseGroup.Title)
	}
	if !slices.Equal(titles, []string{"Untitled", "Live", "Bleach (Deluxe)"}) {
		t.Errorf("order = %v, want undated first, then newest", titles)
	}
	live := all[1]
	if live.Role != domain.RoleFeatured || live.IsBaseline || live.ReleaseGroup.SecondaryTypes[0] != "Live" ||
		live.ReleaseGroup.FirstRelease.Raw != "2026-09-01" || live.ReleaseGroup.ArtistCredit != "Nirvana feat. X" {
		t.Errorf("live = %+v", live)
	}
	if all[0].ReleaseGroup.PrimaryType != "" || all[0].ReleaseGroup.SecondaryTypes == nil {
		t.Errorf("untitled = %+v", all[0].ReleaseGroup)
	}

	featured, err := repo.ListByArtist(ctx, nirvana, domain.RoleFeatured)
	if err != nil || len(featured) != 1 {
		t.Errorf("featured = %d, %v", len(featured), err)
	}
}

func TestArtistRepository_TrackedAndSynced(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	artists := NewArtistRepository(testPool)
	seedSubscription(t, domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"}, domain.FilterOverride{})
	if err := artists.Upsert(ctx, domain.Artist{MBID: radiohead, Name: "Radiohead", SortName: "Radiohead"}); err != nil {
		t.Fatal(err)
	}

	tracked, err := artists.ListTracked(ctx)
	if err != nil || !slices.Equal(tracked, []domain.MBID{nirvana}) {
		t.Errorf("tracked = %v, %v; want only the subscribed artist", tracked, err)
	}

	at := time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)
	if err := artists.MarkSynced(ctx, nirvana, at); err != nil {
		t.Fatal(err)
	}
	a, err := artists.Get(ctx, nirvana)
	if err != nil || a.LastSyncedAt == nil || !a.LastSyncedAt.Equal(at) {
		t.Errorf("artist = %+v, %v", a, err)
	}
	if _, err := artists.Get(ctx, sigurRos); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown artist: %v", err)
	}
}

func TestSyncRunRepository(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	runs := NewSyncRunRepository(testPool)
	at := time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)

	if _, err := runs.Latest(ctx); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("latest of none: %v", err)
	}
	empty, err := runs.Create(ctx, domain.TriggerSchedule, 0, at)
	if err != nil || !empty.Finished() {
		t.Fatalf("empty run = %+v, %v; want finished", empty, err)
	}

	run, err := runs.Create(ctx, domain.TriggerManual, 2, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := runs.Running(ctx, at); err != nil || got.ID != run.ID {
		t.Errorf("running = %+v, %v", got, err)
	}
	if _, err := runs.Running(ctx, at.Add(time.Hour)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("a run started before the cutoff must be ignored: %v", err)
	}

	if err := runs.ArtistDone(ctx, run.ID, 2, at.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := runs.ArtistDone(ctx, run.ID, 1, at.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := runs.ArtistDone(ctx, run.ID, 5, at.Add(4*time.Minute)); err != nil { // late duplicate: ignored
		t.Fatal(err)
	}
	got, err := runs.Latest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != run.ID || got.ArtistsDone != 2 || got.NewReleases != 3 || got.FinishedAt == nil || !got.FinishedAt.Equal(at.Add(3*time.Minute)) {
		t.Errorf("run = %+v", got)
	}
}

func TestJobQueue_Lifecycle(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	q := NewJobQueue(testPool)
	now := time.Now().UTC()
	job := usecase.NewSyncArtistJob(string(nirvana), "", "manual", now.Add(-time.Second))

	if ok, err := q.Enqueue(ctx, job); err != nil || !ok {
		t.Fatalf("enqueue: %t, %v", ok, err)
	}
	if ok, _ := q.Enqueue(ctx, job); ok {
		t.Error("duplicate dedup key must be ignored")
	}

	claimed, ok, err := q.Claim(ctx)
	if err != nil || !ok || claimed.Kind != usecase.JobSyncArtist || claimed.Attempts != 1 || claimed.MaxAttempts != 5 {
		t.Fatalf("claim = %+v, %t, %v", claimed, ok, err)
	}
	if string(claimed.Payload) != `{"artist_mbid": "`+string(nirvana)+`"}` {
		t.Errorf("payload = %s", claimed.Payload)
	}
	if _, ok, _ := q.Claim(ctx); ok {
		t.Error("a locked job must not be claimed twice")
	}

	// Retry in the future: not claimable yet; then make it due.
	if err := q.Retry(ctx, claimed.ID, now.Add(time.Hour), errors.New("503")); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := q.Claim(ctx); ok {
		t.Error("a job scheduled in the future must wait")
	}
	if err := q.Retry(ctx, claimed.ID, now.Add(-time.Second), errors.New("503")); err != nil {
		t.Fatal(err)
	}
	again, ok, _ := q.Claim(ctx)
	if !ok || again.Attempts != 2 {
		t.Fatalf("second claim = %+v, %t", again, ok)
	}

	// A dead job is revived by enqueueing the same key again.
	if _, err := testPool.Exec(ctx, "UPDATE jobs SET attempts = max_attempts WHERE id = $1", again.ID); err != nil {
		t.Fatal(err)
	}
	if err := q.Bury(ctx, again.ID, errors.New("gave up")); err != nil {
		t.Fatal(err)
	}
	if ok, err := q.Enqueue(ctx, job); err != nil || !ok {
		t.Errorf("revive dead job: %t, %v", ok, err)
	}
	revived, ok, _ := q.Claim(ctx)
	if !ok || revived.ID != again.ID || revived.Attempts != 1 {
		t.Errorf("revived = %+v, %t", revived, ok)
	}

	if err := q.Complete(ctx, revived.ID); err != nil {
		t.Fatal(err)
	}
	if ok, _ := q.Enqueue(ctx, job); !ok {
		t.Error("after completion the dedup key is free again")
	}
}

func TestJobQueue_SkipLockedAndStale(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	q := NewJobQueue(testPool)
	past := time.Now().UTC().Add(-time.Second)
	for _, a := range []domain.MBID{nirvana, radiohead, sigurRos} {
		if _, err := q.Enqueue(ctx, usecase.NewSyncArtistJob(string(a), "", "manual", past)); err != nil {
			t.Fatal(err)
		}
	}

	// Concurrent workers never get the same job.
	var mu sync.Mutex
	seen := map[int64]int{}
	var wg sync.WaitGroup
	for range 6 {
		wg.Go(func() {
			if j, ok, err := q.Claim(ctx); err == nil && ok {
				mu.Lock()
				seen[j.ID]++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if len(seen) != 3 {
		t.Errorf("claimed %d distinct jobs, want 3", len(seen))
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("job %d claimed %d times", id, n)
		}
	}

	n, err := q.ReleaseStale(ctx, time.Now().UTC().Add(time.Minute))
	if err != nil || n != 3 {
		t.Errorf("released %d, %v; want 3", n, err)
	}
}
