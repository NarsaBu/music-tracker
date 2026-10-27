package domain

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// ReleaseDetails is what the release page shows beyond the stored release group:
// genres, the tracklist of one edition and the artists credited on it.
type ReleaseDetails struct {
	// Genres, most voted first.
	Genres []string
	// Edition is the release the tracklist comes from (PickEdition).
	Edition Edition
	Media   []Medium
	// Artists are credited on the release group itself.
	Artists []CreditName
	// Guests are credited on tracks but not on the release group: featurings.
	Guests []CreditName
}

// Edition is one release (pressing, country, reissue) of a release group.
type Edition struct {
	MBID    MBID
	Title   string
	Date    ReleaseDate
	Country string
	Status  string // Official, Promotion, Bootleg, …
}

type Medium struct {
	Position int
	Format   string // CD, Digital Media, 12" Vinyl, …; empty when unknown
	Title    string
	Tracks   []Track
}

type Track struct {
	Number string // as printed: "1", "A1", …
	Title  string
	Length time.Duration // zero when unknown
	// ArtistCredit is the display string; Credits the structured one.
	ArtistCredit string
	Credits      ArtistCredit
}

// PickEdition chooses the edition to show the tracklist of: the earliest official
// one, so a deluxe reissue with bonus tracks does not stand for the album.
// Undated editions go last; ok is false when there are none.
func PickEdition(editions []Edition) (Edition, bool) {
	if len(editions) == 0 {
		return Edition{}, false
	}
	sorted := slices.Clone(editions)
	slices.SortStableFunc(sorted, func(a, b Edition) int {
		if ao, bo := a.Status == "Official", b.Status == "Official"; ao != bo {
			if ao {
				return -1
			}
			return 1
		}
		return compareEditionDates(a.Date, b.Date)
	})
	return sorted[0], true
}

// compareEditionDates orders partial dates part by part. A missing part is not
// "January 1st": an edition known only by its year loses to a dated one of that
// year, which is usually the actual first release. Undated editions go last.
func compareEditionDates(a, b ReleaseDate) int {
	switch {
	case a.Day == nil && b.Day == nil:
		return 0
	case a.Day == nil:
		return 1
	case b.Day == nil:
		return -1
	}
	ap, bp := strings.Split(a.Raw, "-"), strings.Split(b.Raw, "-")
	for i := range 3 {
		switch {
		case i >= len(ap) && i >= len(bp):
			return 0
		case i >= len(ap):
			return 1
		case i >= len(bp):
			return -1
		}
		if c := cmp.Compare(ap[i], bp[i]); c != 0 {
			return c
		}
	}
	return 0
}

// UniqueArtists lists the artists of a credit once each, in credit order;
// names without an artist id are dropped.
func UniqueArtists(credit ArtistCredit) []CreditName {
	var out []CreditName
	for _, c := range credit {
		if c.ArtistMBID == "" || slices.ContainsFunc(out, func(x CreditName) bool { return x.ArtistMBID == c.ArtistMBID }) {
			continue
		}
		out = append(out, CreditName{ArtistMBID: c.ArtistMBID, Name: c.Name})
	}
	return out
}

// GuestsOf lists artists credited on tracks but not on the release itself,
// in order of first appearance.
func GuestsOf(release ArtistCredit, media []Medium) []CreditName {
	var all ArtistCredit
	for _, m := range media {
		for _, t := range m.Tracks {
			all = append(all, t.Credits...)
		}
	}
	main := UniqueArtists(release)
	var out []CreditName
	for _, a := range UniqueArtists(all) {
		if !slices.ContainsFunc(main, func(x CreditName) bool { return x.ArtistMBID == a.ArtistMBID }) {
			out = append(out, a)
		}
	}
	return out
}
