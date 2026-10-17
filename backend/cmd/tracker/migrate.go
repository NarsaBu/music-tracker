package main

import (
	"errors"
	"fmt"
	"os"

	"music-tracker/backend/internal/adapter/postgres"
)

// migrateCmd needs only DATABASE_URL, so it works without the full app config.
func migrateCmd(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: tracker migrate up|down|version")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required")
	}

	m, err := postgres.NewMigrator(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()

	switch args[0] {
	case "up":
		return m.Up()
	case "down":
		return m.Down()
	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			return err
		}
		fmt.Printf("version=%d dirty=%t\n", v, dirty)
		return nil
	default:
		return fmt.Errorf("unknown migrate command %q", args[0])
	}
}

func migrateUp(dsn string) (err error) {
	m, err := postgres.NewMigrator(dsn)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, m.Close()) }()
	if err := m.Up(); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
