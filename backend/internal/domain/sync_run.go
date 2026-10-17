package domain

import "time"

type SyncTrigger string

const (
	TriggerSchedule SyncTrigger = "schedule"
	TriggerManual   SyncTrigger = "manual"
)

// SyncRun is one pass over all tracked artists (scheduled or started by hand).
type SyncRun struct {
	ID           string
	Trigger      SyncTrigger
	StartedAt    time.Time
	FinishedAt   *time.Time
	ArtistsTotal int
	ArtistsDone  int
	// NewReleases counts release groups that would be notified: not baseline, not stale.
	NewReleases int
}

func (r SyncRun) Finished() bool { return r.FinishedAt != nil }
