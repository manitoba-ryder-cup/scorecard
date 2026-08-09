package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/travisbale/knowhere/identity"
)

// txKey carries the transaction an enclosing WithinTenantTx opened, so repo methods
// running under it join that transaction instead of starting their own.
type txKey struct{}

// WithinTenantTx runs fn inside one tenant-scoped transaction, so every repository call
// fn makes commits or rolls back together. Without it each call is its own transaction,
// which is right for a request handler — one edit, one unit of work — but wrong for a
// multi-step operation like seeding a tournament, where failing partway leaves the event
// half-built and a rerun creates a second one.
//
// Nesting is a no-op rather than an error: an inner call joins the outer transaction,
// because rolling back only the inner part would leave exactly the split this prevents.
func WithinTenantTx(ctx context.Context, db *DB, fn func(context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*sqlc.Queries); ok {
		return fn(ctx)
	}
	return db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		return fn(context.WithValue(ctx, txKey{}, q))
	})
}

// withTenant resolves the request's tenant and runs fn inside a single tenant-scoped
// transaction, returning fn's result. It collapses the GetTenant -> WithTenantContext
// -> assign-through-pointer boilerplate that every tenant-scoped repo method repeated.
func withTenant[T any](ctx context.Context, db *DB, fn func(q *sqlc.Queries, tenantID uuid.UUID) (T, error)) (T, error) {
	var result T
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return result, err
	}
	// Already inside a WithinTenantTx: reuse its transaction, which has the tenant set.
	if q, ok := ctx.Value(txKey{}).(*sqlc.Queries); ok {
		return fn(q, tenantID)
	}
	err = db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		r, e := fn(q, tenantID)
		if e != nil {
			return e
		}
		result = r
		return nil
	})
	return result, err
}

// withTenantExec is withTenant for methods that produce no value — a tenant-scoped
// transaction that only reports success or failure (upserts, deletes).
func withTenantExec(ctx context.Context, db *DB, fn func(q *sqlc.Queries, tenantID uuid.UUID) error) error {
	_, err := withTenant(ctx, db, func(q *sqlc.Queries, tenantID uuid.UUID) (struct{}, error) {
		return struct{}{}, fn(q, tenantID)
	})
	return err
}

// mapSlice converts a slice of sqlc rows to domain values with a per-row mapper,
// replacing the make+for loop each list repo repeated.
func mapSlice[T, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}
