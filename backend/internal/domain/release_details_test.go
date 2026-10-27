package domain

import (
	"slices"
	"testing"
)

func TestPickEdition(t *testing.T) {
	editions := []Edition{
		{MBID: "deluxe", Status: "Official", Date: ParseReleaseDate("2017-06-23")},
		{MBID: "undated", Status: "Official"},
		{MBID: "promo", Status: "Promotion", Date: ParseReleaseDate("1997-04-01")},
		{MBID: "original", Status: "Official", Date: ParseReleaseDate("1997-05-21")},
		{MBID: "same-day", Status: "Official", Date: ParseReleaseDate("1997-05-21")},
	}
	got, ok := PickEdition(editions)
	if !ok || got.MBID != "original" {
		t.Errorf("picked %q, want the earliest official edition (first on a tie)", got.MBID)
	}

	if got, _ := PickEdition([]Edition{{MBID: "undated", Status: "Official"}, {MBID: "boot", Status: "Bootleg", Date: ParseReleaseDate("1990")}}); got.MBID != "undated" {
		t.Errorf("picked %q, want an official edition over a dated bootleg", got.MBID)
	}
	// A year alone is not January 1st: the dated edition of that year wins.
	got, _ = PickEdition([]Edition{
		{MBID: "year-only", Status: "Official", Date: ParseReleaseDate("1997")},
		{MBID: "month-only", Status: "Official", Date: ParseReleaseDate("1997-06")},
		{MBID: "dated", Status: "Official", Date: ParseReleaseDate("1997-05-21")},
	})
	if got.MBID != "dated" {
		t.Errorf("picked %q, want the precisely dated edition of the same year", got.MBID)
	}
	got, _ = PickEdition([]Edition{
		{MBID: "later-dated", Status: "Official", Date: ParseReleaseDate("1998-01-10")},
		{MBID: "earlier-year", Status: "Official", Date: ParseReleaseDate("1997")},
	})
	if got.MBID != "earlier-year" {
		t.Errorf("picked %q, want an earlier year over a later precise date", got.MBID)
	}

	if _, ok := PickEdition(nil); ok {
		t.Error("no editions must not pick one")
	}
}

func TestGuestsOf(t *testing.T) {
	const rocky, guest, other MBID = "rocky", "guest", "other"
	release := ArtistCredit{{ArtistMBID: rocky, Name: "A$AP Rocky"}}
	media := []Medium{
		{Tracks: []Track{
			{Credits: ArtistCredit{{ArtistMBID: rocky, Name: "A$AP Rocky", JoinPhrase: " feat. "}, {ArtistMBID: guest, Name: "Guest"}}},
			{Credits: ArtistCredit{{ArtistMBID: rocky, Name: "A$AP Rocky"}}},
		}},
		{Tracks: []Track{
			// The same guest again, credited under another name, and an artist without an id.
			{Credits: ArtistCredit{{ArtistMBID: other, Name: "Other", JoinPhrase: " & "}, {ArtistMBID: guest, Name: "G."}, {Name: "Nobody"}}},
		}},
	}

	got := GuestsOf(release, media)
	want := []CreditName{{ArtistMBID: guest, Name: "Guest"}, {ArtistMBID: other, Name: "Other"}}
	if !slices.Equal(got, want) {
		t.Errorf("guests = %+v, want %+v", got, want)
	}
	if got := GuestsOf(release, nil); len(got) != 0 {
		t.Errorf("no tracks: %+v", got)
	}
}
