package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
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

func toUUID(mbid domain.MBID) (uuid.UUID, error) {
	id, err := uuid.Parse(string(mbid))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("mbid %q: %w", mbid, err)
	}
	return id, nil
}

// optionalText maps "" to NULL.
func optionalText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
