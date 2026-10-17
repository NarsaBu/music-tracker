package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/domain"
)

// SettingsRepository reads and writes the singleton settings row.
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
