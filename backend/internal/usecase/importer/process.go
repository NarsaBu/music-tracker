package importer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// candidateLimit is how many search results a line keeps for review.
const candidateLimit = 5

// ProcessItem matches one pending line: an MBID is looked up, a name is searched
// and subscribed only when the match is confident (domain.AutoMatch); otherwise
// the candidates wait for the user (US-3).
type ProcessItem struct {
	catalog    usecase.MusicCatalog
	imports    usecase.ImportRepository
	subscriber Subscriber
	tx         usecase.TxManager
	clock      usecase.Clock
}

func NewProcessItem(
	catalog usecase.MusicCatalog,
	imports usecase.ImportRepository,
	subscriber Subscriber,
	tx usecase.TxManager,
	clock usecase.Clock,
) *ProcessItem {
	return &ProcessItem{catalog: catalog, imports: imports, subscriber: subscriber, tx: tx, clock: clock}
}

// Execute is idempotent: an item that is no longer pending is left alone.
func (uc *ProcessItem) Execute(ctx context.Context, itemID string) error {
	item, err := uc.imports.GetItem(ctx, "", itemID)
	if err != nil {
		return fmt.Errorf("import item %s: %w", itemID, err)
	}
	if item.Status != domain.ImportPending {
		return nil
	}

	// MusicBrainz is asked outside the transaction: under the rate limit it may take a while.
	outcome, artist, err := uc.match(ctx, item)
	if err != nil {
		return fmt.Errorf("import item %s: %w", itemID, err)
	}
	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		locked, err := uc.imports.GetItem(ctx, "", itemID)
		if err != nil || locked.Status != domain.ImportPending {
			return err
		}
		if artist != nil {
			if _, err := uc.subscriber.SubscribeArtist(ctx, *artist, overrideOf(item, nil)); err != nil {
				return err
			}
		}
		if err := uc.imports.UpdateItem(ctx, outcome); err != nil {
			return err
		}
		return uc.imports.FinishIfDone(ctx, item.JobID, uc.clock.Now())
	})
	if err != nil {
		return fmt.Errorf("import item %s: %w", itemID, err)
	}
	return nil
}

// match decides the outcome; artist is set when the line is to be subscribed.
func (uc *ProcessItem) match(ctx context.Context, item domain.ImportItem) (domain.ImportItem, *domain.Artist, error) {
	if item.InputMBID != "" {
		a, err := uc.catalog.GetArtist(ctx, item.InputMBID)
		if errors.Is(err, domain.ErrNotFound) {
			item.Status, item.Error = domain.ImportNotFound, "MusicBrainz does not know this MBID"
			return item, nil, nil
		}
		if err != nil {
			return item, nil, err
		}
		item.Status, item.Artist = domain.ImportMatched, &a
		return item, &a, nil
	}

	found, err := uc.catalog.SearchArtists(ctx, item.RawValue, candidateLimit)
	if err != nil {
		return item, nil, err
	}
	if best, ok := domain.AutoMatch(found); ok {
		item.Status, item.Artist = domain.ImportMatched, &best.Artist
		return item, &best.Artist, nil
	}
	item.Candidates = found
	item.Status = domain.ImportNeedsReview
	if len(found) == 0 {
		item.Status = domain.ImportNotFound
	}
	return item, nil, nil
}

// HandleJob runs a usecase.JobImportItem job. When the job gives up for good the
// line is marked failed, so that the import can finish and the user can resolve it.
func (uc *ProcessItem) HandleJob(ctx context.Context, payload []byte, lastAttempt bool) error {
	var p usecase.ImportItemPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return &usecase.PermanentError{Err: fmt.Errorf("decode payload: %w", err)}
	}
	err := uc.Execute(ctx, p.ItemID)
	if errors.Is(err, domain.ErrNotFound) {
		return &usecase.PermanentError{Err: err}
	}
	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		err = &usecase.PermanentError{Err: err}
	}
	var permanent *usecase.PermanentError
	if err != nil && (lastAttempt || errors.As(err, &permanent)) {
		if failErr := uc.fail(ctx, p.ItemID, err); failErr != nil {
			return errors.Join(err, failErr)
		}
	}
	return err
}

func (uc *ProcessItem) fail(ctx context.Context, itemID string, cause error) error {
	return uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		item, err := uc.imports.GetItem(ctx, "", itemID)
		if err != nil || item.Status != domain.ImportPending {
			return err
		}
		item.Status, item.Error = domain.ImportFailed, cause.Error()
		if err := uc.imports.UpdateItem(ctx, item); err != nil {
			return err
		}
		return uc.imports.FinishIfDone(ctx, item.JobID, uc.clock.Now())
	})
}

// overrideOf: the CSV include_featured column, unless the user chose otherwise.
func overrideOf(item domain.ImportItem, includeFeatured *bool) domain.FilterOverride {
	if includeFeatured == nil {
		includeFeatured = item.IncludeFeatured
	}
	return domain.FilterOverride{IncludeFeatured: includeFeatured}
}
