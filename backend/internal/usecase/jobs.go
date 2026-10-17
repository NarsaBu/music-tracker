package usecase

import "time"

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

// PermanentError marks a job failure that retrying cannot fix; the job is buried at once.
type PermanentError struct{ Err error }

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }
