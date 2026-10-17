// Package sync holds the synchronization use cases (docs/07-musicbrainz.md §4):
// ScheduleSyncRun, Artist (SyncArtist), RequestArtist and LatestSyncRun.
package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

type Config struct {
	// StaleThreshold: a release older than this, found only now, is not news (FR-3.4).
	StaleThreshold time.Duration
}

type ArtistResult struct {
	Fetched    int // release groups in the catalog
	Discovered int // linked to the artist for the first time
	New        int // discovered, not baseline, not stale: would be notified
	Baseline   bool
}

// Artist fetches an artist's release groups, stores them and links the new
// ones with the artist's credit role. The first sync of an artist is the baseline:
// everything it finds is history, not news (ADR-3).
type Artist struct {
	catalog  usecase.MusicCatalog
	artists  usecase.ArtistRepository
	releases usecase.ReleaseGroupRepository
	runs     usecase.SyncRunRepository
	tx       usecase.TxManager
	clock    usecase.Clock
	cfg      Config
}

func NewArtist(
	catalog usecase.MusicCatalog,
	artists usecase.ArtistRepository,
	releases usecase.ReleaseGroupRepository,
	runs usecase.SyncRunRepository,
	tx usecase.TxManager,
	clock usecase.Clock,
	cfg Config,
) *Artist {
	return &Artist{catalog: catalog, artists: artists, releases: releases, runs: runs, tx: tx, clock: clock, cfg: cfg}
}

// Execute syncs one artist; runID, when set, is advanced in the same transaction.
func (uc *Artist) Execute(ctx context.Context, mbid domain.MBID, runID string) (ArtistResult, error) {
	artist, err := uc.artists.Get(ctx, mbid)
	if err != nil {
		return ArtistResult{}, fmt.Errorf("sync artist %s: %w", mbid, err)
	}
	res := ArtistResult{Baseline: artist.LastSyncedAt == nil}

	// A partially loaded catalog must not be written: fetch everything first.
	remote, err := uc.catalog.ListReleaseGroups(ctx, mbid)
	if err != nil {
		return ArtistResult{}, fmt.Errorf("sync artist %s: list release groups: %w", mbid, err)
	}
	known, err := uc.releases.KnownMBIDs(ctx, mbid)
	if err != nil {
		return ArtistResult{}, fmt.Errorf("sync artist %s: %w", mbid, err)
	}

	now := uc.clock.Now()
	var links []domain.ArtistReleaseGroup
	for _, rg := range remote {
		if known[rg.MBID] {
			continue
		}
		known[rg.MBID] = true
		links = append(links, domain.ArtistReleaseGroup{
			ArtistMBID:   mbid,
			ReleaseGroup: rg,
			Role:         domain.RoleOf(rg.Credits, mbid),
			IsBaseline:   res.Baseline,
			DiscoveredAt: now,
		})
		if !res.Baseline && !rg.FirstRelease.IsStale(now, uc.cfg.StaleThreshold) {
			res.New++
		}
	}
	res.Fetched, res.Discovered = len(remote), len(links)

	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.releases.UpsertMany(ctx, remote, now); err != nil {
			return err
		}
		if err := uc.releases.LinkMany(ctx, links); err != nil {
			return err
		}
		if err := uc.artists.MarkSynced(ctx, mbid, now); err != nil {
			return err
		}
		if runID != "" {
			return uc.runs.ArtistDone(ctx, runID, res.New, now)
		}
		return nil
	})
	if err != nil {
		return ArtistResult{}, fmt.Errorf("sync artist %s: store: %w", mbid, err)
	}
	return res, nil
}

// HandleJob runs a usecase.JobSyncArtist job. When the job gives up for good the
// artist still counts as done, so that the run can finish.
func (uc *Artist) HandleJob(ctx context.Context, payload []byte, lastAttempt bool) error {
	var p usecase.SyncArtistPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return &usecase.PermanentError{Err: fmt.Errorf("decode payload: %w", err)}
	}
	mbid, err := domain.ParseMBID(p.ArtistMBID)
	if err != nil {
		return &usecase.PermanentError{Err: err}
	}

	_, err = uc.Execute(ctx, mbid, p.RunID)
	if errors.Is(err, domain.ErrNotFound) {
		// Unknown artist here or in MusicBrainz (deleted, merged): retrying will not help.
		err = &usecase.PermanentError{Err: err}
	}
	var permanent *usecase.PermanentError
	if err != nil && p.RunID != "" && (lastAttempt || errors.As(err, &permanent)) {
		if doneErr := uc.runs.ArtistDone(ctx, p.RunID, 0, uc.clock.Now()); doneErr != nil {
			return errors.Join(err, doneErr)
		}
	}
	return err
}
