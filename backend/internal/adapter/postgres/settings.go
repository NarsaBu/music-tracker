package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
	"music-tracker/backend/internal/domain"
)

// SettingsRepository implements usecase.SettingsRepository on the singleton settings row.
type SettingsRepository struct {
	pool *pgxpool.Pool
}

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

// FilterDefaults implements usecase.FilterDefaultsReader.
func (r *SettingsRepository) FilterDefaults(ctx context.Context) (domain.ReleaseFilter, error) {
	row, err := queries(ctx, r.pool).GetFilterDefaults(ctx)
	if err != nil {
		return domain.ReleaseFilter{}, fmt.Errorf("get filter defaults: %w", err)
	}
	return domain.ReleaseFilter{
		PrimaryTypes:           fromPrimaryTypes(row.PrimaryTypes),
		ExcludedSecondaryTypes: row.ExcludedSecondaryTypes,
		IncludeFeatured:        row.IncludeFeatured,
	}, nil
}

func (r *SettingsRepository) Get(ctx context.Context) (domain.Settings, error) {
	row, err := queries(ctx, r.pool).GetSettings(ctx)
	if err != nil {
		return domain.Settings{}, fmt.Errorf("get settings: %w", err)
	}
	// The weekday is NULL until a weekly digest is chosen; Friday is the UI default.
	weekday := time.Friday
	if row.DigestWeekday.Valid {
		weekday = time.Weekday(row.DigestWeekday.Int16)
	}
	minutes := row.DigestTime.Microseconds / int64(time.Minute/time.Microsecond)
	return domain.Settings{
		Filter: domain.ReleaseFilter{
			PrimaryTypes:           fromPrimaryTypes(row.PrimaryTypes),
			ExcludedSecondaryTypes: row.ExcludedSecondaryTypes,
			IncludeFeatured:        row.IncludeFeatured,
		},
		NotifyAnnouncements: row.NotifyAnnouncements,
		NewReleaseDays:      int(row.NewReleaseDays),
		Email: domain.EmailSettings{
			Enabled: row.EmailEnabled,
			To:      row.EmailTo.String,
			Digest: domain.DigestSchedule{
				Frequency: domain.DigestFrequency(row.DigestFrequency),
				Weekday:   weekday,
				Hour:      int(minutes / 60),
				Minute:    int(minutes % 60),
				Timezone:  row.Timezone,
			},
		},
		RSS:          domain.RSSSettings{Enabled: row.RssEnabled, FeedToken: row.FeedToken},
		LastDigestAt: utcPtr(row.LastDigestAt),
		NextDigestAt: utcPtr(row.NextDigestAt),
	}, nil
}

func (r *SettingsRepository) Save(ctx context.Context, s domain.Settings) error {
	d := s.Email.Digest
	err := queries(ctx, r.pool).UpdateSettings(ctx, sqlc.UpdateSettingsParams{
		PrimaryTypes:           toPrimaryTypes(s.Filter.PrimaryTypes),
		ExcludedSecondaryTypes: nonNil(s.Filter.ExcludedSecondaryTypes),
		IncludeFeatured:        s.Filter.IncludeFeatured,
		NotifyAnnouncements:    s.NotifyAnnouncements,
		NewReleaseDays:         int32(s.NewReleaseDays), //nolint:gosec // 1–3650, validated
		EmailEnabled:           s.Email.Enabled,
		EmailTo:                optionalText(s.Email.To),
		DigestFrequency:        sqlc.DigestFrequency(d.Frequency),
		DigestWeekday:          pgtype.Int2{Int16: int16(d.Weekday), Valid: true}, //nolint:gosec // 0–6, validated
		DigestTime:             pgtype.Time{Microseconds: int64(d.Hour*60+d.Minute) * int64(time.Minute/time.Microsecond), Valid: true},
		Timezone:               d.Timezone,
		NextDigestAt:           s.NextDigestAt,
		RssEnabled:             s.RSS.Enabled,
	})
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}

func (r *SettingsRepository) SetFeedToken(ctx context.Context, token string) error {
	if err := queries(ctx, r.pool).SetFeedToken(ctx, token); err != nil {
		return fmt.Errorf("set feed token: %w", err)
	}
	return nil
}

func (r *SettingsRepository) SetDigestTimes(ctx context.Context, last, next *time.Time) error {
	if err := queries(ctx, r.pool).SetDigestTimes(ctx, sqlc.SetDigestTimesParams{LastDigestAt: last, NextDigestAt: next}); err != nil {
		return fmt.Errorf("set digest times: %w", err)
	}
	return nil
}
