// Package postgres implements repositories and transactions on PostgreSQL (pgx + sqlc).
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-tracker/backend/internal/adapter/postgres/sqlc"
)

// NewPool connects to PostgreSQL and verifies the connection.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
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
//
//nolint:unused // used by repositories starting from stage 1
func queries(ctx context.Context, pool *pgxpool.Pool) *sqlc.Queries {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return sqlc.New(tx)
	}
	return sqlc.New(pool)
}
