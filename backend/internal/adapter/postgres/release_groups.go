package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
	"music-tracker/backend/internal/domain"
)

// ReleaseGroupRepository implements usecase.ReleaseGroupRepository.
type ReleaseGroupRepository struct {
	pool *pgxpool.Pool
}

func NewReleaseGroupRepository(pool *pgxpool.Pool) *ReleaseGroupRepository {
	return &ReleaseGroupRepository{pool: pool}
}

func (r *ReleaseGroupRepository) KnownMBIDs(ctx context.Context, artist domain.MBID) (map[domain.MBID]bool, error) {
	id, err := toUUID(artist)
	if err != nil {
		return nil, err
	}
	ids, err := queries(ctx, r.pool).KnownReleaseGroups(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("known release groups of %s: %w", artist, err)
	}
	out := make(map[domain.MBID]bool, len(ids))
	for _, m := range toMBIDs(ids) {
		out[m] = true
	}
	return out, nil
}

func (r *ReleaseGroupRepository) UpsertMany(ctx context.Context, rgs []domain.ReleaseGroup, at time.Time) error {
	params := make([]sqlc.UpsertReleaseGroupParams, 0, len(rgs))
	for _, rg := range rgs {
		id, err := toUUID(rg.MBID)
		if err != nil {
			return err
		}
		params = append(params, sqlc.UpsertReleaseGroupParams{
			Mbid:             id,
			Title:            rg.Title,
			PrimaryType:      sqlc.NullReleasePrimaryType{ReleasePrimaryType: sqlc.ReleasePrimaryType(rg.PrimaryType), Valid: rg.PrimaryType != ""},
			SecondaryTypes:   nonNil(rg.SecondaryTypes),
			FirstReleaseDate: optionalText(rg.FirstRelease.Raw),
			FirstReleaseOn:   rg.FirstRelease.Day,
			ArtistCredit:     rg.ArtistCredit,
			DiscoveredAt:     at,
		})
	}
	return execBatch("upsert release groups", queries(ctx, r.pool).UpsertReleaseGroup(ctx, params))
}

func (r *ReleaseGroupRepository) LinkMany(ctx context.Context, links []domain.ArtistReleaseGroup) error {
	params := make([]sqlc.LinkArtistReleaseGroupParams, 0, len(links))
	for _, l := range links {
		artist, err := toUUID(l.ArtistMBID)
		if err != nil {
			return err
		}
		rg, err := toUUID(l.ReleaseGroup.MBID)
		if err != nil {
			return err
		}
		params = append(params, sqlc.LinkArtistReleaseGroupParams{
			ArtistMbid:       artist,
			ReleaseGroupMbid: rg,
			CreditRole:       sqlc.CreditRole(l.Role),
			IsBaseline:       l.IsBaseline,
			DiscoveredAt:     l.DiscoveredAt,
		})
	}
	return execBatch("link release groups", queries(ctx, r.pool).LinkArtistReleaseGroup(ctx, params))
}

func (r *ReleaseGroupRepository) ListByArtist(ctx context.Context, artist domain.MBID, role domain.CreditRole) ([]domain.ArtistReleaseGroup, error) {
	id, err := toUUID(artist)
	if err != nil {
		return nil, err
	}
	rows, err := queries(ctx, r.pool).ListArtistReleaseGroups(ctx, sqlc.ListArtistReleaseGroupsParams{
		ArtistMbid: id,
		Role:       sqlc.NullCreditRole{CreditRole: sqlc.CreditRole(role), Valid: role != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("list release groups of %s: %w", artist, err)
	}
	out := make([]domain.ArtistReleaseGroup, len(rows))
	for i, row := range rows {
		out[i] = domain.ArtistReleaseGroup{
			ArtistMBID:   artist,
			ReleaseGroup: releaseGroupFromRow(row.ReleaseGroup),
			Role:         domain.CreditRole(row.CreditRole),
			IsBaseline:   row.IsBaseline,
			DiscoveredAt: row.LinkedAt.UTC(),
		}
	}
	return out, nil
}

func releaseGroupFromRow(rg sqlc.ReleaseGroup) domain.ReleaseGroup {
	var pt domain.PrimaryType
	if rg.PrimaryType.Valid {
		pt = domain.PrimaryType(rg.PrimaryType.ReleasePrimaryType)
	}
	return domain.ReleaseGroup{
		MBID:           domain.MBID(rg.Mbid.String()),
		Title:          rg.Title,
		PrimaryType:    pt,
		SecondaryTypes: nonNil(rg.SecondaryTypes),
		FirstRelease:   domain.ParseReleaseDate(rg.FirstReleaseDate.String),
		ArtistCredit:   rg.ArtistCredit,
	}
}

// batchResults is what sqlc generates for :batchexec queries.
type batchResults interface {
	Exec(f func(int, error))
	Close() error
}

// execBatch runs all queued statements and returns the first error.
func execBatch(what string, b batchResults) error {
	var errs []error
	b.Exec(func(i int, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: item %d: %w", what, i, err))
		}
	})
	return errors.Join(append(errs, b.Close())...)
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
