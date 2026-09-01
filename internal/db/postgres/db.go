package postgres

import (
	"context"

	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/travisbale/knowhere/db/postgres"
)

// DB is the database instance for scorecard using knowhere's generic wrapper
type DB = postgres.DB[*sqlc.Queries]

// NewDB builds a connection pool using knowhere's database wrapper. It does not reach the
// database; the pool dials as it is used.
// This provides tenant context propagation via RLS and transaction management.
func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	// d is a pool or a transaction; both satisfy sqlc.DBTX.
	newQ := func(d any) *sqlc.Queries {
		return sqlc.New(d.(sqlc.DBTX))
	}

	// Use knowhere's default configuration (25 max conns, 5 min conns, etc.)
	return postgres.NewDB(ctx, databaseURL, newQ, nil)
}
