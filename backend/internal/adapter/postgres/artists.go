package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
	"music-tracker/backend/internal/domain"
)

// ArtistRepository implements usecase.ArtistRepository.
type ArtistRepository struct {
	pool *pgxpool.Pool
}

func NewArtistRepository(pool *pgxpool.Pool) *ArtistRepository { return &ArtistRepository{pool: pool} }

func (r *ArtistRepository) Upsert(ctx context.Context, a domain.Artist) error {
	id, err := toUUID(a.MBID)
	if err != nil {
		return err
	}
	err = queries(ctx, r.pool).UpsertArtist(ctx, sqlc.UpsertArtistParams{
		Mbid:           id,
		Name:           a.Name,
		SortName:       a.SortName,
		Disambiguation: a.Disambiguation,
		Type:           optionalText(a.Type),
		Country:        optionalText(a.Country),
	})
	if err != nil {
		return fmt.Errorf("upsert artist %s: %w", a.MBID, err)
	}
	return nil
}

func (r *ArtistRepository) Get(ctx context.Context, mbid domain.MBID) (domain.Artist, error) {
	id, err := toUUID(mbid)
	if err != nil {
		return domain.Artist{}, err
	}
	row, err := queries(ctx, r.pool).GetArtist(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Artist{}, fmt.Errorf("artist %s: %w", mbid, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Artist{}, fmt.Errorf("get artist %s: %w", mbid, err)
	}
	return artistFromRow(row), nil
}

func (r *ArtistRepository) ListTracked(ctx context.Context) ([]domain.MBID, error) {
	ids, err := queries(ctx, r.pool).ListTrackedArtists(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tracked artists: %w", err)
	}
	return toMBIDs(ids), nil
}

func (r *ArtistRepository) MarkSynced(ctx context.Context, mbid domain.MBID, at time.Time) error {
	id, err := toUUID(mbid)
	if err != nil {
		return err
	}
	if err := queries(ctx, r.pool).MarkArtistSynced(ctx, sqlc.MarkArtistSyncedParams{Mbid: id, LastSyncedAt: &at}); err != nil {
		return fmt.Errorf("mark artist %s synced: %w", mbid, err)
	}
	return nil
}

func artistFromRow(a sqlc.Artist) domain.Artist {
	return domain.Artist{
		MBID:           domain.MBID(a.Mbid.String()),
		Name:           a.Name,
		SortName:       a.SortName,
		Disambiguation: a.Disambiguation,
		Type:           a.Type.String,
		Country:        a.Country.String,
		LastSyncedAt:   utcPtr(a.LastSyncedAt),
	}
}

// pgx scans timestamptz in the local zone; the app works in UTC (docs/03-backend.md §4).
func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

func toMBIDs(ids []uuid.UUID) []domain.MBID {
	out := make([]domain.MBID, len(ids))
	for i, id := range ids {
		out[i] = domain.MBID(id.String())
	}
	return out
}

func toUUID(mbid domain.MBID) (uuid.UUID, error) {
	id, err := uuid.Parse(string(mbid))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("mbid %q: %w", mbid, err)
	}
	return id, nil
}

// optionalText maps "" to NULL.
func optionalText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
