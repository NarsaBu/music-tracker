//go:build integration

package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// testPool is a migrated database in a throwaway container, shared by the package tests.
var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	os.Exit(runWithDatabase(m))
}

func runWithDatabase(m *testing.M) int {
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("tracker"),
		tcpostgres.WithUsername("tracker"),
		tcpostgres.WithPassword("tracker"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start postgres container:", err)
		return 1
	}
	defer func() { _ = ctr.Terminate(ctx) }()

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, "connection string:", err)
		return 1
	}
	mg, err := NewMigrator(dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := mg.Up(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate up:", err)
		return 1
	}
	_ = mg.Close()

	testPool, err = NewPool(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer testPool.Close()
	return m.Run()
}

// resetTables empties the tables a test touches.
func resetTables(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), "TRUNCATE subscriptions, artists"); err != nil {
		t.Fatal(err)
	}
}
