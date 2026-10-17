package usecase

import (
	"fmt"
	"time"
)

// Job is a unit of background work. Payload is marshaled to JSON by the queue.
type Job struct {
	Kind string
	// DedupKey makes enqueueing idempotent; empty means no deduplication.
	DedupKey string
	Payload  any
	RunAt    time.Time
}

// JobSyncArtist loads an artist's release groups from the catalog.
const JobSyncArtist = "sync_artist"

// SyncArtistPayload is the payload of JobSyncArtist.
type SyncArtistPayload struct {
	ArtistMBID string `json:"artist_mbid"`
	// RunID is set when the job belongs to a sync run; empty for single-artist syncs.
	RunID string `json:"run_id,omitempty"`
}

// NewSyncArtistJob builds a sync job. The dedup scope keeps one pending job per
// artist and scope: a run ID, "baseline" or "manual".
func NewSyncArtistJob(artist, runID, scope string, at time.Time) Job {
	return Job{
		Kind:     JobSyncArtist,
		DedupKey: JobSyncArtist + ":" + scope + ":" + artist,
		Payload:  SyncArtistPayload{ArtistMBID: artist, RunID: runID},
		RunAt:    at,
	}
}

// JobSendDigest sends the email digest that was due at Due.
const JobSendDigest = "send_digest"

type SendDigestPayload struct {
	Due time.Time `json:"due"`
}

// NewSendDigestJob deduplicates by the due time: one digest per slot.
func NewSendDigestJob(due, at time.Time) Job {
	return Job{
		Kind:     JobSendDigest,
		DedupKey: fmt.Sprintf("%s:%d", JobSendDigest, due.Unix()),
		Payload:  SendDigestPayload{Due: due},
		RunAt:    at,
	}
}

// JobImportItem matches one line of an import file to an artist.
const JobImportItem = "import_item"

type ImportItemPayload struct {
	ItemID string `json:"item_id"`
}

func NewImportItemJob(itemID string, at time.Time) Job {
	return Job{
		Kind:     JobImportItem,
		DedupKey: JobImportItem + ":" + itemID,
		Payload:  ImportItemPayload{ItemID: itemID},
		RunAt:    at,
	}
}

// PermanentError marks a job failure that retrying cannot fix; the job is buried at once.
type PermanentError struct{ Err error }

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }
