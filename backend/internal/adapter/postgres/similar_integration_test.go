//go:build integration

package postgres

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
)

func TestSimilarArtistsCache_RoundTrip(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	cache := NewSimilarArtistsCache(testPool)

	if _, _, err := cache.Get(ctx, nirvana); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty cache: %v, want ErrNotFound", err)
	}

	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	items := []domain.SimilarArtist{
		{MBID: radiohead, Name: "Radiohead", Disambiguation: "UK rock band", Type: "Group", Score: 900},
		{MBID: rgOld, Name: "Unknown type", Score: 10},
	}
	if err := cache.Put(ctx, nirvana, items, at); err != nil {
		t.Fatal(err)
	}
	got, fetchedAt, err := cache.Get(ctx, nirvana)
	if err != nil || !slices.Equal(got, items) || !fetchedAt.Equal(at) {
		t.Errorf("got %+v at %v, err = %v; want %+v at %v", got, fetchedAt, err, items, at)
	}

	// A refresh replaces the answer, an empty one included.
	later := at.Add(time.Hour)
	if err := cache.Put(ctx, nirvana, nil, later); err != nil {
		t.Fatal(err)
	}
	got, fetchedAt, err = cache.Get(ctx, nirvana)
	if err != nil || len(got) != 0 || !fetchedAt.Equal(later) {
		t.Errorf("after refresh: got %+v at %v, err = %v", got, fetchedAt, err)
	}
}
