package artist

import (
	"context"

	"music-tracker/backend/internal/domain"
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

func (f *fakeCatalog) ListReleaseGroups(context.Context, domain.MBID) ([]domain.ReleaseGroup, error) {
	return nil, f.err
}
