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

// ReleaseDetailsCache implements usecase.ReleaseDetailsCache. The domain value is
// stored as JSON as is: a payload that no longer decodes is a miss and gets refetched.
type ReleaseDetailsCache struct {
	pool *pgxpool.Pool
}

func NewReleaseDetailsCache(pool *pgxpool.Pool) *ReleaseDetailsCache {
	return &ReleaseDetailsCache{pool: pool}
}

func (c *ReleaseDetailsCache) Get(ctx context.Context, rg domain.MBID) (domain.ReleaseDetails, time.Time, error) {
	id, err := toUUID(rg)
	if err != nil {
		return domain.ReleaseDetails{}, time.Time{}, err
	}
	row, err := queries(ctx, c.pool).GetReleaseDetails(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReleaseDetails{}, time.Time{}, fmt.Errorf("release details of %s: %w", rg, domain.ErrNotFound)
	}
	if err != nil {
		return domain.ReleaseDetails{}, time.Time{}, fmt.Errorf("get release details of %s: %w", rg, err)
	}
	var d domain.ReleaseDetails
	if err := json.Unmarshal(row.Payload, &d); err != nil {
		return domain.ReleaseDetails{}, time.Time{}, fmt.Errorf("release details of %s: stale format: %w", rg, domain.ErrNotFound)
	}
	return d, row.FetchedAt.UTC(), nil
}

func (c *ReleaseDetailsCache) Put(ctx context.Context, rg domain.MBID, d domain.ReleaseDetails, at time.Time) error {
	id, err := toUUID(rg)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("encode release details of %s: %w", rg, err)
	}
	if err := queries(ctx, c.pool).PutReleaseDetails(ctx, sqlc.PutReleaseDetailsParams{ReleaseGroupMbid: id, Payload: raw, FetchedAt: at}); err != nil {
		return fmt.Errorf("store release details of %s: %w", rg, err)
	}
	return nil
}
