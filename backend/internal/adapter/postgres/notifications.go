package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// NotificationRepository implements usecase.NotificationRepository (the outbox).
type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) Insert(ctx context.Context, n domain.Notification) error {
	rg, err := toUUID(n.ReleaseGroup)
	if err != nil {
		return err
	}
	artist, err := toUUID(n.Artist)
	if err != nil {
		return err
	}
	var sentAt *time.Time
	if n.Status == domain.StatusSent {
		sentAt = &n.At
	}
	err = queries(ctx, r.pool).InsertNotification(ctx, sqlc.InsertNotificationParams{
		ReleaseGroupMbid: rg,
		ArtistMbid:       artist,
		CreditRole:       sqlc.CreditRole(n.Role),
		Channel:          sqlc.NotificationChannel(n.Channel),
		Status:           sqlc.NotificationStatus(n.Status),
		CreatedAt:        n.At,
		SentAt:           sentAt,
	})
	if err != nil {
		return fmt.Errorf("insert %s notification for %s: %w", n.Channel, n.ReleaseGroup, err)
	}
	return nil
}

func (r *NotificationRepository) ClaimPendingEmail(ctx context.Context) ([]domain.NotificationItem, error) {
	rows, err := queries(ctx, r.pool).ClaimPendingEmail(ctx)
	if err != nil {
		return nil, fmt.Errorf("claim pending email: %w", err)
	}
	out := make([]domain.NotificationItem, len(rows))
	for i, row := range rows {
		out[i] = notificationItem(row.ID, row.CreditRole, row.CreatedAt, row.ReleaseGroup, row.Artist)
	}
	return out, nil
}

func (r *NotificationRepository) MarkSent(ctx context.Context, ids []string, at time.Time) error {
	uuids, err := parseIDs(ids)
	if err != nil {
		return err
	}
	if err := queries(ctx, r.pool).MarkNotificationsSent(ctx, sqlc.MarkNotificationsSentParams{At: &at, Ids: uuids}); err != nil {
		return fmt.Errorf("mark notifications sent: %w", err)
	}
	return nil
}

func (r *NotificationRepository) MarkFailed(ctx context.Context, ids []string, cause error) error {
	uuids, err := parseIDs(ids)
	if err != nil {
		return err
	}
	err = queries(ctx, r.pool).MarkNotificationsFailed(ctx, sqlc.MarkNotificationsFailedParams{
		LastError: errorText(cause), AttemptLimit: domain.MaxEmailAttempts, Ids: uuids,
	})
	if err != nil {
		return fmt.Errorf("mark notifications failed: %w", err)
	}
	return nil
}

func (r *NotificationRepository) ListFeed(ctx context.Context, q usecase.FeedQuery) ([]domain.NotificationItem, error) {
	primaryType := sqlc.NullReleasePrimaryType{ReleasePrimaryType: sqlc.ReleasePrimaryType(q.Type), Valid: q.Type != ""}
	role := sqlc.NullCreditRole{CreditRole: sqlc.CreditRole(q.Role), Valid: q.Role != ""}
	lim := int32(q.Limit) //nolint:gosec // bounded by the use case
	var (
		beforeAt *time.Time
		beforeID uuid.NullUUID
	)
	if q.Before != nil {
		id, err := uuid.Parse(q.Before.ID)
		if err != nil {
			return nil, fmt.Errorf("feed cursor: %w", err)
		}
		beforeAt, beforeID = &q.Before.At, uuid.NullUUID{UUID: id, Valid: true}
	}

	var out []domain.NotificationItem
	if q.Sort == usecase.FeedByRelease {
		rows, err := queries(ctx, r.pool).ListFeedByRelease(ctx, sqlc.ListFeedByReleaseParams{
			BeforeOn: beforeAt, BeforeID: beforeID, PrimaryType: primaryType, Role: role, Lim: lim,
		})
		if err != nil {
			return nil, fmt.Errorf("list feed: %w", err)
		}
		for _, row := range rows {
			out = append(out, notificationItem(row.ID, row.CreditRole, row.CreatedAt, row.ReleaseGroup, row.Artist))
		}
		return out, nil
	}
	rows, err := queries(ctx, r.pool).ListFeed(ctx, sqlc.ListFeedParams{
		BeforeAt: beforeAt, BeforeID: beforeID, PrimaryType: primaryType, Role: role, Lim: lim,
	})
	if err != nil {
		return nil, fmt.Errorf("list feed: %w", err)
	}
	for _, row := range rows {
		out = append(out, notificationItem(row.ID, row.CreditRole, row.CreatedAt, row.ReleaseGroup, row.Artist))
	}
	return out, nil
}

func notificationItem(id uuid.UUID, role sqlc.CreditRole, at time.Time, rg sqlc.ReleaseGroup, a sqlc.Artist) domain.NotificationItem {
	return domain.NotificationItem{
		ID:           id.String(),
		ReleaseGroup: releaseGroupFromRow(rg),
		FoundVia:     artistFromRow(a),
		Role:         domain.CreditRole(role),
		CreatedAt:    at.UTC(),
	}
}

func parseIDs(ids []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, len(ids))
	for i, s := range ids {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("notification id %q: %w", s, err)
		}
		out[i] = id
	}
	return out, nil
}
