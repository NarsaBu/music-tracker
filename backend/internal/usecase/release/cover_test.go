package release

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/memory"
)

const (
	edition1 domain.MBID = "00000000-0000-0000-0000-0000000000e1"
	edition2 domain.MBID = "00000000-0000-0000-0000-0000000000e2"
)

type fakeArchive struct {
	fronts  map[domain.MBID]string // release group or release → image URL
	images  map[string]domain.CoverImage
	err     error // returned by every front lookup when set
	probed  []string
	opened  string
	content string
}

func (f *fakeArchive) front(id domain.MBID) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if u, ok := f.fronts[id]; ok {
		return u, nil
	}
	return "", domain.ErrNotFound
}

func (f *fakeArchive) ReleaseGroupFront(_ context.Context, rg domain.MBID) (string, error) {
	return f.front(rg)
}

func (f *fakeArchive) ReleaseFront(_ context.Context, r domain.MBID) (string, error) {
	return f.front(r)
}

func (f *fakeArchive) Probe(_ context.Context, url string) (domain.CoverImage, error) {
	f.probed = append(f.probed, url)
	img, ok := f.images[url]
	if !ok {
		return domain.CoverImage{}, domain.ErrNotFound
	}
	img.URL = url
	return img, nil
}

func (f *fakeArchive) Open(_ context.Context, url string) (io.ReadCloser, error) {
	f.opened = url
	return io.NopCloser(strings.NewReader(f.content)), nil
}

func coverStore() *memory.Store {
	s := storeWithRelease()
	rg := s.Releases[okComputer]
	rg.ArtistCredit, rg.Title = "AC/DC", "Back: In Black?"
	s.Releases[okComputer] = rg
	return s
}

func newFindCover(s *memory.Store, archive *fakeArchive) *FindCover {
	catalog := &fakeCatalog{details: domain.ReleaseDetails{Editions: []domain.Edition{{MBID: edition1}, {MBID: edition2}}}}
	return NewFindCover(s.ReleaseRepo(), catalog, fakeDetailsCache{}, archive, &memory.Clock{T: now})
}

func TestFindCover_FallsBackToEditionsForABigEnoughCover(t *testing.T) {
	archive := &fakeArchive{
		fronts: map[domain.MBID]string{okComputer: "rg.jpg", edition1: "rg.jpg", edition2: "big.png"},
		images: map[string]domain.CoverImage{
			"rg.jpg":  {Width: 500, Height: 500, ContentType: "image/jpeg"},
			"big.png": {Width: 3000, Height: 3000, ContentType: "image/png", Size: 9000},
		},
	}
	v, err := newFindCover(coverStore(), archive).Execute(context.Background(), string(okComputer))
	if err != nil {
		t.Fatal(err)
	}
	if !v.Found || v.Cover.URL != "big.png" || v.Filename != "AC_DC - Back_ In Black_.png" {
		t.Errorf("view = %+v", v)
	}
	// The release group's image is probed once even though edition 1 shares it.
	if strings.Join(archive.probed, ",") != "rg.jpg,big.png" {
		t.Errorf("probed = %v", archive.probed)
	}
}

func TestFindCover_TooSmallEverywhere(t *testing.T) {
	archive := &fakeArchive{
		fronts: map[domain.MBID]string{okComputer: "small.jpg", edition2: "medium.jpg"},
		images: map[string]domain.CoverImage{
			"small.jpg":  {Width: 250, Height: 250},
			"medium.jpg": {Width: 1200, Height: 1000}, // wide enough, not tall enough
		},
	}
	s := coverStore()
	v, err := newFindCover(s, archive).Execute(context.Background(), string(okComputer))
	if err != nil || v.Found || v.Largest == nil || v.Largest.Width != 1200 {
		t.Errorf("view = %+v, err = %v; want not found with the 1200×1000 one as largest", v, err)
	}

	_, _, err = NewOpenCover(newFindCover(s, archive), archive).Execute(context.Background(), string(okComputer))
	if !errors.Is(err, domain.ErrNotFound) || archive.opened != "" {
		t.Errorf("open: %v, opened %q; want ErrNotFound without a download", err, archive.opened)
	}
}

func TestFindCover_ArchiveDownIsNotNoCover(t *testing.T) {
	archive := &fakeArchive{err: fmt.Errorf("%w: caa", domain.ErrUpstreamUnavailable)}
	_, err := newFindCover(coverStore(), archive).Execute(context.Background(), string(okComputer))
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("err = %v, want ErrUpstreamUnavailable", err)
	}
}

func TestOpenCover_StreamsTheFoundCover(t *testing.T) {
	archive := &fakeArchive{
		fronts:  map[domain.MBID]string{okComputer: "big.jpg"},
		images:  map[string]domain.CoverImage{"big.jpg": {Width: 1200, Height: 1200, ContentType: "image/jpeg"}},
		content: "jpeg bytes",
	}
	body, v, err := NewOpenCover(newFindCover(coverStore(), archive), archive).Execute(context.Background(), string(okComputer))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(body)
	_ = body.Close()
	if string(got) != "jpeg bytes" || archive.opened != "big.jpg" || v.Filename != "AC_DC - Back_ In Black_.jpg" {
		t.Errorf("got %q from %q as %q", got, archive.opened, v.Filename)
	}
}
