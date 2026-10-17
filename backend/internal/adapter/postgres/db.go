// Package postgres implements repositories and transactions on PostgreSQL (pgx + sqlc).
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
)

// enumTypes are the custom types pgx must learn on connect: without them arrays
// of enums (e.g. release_primary_type[]) can be neither encoded nor scanned.
// Each enum is followed by its array type. Migrations must run before NewPool.
var enumTypes = []string{
	"release_primary_type", "_release_primary_type",
	"credit_role", "_credit_role",
	"notification_channel", "_notification_channel",
	"notification_status", "_notification_status",
	"digest_frequency", "_digest_frequency",
	"import_item_status", "_import_item_status",
}

// NewPool connects to PostgreSQL and verifies the connection.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		types, err := conn.LoadTypes(ctx, enumTypes)
		if err != nil {
			return fmt.Errorf("load enum types: %w", err)
		}
		conn.TypeMap().RegisterTypes(types)
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

type txKey struct{}

// TxManager implements usecase.TxManager. The active transaction travels in ctx.
type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager { return &TxManager{pool: pool} }

func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx) // already inside a transaction: join it
	}
	return pgx.BeginFunc(ctx, m.pool, func(tx pgx.Tx) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// queries returns sqlc queries bound to the transaction from ctx, or to the pool.
// Repositories call it at the start of every method.
func queries(ctx context.Context, pool *pgxpool.Pool) *sqlc.Queries {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return sqlc.New(tx)
	}
	return sqlc.New(pool)
}
