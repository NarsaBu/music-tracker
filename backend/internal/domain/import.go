package domain

import (
	"strings"
	"time"
)

// ImportItemStatus is the state of one line of an import file.
type ImportItemStatus string

const (
	ImportPending     ImportItemStatus = "pending"      // waiting for the worker
	ImportMatched     ImportItemStatus = "matched"      // subscribed (or already was)
	ImportNeedsReview ImportItemStatus = "needs_review" // ambiguous: the user picks a candidate
	ImportNotFound    ImportItemStatus = "not_found"
	ImportSkipped     ImportItemStatus = "skipped" // dismissed by the user
	ImportFailed      ImportItemStatus = "failed"  // gave up after retries
)

var ImportItemStatuses = []ImportItemStatus{
	ImportPending, ImportMatched, ImportNeedsReview, ImportNotFound, ImportSkipped, ImportFailed,
}

func ParseImportItemStatus(s string) (ImportItemStatus, error) {
	for _, st := range ImportItemStatuses {
		if string(st) == s {
			return st, nil
		}
	}
	return "", &ValidationError{Fields: []FieldError{{Field: "status", Message: "unknown import item status"}}}
}

// Resolvable reports whether the user may still choose an artist for the line (US-3).
func (s ImportItemStatus) Resolvable() bool {
	return s == ImportNeedsReview || s == ImportNotFound || s == ImportFailed
}

// MaxImportLines bounds a single import file (docs/06-api.md §5).
const MaxImportLines = 5000

// ImportLine is a parsed line of an import file.
type ImportLine struct {
	LineNo int
	// Value is the name (or MBID) as written in the file, for display.
	Value string
	// MBID is set when the line names the artist by MBID or MusicBrainz link: it is looked up, not searched.
	MBID MBID
	// IncludeFeatured is the CSV override; nil = the global setting.
	IncludeFeatured *bool
}

// ImportJob is one uploaded file.
type ImportJob struct {
	ID       string
	Filename string
	Total    int
	// Counts holds the number of lines per status; absent statuses are zero.
	Counts    map[ImportItemStatus]int
	CreatedAt time.Time
	// FinishedAt is set once no line is pending; lines may still need review.
	FinishedAt *time.Time
}

func (j ImportJob) Finished() bool { return j.FinishedAt != nil }

// ImportItem is one line of an import and the outcome of matching it.
type ImportItem struct {
	ID              string
	JobID           string
	LineNo          int
	RawValue        string
	InputMBID       MBID
	IncludeFeatured *bool
	Status          ImportItemStatus
	// Candidates are the search results offered for review, best first.
	Candidates []ArtistCandidate
	// Artist is the subscribed artist of a matched line.
	Artist *Artist
	Error  string
}

// Auto-match thresholds (FR-2.3): a name is subscribed without asking only when
// the best search result is confident and clearly ahead of the runner-up.
const (
	AutoMatchMinScore = 95
	AutoMatchMinLead  = 10
)

// AutoMatch picks the candidate a name can be subscribed to without review.
func AutoMatch(cands []ArtistCandidate) (ArtistCandidate, bool) {
	if len(cands) == 0 {
		return ArtistCandidate{}, false
	}
	best, second := 0, -1
	for i := 1; i < len(cands); i++ {
		switch {
		case cands[i].Score > cands[best].Score:
			best, second = i, best
		case second < 0 || cands[i].Score > cands[second].Score:
			second = i
		}
	}
	if cands[best].Score < AutoMatchMinScore {
		return ArtistCandidate{}, false
	}
	if second >= 0 && cands[best].Score-cands[second].Score < AutoMatchMinLead {
		return ArtistCandidate{}, false
	}
	return cands[best], true
}

// ParseArtistRef accepts a bare MBID or a MusicBrainz artist link
// (https://musicbrainz.org/artist/<mbid>, with any trailing path or query).
func ParseArtistRef(s string) (MBID, bool) {
	s = strings.TrimSpace(s)
	if mbid, err := ParseMBID(s); err == nil {
		return mbid, true
	}
	_, rest, ok := strings.Cut(strings.ToLower(s), "musicbrainz.org/artist/")
	if !ok || len(rest) < 36 {
		return "", false
	}
	if len(rest) > 36 && !strings.ContainsRune("/?#", rune(rest[36])) {
		return "", false
	}
	mbid, err := ParseMBID(rest[:36])
	return mbid, err == nil
}
