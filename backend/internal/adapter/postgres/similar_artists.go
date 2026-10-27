package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
	"music-tracker/backend/internal/domain"
)

// SimilarArtistsCache implements usecase.SimilarArtistsCache.
type SimilarArtistsCache struct {
	pool *pgxpool.Pool
}

func NewSimilarArtistsCache(pool *pgxpool.Pool) *SimilarArtistsCache {
	return &SimilarArtistsCache{pool: pool}
}

// similarArtistJSON is the stored shape of one item; keep it backward compatible.
type similarArtistJSON struct {
	MBID           string `json:"mbid"`
	Name           string `json:"name"`
	Disambiguation string `json:"disambiguation,omitempty"`
	Type           string `json:"type,omitempty"`
	Score          int    `json:"score"`
}

func (c *SimilarArtistsCache) Get(ctx context.Context, artist domain.MBID) ([]domain.SimilarArtist, time.Time, error) {
	id, err := toUUID(artist)
	if err != nil {
		return nil, time.Time{}, err
	}
	row, err := queries(ctx, c.pool).GetSimilarArtists(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, time.Time{}, fmt.Errorf("similar artists of %s: %w", artist, domain.ErrNotFound)
	}
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("get similar artists of %s: %w", artist, err)
	}
	var stored []similarArtistJSON
	if err := json.Unmarshal(row.Items, &stored); err != nil {
		return nil, time.Time{}, fmt.Errorf("decode similar artists of %s: %w", artist, err)
	}
	items := make([]domain.SimilarArtist, len(stored))
	for i, s := range stored {
		items[i] = domain.SimilarArtist{
			MBID: domain.MBID(s.MBID), Name: s.Name, Disambiguation: s.Disambiguation, Type: s.Type, Score: s.Score,
		}
	}
	return items, row.FetchedAt.UTC(), nil
}

func (c *SimilarArtistsCache) Put(ctx context.Context, artist domain.MBID, items []domain.SimilarArtist, at time.Time) error {
	id, err := toUUID(artist)
	if err != nil {
		return err
	}
	stored := make([]similarArtistJSON, len(items))
	for i, a := range items {
		stored[i] = similarArtistJSON{
			MBID: string(a.MBID), Name: a.Name, Disambiguation: a.Disambiguation, Type: a.Type, Score: a.Score,
		}
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("encode similar artists of %s: %w", artist, err)
	}
	err = queries(ctx, c.pool).PutSimilarArtists(ctx, sqlc.PutSimilarArtistsParams{ArtistMbid: id, Items: raw, FetchedAt: at})
	if err != nil {
		return fmt.Errorf("store similar artists of %s: %w", artist, err)
	}
	return nil
}
