// Package usecase contains application business operations. Ports (interfaces
// to the outside world) are declared here, next to their consumers; adapters
// implement them.
package usecase

import (
	"context"
	"time"
)

// TxManager runs fn inside a single database transaction. Repositories pick the
// transaction up from ctx, so use cases stay unaware of the storage details.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Clock abstracts time for deterministic tests.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production Clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
