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

// ReviewCache implements usecase.ReviewCache.
type ReviewCache struct {
	pool *pgxpool.Pool
}

func NewReviewCache(pool *pgxpool.Pool) *ReviewCache { return &ReviewCache{pool: pool} }

// reviewsJSON is the stored shape; keep it backward compatible.
type reviewsJSON struct {
	Items   []reviewJSON `json:"items"`
	Total   int          `json:"total"`
	Average float64      `json:"average,omitempty"`
	Rated   int          `json:"rated,omitempty"`
}

type reviewJSON struct {
	ID          string    `json:"id"`
	Author      string    `json:"author"`
	Rating      int       `json:"rating,omitempty"`
	Text        string    `json:"text"`
	Language    string    `json:"language,omitempty"`
	PublishedAt time.Time `json:"published_at"`
	License     string    `json:"license,omitempty"`
}

func (c *ReviewCache) Get(ctx context.Context, rg domain.MBID) (domain.Reviews, time.Time, error) {
	id, err := toUUID(rg)
	if err != nil {
		return domain.Reviews{}, time.Time{}, err
	}
	row, err := queries(ctx, c.pool).GetReleaseReviews(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Reviews{}, time.Time{}, fmt.Errorf("reviews of %s: %w", rg, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Reviews{}, time.Time{}, fmt.Errorf("get reviews of %s: %w", rg, err)
	}
	var stored reviewsJSON
	if err := json.Unmarshal(row.Payload, &stored); err != nil {
		return domain.Reviews{}, time.Time{}, fmt.Errorf("decode reviews of %s: %w", rg, err)
	}
	out := domain.Reviews{Items: make([]domain.Review, len(stored.Items)), Total: stored.Total, Average: stored.Average, Rated: stored.Rated}
	for i, r := range stored.Items {
		out.Items[i] = domain.Review(r)
	}
	return out, row.FetchedAt.UTC(), nil
}

func (c *ReviewCache) Put(ctx context.Context, rg domain.MBID, reviews domain.Reviews, at time.Time) error {
	id, err := toUUID(rg)
	if err != nil {
		return err
	}
	stored := reviewsJSON{Items: make([]reviewJSON, len(reviews.Items)), Total: reviews.Total, Average: reviews.Average, Rated: reviews.Rated}
	for i, r := range reviews.Items {
		stored.Items[i] = reviewJSON(r)
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("encode reviews of %s: %w", rg, err)
	}
	if err := queries(ctx, c.pool).PutReleaseReviews(ctx, sqlc.PutReleaseReviewsParams{ReleaseGroupMbid: id, Payload: raw, FetchedAt: at}); err != nil {
		return fmt.Errorf("store reviews of %s: %w", rg, err)
	}
	return nil
}
