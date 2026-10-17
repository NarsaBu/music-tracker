package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers pgx5:// driver
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"music-tracker/backend/migrations"
)

// Migrator applies the SQL migrations embedded into the binary.
type Migrator struct {
	m *migrate.Migrate
}

func NewMigrator(dsn string) (*Migrator, error) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("open embedded migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, toPgx5URL(dsn))
	if err != nil {
		return nil, fmt.Errorf("init migrate: %w", err)
	}
	return &Migrator{m: m}, nil
}

func (mg *Migrator) Up() error   { return ignoreNoChange(mg.m.Up()) }
func (mg *Migrator) Down() error { return ignoreNoChange(mg.m.Steps(-1)) }

func (mg *Migrator) Version() (version uint, dirty bool, err error) {
	v, d, err := mg.m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	return v, d, err
}

func (mg *Migrator) Close() error {
	srcErr, dbErr := mg.m.Close()
	return errors.Join(srcErr, dbErr)
}

func ignoreNoChange(err error) error {
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}

// toPgx5URL converts a standard postgres:// DSN to the scheme golang-migrate's pgx/v5 driver expects.
func toPgx5URL(dsn string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if rest, ok := strings.CutPrefix(dsn, prefix); ok {
			return "pgx5://" + rest
		}
	}
	return dsn
}
