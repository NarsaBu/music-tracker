//go:build integration

package postgres

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
)

func TestReleaseGroupRepository_RatingAndGet(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	seedReleases(t, at)
	repo := NewReleaseGroupRepository(testPool)

	rated := domain.ReleaseGroup{
		MBID: rgOld, Title: "Album", PrimaryType: domain.PrimaryAlbum, ArtistCredit: "Nirvana",
		SecondaryTypes: []string{}, Rating: domain.Rating{Value: 4.55, Votes: 89},
	}
	if err := repo.UpsertMany(ctx, []domain.ReleaseGroup{rated}, at); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, rgOld)
	if err != nil || got.Rating != rated.Rating || got.Title != "Album" {
		t.Errorf("rated = %+v, err = %v", got, err)
	}
	if got, err := repo.Get(ctx, rgNew); err != nil || got.Rating != (domain.Rating{}) {
		t.Errorf("unrated = %+v, err = %v", got.Rating, err)
	}
	if _, err := repo.Get(ctx, "00000000-0000-0000-0000-0000000000ff"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown: %v, want ErrNotFound", err)
	}
}

func TestReviewCache_RoundTrip(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	seedReleases(t, at)
	cache := NewReviewCache(testPool)

	if _, _, err := cache.Get(ctx, rgOld); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty cache: %v, want ErrNotFound", err)
	}
	want := domain.Reviews{
		Items: []domain.Review{{
			ID: "c508a8ec-1bd0-4f92-bcd8-581a2cd2e75f", Author: "Jon Lusk", Rating: 4, Text: "Fine.",
			Language: "en", PublishedAt: time.Date(2007, 4, 25, 0, 0, 0, 0, time.UTC), License: "CC BY-SA 3.0",
		}},
		Total: 2, Average: 4.5, Rated: 2,
	}
	if err := cache.Put(ctx, rgOld, want, at); err != nil {
		t.Fatal(err)
	}
	got, fetchedAt, err := cache.Get(ctx, rgOld)
	if err != nil || !reflect.DeepEqual(got, want) || !fetchedAt.Equal(at) {
		t.Errorf("got %+v at %v, err = %v; want %+v", got, fetchedAt, err, want)
	}
}

func TestReleaseDetailsCache_RoundTrip(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	seedReleases(t, at)
	cache := NewReleaseDetailsCache(testPool)

	if _, _, err := cache.Get(ctx, rgOld); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty cache: %v, want ErrNotFound", err)
	}
	credit := domain.ArtistCredit{{ArtistMBID: nirvana, Name: "Nirvana", JoinPhrase: " feat. "}, {ArtistMBID: radiohead, Name: "Radiohead"}}
	want := domain.ReleaseDetails{
		Genres: []string{"grunge"},
		Edition: domain.Edition{
			MBID: rgNew, Title: "Album", Date: domain.ParseReleaseDate("1991-09-24"), Country: "US", Status: "Official",
		},
		Media: []domain.Medium{{Position: 1, Format: "CD", Tracks: []domain.Track{
			{Number: "1", Title: "Song", Length: 301 * time.Second, ArtistCredit: credit.String(), Credits: credit},
		}}},
		Artists: []domain.CreditName{{ArtistMBID: nirvana, Name: "Nirvana"}},
		Guests:  []domain.CreditName{{ArtistMBID: radiohead, Name: "Radiohead"}},
	}
	if err := cache.Put(ctx, rgOld, want, at); err != nil {
		t.Fatal(err)
	}
	got, fetchedAt, err := cache.Get(ctx, rgOld)
	if err != nil || !reflect.DeepEqual(got, want) || !fetchedAt.Equal(at) {
		t.Errorf("got %+v at %v, err = %v; want %+v", got, fetchedAt, err, want)
	}
}
