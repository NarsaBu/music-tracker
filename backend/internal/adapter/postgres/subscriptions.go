package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// SubscriptionRepository implements usecase.SubscriptionRepository.
type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

func (r *SubscriptionRepository) Create(ctx context.Context, s domain.Subscription) (bool, error) {
	id, err := toUUID(s.Artist.MBID)
	if err != nil {
		return false, err
	}
	n, err := queries(ctx, r.pool).InsertSubscription(ctx, sqlc.InsertSubscriptionParams{
		ArtistMbid:             id,
		PrimaryTypes:           toPrimaryTypes(s.Override.PrimaryTypes),
		ExcludedSecondaryTypes: s.Override.ExcludedSecondaryTypes,
		IncludeFeatured:        optionalBool(s.Override.IncludeFeatured),
		BaselineAt:             s.BaselineAt,
		CreatedAt:              s.CreatedAt,
	})
	if err != nil {
		return false, fmt.Errorf("insert subscription %s: %w", s.Artist.MBID, err)
	}
	return n == 1, nil
}

func (r *SubscriptionRepository) Get(ctx context.Context, artist domain.MBID) (domain.Subscription, error) {
	id, err := toUUID(artist)
	if err != nil {
		return domain.Subscription{}, err
	}
	row, err := queries(ctx, r.pool).GetSubscription(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Subscription{}, fmt.Errorf("subscription %s: %w", artist, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("get subscription %s: %w", artist, err)
	}
	return subscriptionFromRow(row.Subscription, row.Artist), nil
}

func (r *SubscriptionRepository) Delete(ctx context.Context, artist domain.MBID) error {
	id, err := toUUID(artist)
	if err != nil {
		return err
	}
	n, err := queries(ctx, r.pool).DeleteSubscription(ctx, id)
	if err != nil {
		return fmt.Errorf("delete subscription %s: %w", artist, err)
	}
	if n == 0 {
		return fmt.Errorf("subscription %s: %w", artist, domain.ErrNotFound)
	}
	return nil
}

func (r *SubscriptionRepository) List(ctx context.Context, q usecase.SubscriptionListQuery) ([]domain.Subscription, int, error) {
	search := pgtype.Text{String: escapeLike(q.Search), Valid: q.Search != ""}
	qs := queries(ctx, r.pool)

	rows, err := qs.ListSubscriptions(ctx, sqlc.ListSubscriptionsParams{
		Limit: int32(q.Limit), Offset: int32(q.Offset), Search: search, //nolint:gosec // bounded by the use case
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list subscriptions: %w", err)
	}
	total, err := qs.CountSubscriptions(ctx, search)
	if err != nil {
		return nil, 0, fmt.Errorf("count subscriptions: %w", err)
	}

	out := make([]domain.Subscription, len(rows))
	for i, row := range rows {
		out[i] = subscriptionFromRow(row.Subscription, row.Artist)
	}
	return out, int(total), nil
}

func (r *SubscriptionRepository) UpdateOverride(ctx context.Context, artist domain.MBID, o domain.FilterOverride) error {
	id, err := toUUID(artist)
	if err != nil {
		return err
	}
	n, err := queries(ctx, r.pool).UpdateSubscriptionOverride(ctx, sqlc.UpdateSubscriptionOverrideParams{
		ArtistMbid:             id,
		PrimaryTypes:           toPrimaryTypes(o.PrimaryTypes),
		ExcludedSecondaryTypes: o.ExcludedSecondaryTypes,
		IncludeFeatured:        optionalBool(o.IncludeFeatured),
	})
	if err != nil {
		return fmt.Errorf("update subscription %s: %w", artist, err)
	}
	if n == 0 {
		return fmt.Errorf("subscription %s: %w", artist, domain.ErrNotFound)
	}
	return nil
}

func (r *SubscriptionRepository) SubscribedAmong(ctx context.Context, artists []domain.MBID) (map[domain.MBID]bool, error) {
	ids := make([]uuid.UUID, 0, len(artists))
	for _, a := range artists {
		id, err := toUUID(a)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	found, err := queries(ctx, r.pool).SubscribedAmong(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("subscribed among: %w", err)
	}
	out := make(map[domain.MBID]bool, len(found))
	for _, id := range found {
		out[domain.MBID(id.String())] = true
	}
	return out, nil
}

func subscriptionFromRow(s sqlc.Subscription, a sqlc.Artist) domain.Subscription {
	return domain.Subscription{
		Artist: artistFromRow(a),
		Override: domain.FilterOverride{
			PrimaryTypes:           fromPrimaryTypes(s.PrimaryTypes),
			ExcludedSecondaryTypes: s.ExcludedSecondaryTypes,
			IncludeFeatured:        nullableBool(s.IncludeFeatured),
		},
		BaselineAt: s.BaselineAt.UTC(),
		CreatedAt:  s.CreatedAt.UTC(),
	}
}

// toPrimaryTypes keeps nil as nil: a NULL column means "no override".
func toPrimaryTypes(ts []domain.PrimaryType) []sqlc.ReleasePrimaryType {
	if ts == nil {
		return nil
	}
	out := make([]sqlc.ReleasePrimaryType, len(ts))
	for i, t := range ts {
		out[i] = sqlc.ReleasePrimaryType(t)
	}
	return out
}

func fromPrimaryTypes(ts []sqlc.ReleasePrimaryType) []domain.PrimaryType {
	if ts == nil {
		return nil
	}
	out := make([]domain.PrimaryType, len(ts))
	for i, t := range ts {
		out[i] = domain.PrimaryType(t)
	}
	return out
}

func optionalBool(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

func nullableBool(b pgtype.Bool) *bool {
	if !b.Valid {
		return nil
	}
	return &b.Bool
}

// escapeLike makes s match literally inside a LIKE pattern (default escape character is \).
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string { return likeEscaper.Replace(s) }
