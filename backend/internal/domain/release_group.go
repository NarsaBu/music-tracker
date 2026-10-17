package domain

import (
	"strings"
	"time"
)

// PrimaryType is a MusicBrainz release group primary type, stored lowercase.
type PrimaryType string

const (
	PrimaryAlbum     PrimaryType = "album"
	PrimarySingle    PrimaryType = "single"
	PrimaryEP        PrimaryType = "ep"
	PrimaryBroadcast PrimaryType = "broadcast"
	PrimaryOther     PrimaryType = "other"
)

// PrimaryTypes lists every known primary type.
var PrimaryTypes = []PrimaryType{PrimaryAlbum, PrimarySingle, PrimaryEP, PrimaryBroadcast, PrimaryOther}

// SecondaryTypes lists MusicBrainz secondary types, spelled exactly as in the API.
var SecondaryTypes = []string{
	"Compilation", "Soundtrack", "Spokenword", "Interview", "Audiobook", "Audio drama",
	"Live", "Remix", "DJ-mix", "Mixtape/Street", "Demo", "Field recording",
}

// ReleaseGroup is what a listener calls "an album": all editions of one release (ADR-1).
type ReleaseGroup struct {
	MBID           MBID
	Title          string
	PrimaryType    PrimaryType // empty when MusicBrainz has none
	SecondaryTypes []string
	FirstRelease   ReleaseDate
	// ArtistCredit is the display string, e.g. "A feat. B"; it is what gets stored.
	ArtistCredit string
	// Credits is the structured credit used to compute roles; set only by the catalog.
	Credits ArtistCredit
}

// ReleaseDate is a MusicBrainz partial date: "YYYY", "YYYY-MM", "YYYY-MM-DD" or empty.
type ReleaseDate struct {
	Raw string
	// Day is Raw normalized to the first day of its period; nil when Raw is empty or malformed.
	Day *time.Time
}

// ParseReleaseDate never fails: a malformed date is kept as text without Day.
func ParseReleaseDate(raw string) ReleaseDate {
	raw = strings.TrimSpace(raw)
	d := ReleaseDate{Raw: raw}
	for _, layout := range []string{"2006-01-02", "2006-01", "2006"} {
		if len(raw) != len(layout) {
			continue
		}
		if t, err := time.Parse(layout, raw); err == nil {
			d.Day = &t
		}
		break
	}
	return d
}

// IsStale reports a release older than threshold: found late, it is not "news" (FR-3.4).
// An undated release is never stale — often it is a freshly added announcement.
func (d ReleaseDate) IsStale(now time.Time, threshold time.Duration) bool {
	return d.Day != nil && d.Day.Before(now.Add(-threshold))
}

// IsAnnouncement reports a release whose date is still in the future.
func (d ReleaseDate) IsAnnouncement(now time.Time) bool {
	return d.Day != nil && d.Day.After(now)
}

// ArtistReleaseGroup links an artist to a release group with the artist's role in its credit.
type ArtistReleaseGroup struct {
	ArtistMBID   MBID
	ReleaseGroup ReleaseGroup
	Role         CreditRole
	// IsBaseline: found by the first synchronization of the artist, never notified (ADR-3).
	IsBaseline   bool
	DiscoveredAt time.Time
}
