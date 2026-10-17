package domain

import (
	"strings"
	"time"
)

// MBID is a MusicBrainz identifier: a UUID in canonical lowercase form.
type MBID string

// ParseMBID validates s and returns it in canonical form.
func ParseMBID(s string) (MBID, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != 36 {
		return "", invalidMBID()
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return "", invalidMBID()
			}
		default:
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				return "", invalidMBID()
			}
		}
	}
	return MBID(s), nil
}

func invalidMBID() error {
	return &ValidationError{Fields: []FieldError{{Field: "mbid", Message: "must be a UUID"}}}
}

// Artist is a MusicBrainz artist known to the system.
type Artist struct {
	MBID           MBID
	Name           string
	SortName       string
	Disambiguation string
	Type           string // Person, Group, …; empty if unknown
	Country        string // ISO 3166-1 alpha-2; empty if unknown
	LifeSpan       LifeSpan
	// LastSyncedAt is nil until the first (baseline) synchronization.
	LastSyncedAt *time.Time
}

// LifeSpan holds MusicBrainz partial dates: "YYYY", "YYYY-MM" or "YYYY-MM-DD".
type LifeSpan struct {
	Begin string
	End   string
	Ended bool
}

// ArtistCandidate is a search result with the MusicBrainz relevance score (0–100).
type ArtistCandidate struct {
	Artist
	Score int
}
