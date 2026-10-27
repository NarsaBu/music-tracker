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

type ArtistResult struct {
	Fetched    int // release groups in the catalog
	Discovered int // linked to the artist for the first time
	New        int // discovered, news (not stale), passed the filters: notified
	Baseline   bool
}

// ReleaseNotifier creates notifications for fresh discoveries (notify.FanOut).
// It runs inside the sync transaction.
type ReleaseNotifier interface {
	Execute(ctx context.Context, artist domain.MBID, fresh []domain.ArtistReleaseGroup) (int, error)
}

// Artist fetches an artist's release groups, stores them and links the new
// ones with the artist's credit role. The first sync of an artist is the baseline:
// what it finds is history, except releases inside the stale window (ADR-3).
type Artist struct {
	catalog  usecase.MusicCatalog
	artists  usecase.ArtistRepository
	subs     usecase.SubscriptionRepository
	releases usecase.ReleaseGroupRepository
	runs     usecase.SyncRunRepository
	settings usecase.SettingsRepository
	notifier ReleaseNotifier
	tx       usecase.TxManager
	clock    usecase.Clock
}

func NewArtist(
	catalog usecase.MusicCatalog,
	artists usecase.ArtistRepository,
	subs usecase.SubscriptionRepository,
	releases usecase.ReleaseGroupRepository,
	runs usecase.SyncRunRepository,
	settings usecase.SettingsRepository,
	notifier ReleaseNotifier,
	tx usecase.TxManager,
	clock usecase.Clock,
) *Artist {
	return &Artist{
		catalog: catalog, artists: artists, subs: subs, releases: releases, runs: runs,
		settings: settings, notifier: notifier, tx: tx, clock: clock,
	}
}

// Execute syncs one artist; runID, when set, is advanced in the same transaction.
func (uc *Artist) Execute(ctx context.Context, mbid domain.MBID, runID string) (ArtistResult, error) {
	artist, err := uc.artists.Get(ctx, mbid)
	if err != nil {
		return ArtistResult{}, fmt.Errorf("sync artist %s: %w", mbid, err)
	}
	baseline, err := uc.isBaseline(ctx, artist)
	if err != nil {
		return ArtistResult{}, fmt.Errorf("sync artist %s: %w", mbid, err)
	}
	res := ArtistResult{Baseline: baseline}

	// A partially loaded catalog must not be written: fetch everything first.
	remote, err := uc.catalog.ListReleaseGroups(ctx, mbid)
	if err != nil {
		return ArtistResult{}, fmt.Errorf("sync artist %s: list release groups: %w", mbid, err)
	}
	known, err := uc.releases.KnownMBIDs(ctx, mbid)
	if err != nil {
		return ArtistResult{}, fmt.Errorf("sync artist %s: %w", mbid, err)
	}
	s, err := uc.settings.Get(ctx)
	if err != nil {
		return ArtistResult{}, fmt.Errorf("sync artist %s: %w", mbid, err)
	}
	stale := s.StaleThreshold()

	now := uc.clock.Now()
	var links, fresh []domain.ArtistReleaseGroup
	for _, rg := range remote {
		if known[rg.MBID] {
			continue
		}
		known[rg.MBID] = true
		news := isNews(rg.FirstRelease, res.Baseline, now, stale)
		link := domain.ArtistReleaseGroup{
			ArtistMBID:   mbid,
			ReleaseGroup: rg,
			Role:         domain.RoleOf(rg.Credits, mbid),
			IsBaseline:   res.Baseline && !news,
			DiscoveredAt: now,
		}
		links = append(links, link)
		if news {
			fresh = append(fresh, link)
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
		if res.New, err = uc.notifier.Execute(ctx, mbid, fresh); err != nil {
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

// isNews decides whether a discovery goes to fan-out. Outside the baseline that is
// anything not stale (FR-3.4). The baseline keeps only dated releases inside the
// stale window: an undated one is as likely to be decades old as an announcement.
func isNews(d domain.ReleaseDate, baseline bool, now time.Time, stale time.Duration) bool {
	if baseline && d.Day == nil {
		return false
	}
	return !d.IsStale(now, stale)
}

// isBaseline: the artist's first sync, or the first one since a re-subscription —
// what appeared while nobody was subscribed is history, not news (ADR-3).
func (uc *Artist) isBaseline(ctx context.Context, a domain.Artist) (bool, error) {
	if a.LastSyncedAt == nil {
		return true, nil
	}
	sub, err := uc.subs.Get(ctx, a.MBID)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return sub.BaselineAt.After(*a.LastSyncedAt), nil
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
