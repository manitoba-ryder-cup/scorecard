package postgres

import (
	"context"

	"github.com/travisbale/knowhere/db/postgres"
	"github.com/travisbale/scorecard/internal/db/postgres/internal/sqlc"
)

// DB is the database instance for scorecard using knowhere's generic wrapper
type DB = postgres.DB[*sqlc.Queries]

// NewDB creates a new database connection using knowhere's database wrapper.
// This provides tenant context propagation via RLS and transaction management.
func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	// The newQ function wraps sqlc's constructor
	// d can be either *pgxpool.Pool or pgx.Tx, both implement sqlc.DBTX
	newQ := func(d any) *sqlc.Queries {
		return sqlc.New(d.(sqlc.DBTX))
	}

	// Use knowhere's default configuration (25 max conns, 5 min conns, etc.)
	return postgres.NewDB(ctx, databaseURL, newQ, nil)
}
