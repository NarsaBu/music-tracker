package release

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

const (
	// maxCoverEditions bounds the editions searched after the release group's own
	// cover: each one costs an archive request.
	maxCoverEditions = 5
	// probeTimeout: an archive.org mirror that does not answer in time is skipped,
	// not waited for.
	probeTimeout = 15 * time.Second
)

// CoverView is the result of a cover search.
type CoverView struct {
	// Found is false when no front cover of domain.CoverMinSize exists;
	// Largest then describes the biggest one seen, if any.
	Found   bool
	Cover   domain.CoverImage
	Largest *domain.CoverImage
	// Filename is "Artist - Title.ext", safe for a download.
	Filename string
}

// FindCover looks for a front cover of at least domain.CoverMinSize on both sides:
// the release group's own cover first, then the covers of its editions, best first.
type FindCover struct {
	releases usecase.ReleaseGroupRepository
	catalog  usecase.ReleaseCatalog
	details  usecase.ReleaseDetailsCache
	archive  usecase.CoverArchive
	clock    usecase.Clock
}

func NewFindCover(
	releases usecase.ReleaseGroupRepository,
	catalog usecase.ReleaseCatalog,
	details usecase.ReleaseDetailsCache,
	archive usecase.CoverArchive,
	clock usecase.Clock,
) *FindCover {
	return &FindCover{releases: releases, catalog: catalog, details: details, archive: archive, clock: clock}
}

func (uc *FindCover) Execute(ctx context.Context, rawMBID string) (CoverView, error) {
	mbid, err := domain.ParseMBID(rawMBID)
	if err != nil {
		return CoverView{}, err
	}
	rg, err := uc.releases.Get(ctx, mbid)
	if err != nil {
		return CoverView{}, fmt.Errorf("cover of %s: %w", mbid, err)
	}

	view := CoverView{}
	seen := map[string]bool{}
	var upstream error
	try := func(url string, err error) bool {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			return false
		case err != nil:
			upstream = err
			return false
		case seen[url]:
			return false
		}
		seen[url] = true
		probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		defer cancel()
		img, err := uc.archive.Probe(probeCtx, url)
		if err != nil {
			if !errors.Is(err, domain.ErrNotFound) {
				upstream = err
			}
			return false
		}
		if img.AtLeast(domain.CoverMinSize) {
			view.Found, view.Cover = true, img
			return true
		}
		if view.Largest == nil || img.Width*img.Height > view.Largest.Width*view.Largest.Height {
			view.Largest = &img
		}
		return false
	}

	found := false
	for _, f := range uc.fronts(ctx, mbid) {
		if found = try(f.url, f.err); found {
			break
		}
	}
	if !found && view.Largest == nil && upstream != nil {
		// Nothing seen at all because the archive failed: not the same as "no cover".
		return CoverView{}, fmt.Errorf("cover of %s: %w", mbid, upstream)
	}
	if found {
		view.Filename = coverFilename(rg, view.Cover)
	}
	return view, nil
}

type front struct {
	url string
	err error
}

// fronts looks up the front image of the release group and of its editions at
// once, and returns them in order of preference.
func (uc *FindCover) fronts(ctx context.Context, mbid domain.MBID) []front {
	editions := uc.editions(ctx, mbid)
	out := make([]front, 1+len(editions))
	var wg sync.WaitGroup
	wg.Go(func() { out[0].url, out[0].err = uc.archive.ReleaseGroupFront(ctx, mbid) })
	for i, e := range editions {
		wg.Go(func() { out[i+1].url, out[i+1].err = uc.archive.ReleaseFront(ctx, e) })
	}
	wg.Wait()
	return out
}

// editions lists editions to search, best first; without MusicBrainz there are none.
func (uc *FindCover) editions(ctx context.Context, mbid domain.MBID) []domain.MBID {
	d, err := loadCached(ctx, mbid, uc.details, uc.catalog.ReleaseGroupDetails, uc.clock.Now(), DetailsTTL)
	if err != nil {
		return nil
	}
	editions := d.Editions
	if len(editions) == 0 && d.Edition.MBID != "" {
		editions = []domain.Edition{d.Edition} // cached before editions were stored
	}
	var out []domain.MBID
	for _, e := range editions[:min(len(editions), maxCoverEditions)] {
		out = append(out, e.MBID)
	}
	return out
}

// OpenCover starts the download of the cover FindCover picks.
type OpenCover struct {
	find    *FindCover
	archive usecase.CoverArchive
}

func NewOpenCover(find *FindCover, archive usecase.CoverArchive) *OpenCover {
	return &OpenCover{find: find, archive: archive}
}

// Execute returns domain.ErrNotFound when there is no cover big enough.
func (uc *OpenCover) Execute(ctx context.Context, rawMBID string) (io.ReadCloser, CoverView, error) {
	view, err := uc.find.Execute(ctx, rawMBID)
	if err != nil {
		return nil, CoverView{}, err
	}
	if !view.Found {
		return nil, view, fmt.Errorf("no cover of at least %[1]dx%[1]d: %w", domain.CoverMinSize, domain.ErrNotFound)
	}
	body, err := uc.archive.Open(ctx, view.Cover.URL)
	if err != nil {
		return nil, CoverView{}, fmt.Errorf("download cover: %w", err)
	}
	return body, view, nil
}

// coverFilename builds "Artist - Title.ext" without characters file systems reject.
func coverFilename(rg domain.ReleaseGroup, img domain.CoverImage) string {
	name := rg.Title
	if rg.ArtistCredit != "" {
		name = rg.ArtistCredit + " - " + rg.Title
	}
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, name)
	return strings.TrimSpace(name) + img.Extension()
}
