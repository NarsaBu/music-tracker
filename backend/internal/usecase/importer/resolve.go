package importer

import (
	"context"
	"errors"
	"fmt"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

type ResolveInput struct {
	JobID  string
	ItemID string
	// MBID (or a MusicBrainz link) of the chosen artist: a candidate or any other.
	MBID string
	// IncludeFeatured overrides the CSV value; nil keeps it.
	IncludeFeatured *bool
	// Skip dismisses the line instead.
	Skip bool
}

// ResolveItem settles a line the worker could not match (US-3): the user picks
// a candidate (or gives an MBID) to subscribe to, or skips the line.
type ResolveItem struct {
	catalog    usecase.MusicCatalog
	imports    usecase.ImportRepository
	subscriber Subscriber
	tx         usecase.TxManager
}

func NewResolveItem(
	catalog usecase.MusicCatalog,
	imports usecase.ImportRepository,
	subscriber Subscriber,
	tx usecase.TxManager,
) *ResolveItem {
	return &ResolveItem{catalog: catalog, imports: imports, subscriber: subscriber, tx: tx}
}

func (uc *ResolveItem) Execute(ctx context.Context, in ResolveInput) (domain.ImportItem, error) {
	var mbid domain.MBID
	switch {
	case in.Skip && in.MBID != "":
		return domain.ImportItem{}, resolveError("skip", "either mbid or skip, not both")
	case !in.Skip:
		var ok bool
		if mbid, ok = domain.ParseArtistRef(in.MBID); !ok {
			return domain.ImportItem{}, resolveError("mbid", "must be an MBID or a MusicBrainz artist link")
		}
	}

	item, err := uc.resolvable(ctx, in)
	if err != nil {
		return domain.ImportItem{}, err
	}
	var artist *domain.Artist
	if !in.Skip {
		a, err := uc.artist(ctx, item, mbid)
		if err != nil {
			return domain.ImportItem{}, err
		}
		artist = &a
	}

	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		// Re-read under the lock: a concurrent resolve may have won.
		if item, err = uc.resolvable(ctx, in); err != nil {
			return err
		}
		item.Status, item.Error = domain.ImportSkipped, ""
		if artist != nil {
			if _, err := uc.subscriber.SubscribeArtist(ctx, *artist, overrideOf(item, in.IncludeFeatured)); err != nil {
				return err
			}
			item.Status, item.Artist = domain.ImportMatched, artist
		}
		return uc.imports.UpdateItem(ctx, item)
	})
	if err != nil {
		return domain.ImportItem{}, fmt.Errorf("resolve import item %s: %w", in.ItemID, err)
	}
	return item, nil
}

func (uc *ResolveItem) resolvable(ctx context.Context, in ResolveInput) (domain.ImportItem, error) {
	item, err := uc.imports.GetItem(ctx, in.JobID, in.ItemID)
	if err != nil {
		return domain.ImportItem{}, fmt.Errorf("import item %s: %w", in.ItemID, err)
	}
	if !item.Status.Resolvable() {
		return domain.ImportItem{}, fmt.Errorf("import item %s is %s: %w", in.ItemID, item.Status, domain.ErrConflict)
	}
	return item, nil
}

// artist takes the card from the candidates when possible, sparing a catalog request.
func (uc *ResolveItem) artist(ctx context.Context, item domain.ImportItem, mbid domain.MBID) (domain.Artist, error) {
	for _, c := range item.Candidates {
		if c.MBID == mbid {
			return c.Artist, nil
		}
	}
	a, err := uc.catalog.GetArtist(ctx, mbid)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Artist{}, resolveError("mbid", "MusicBrainz does not know this artist")
	}
	if err != nil {
		return domain.Artist{}, fmt.Errorf("get artist %s: %w", mbid, err)
	}
	return a, nil
}

func resolveError(field, msg string) error {
	return &domain.ValidationError{Fields: []domain.FieldError{{Field: field, Message: msg}}}
}
