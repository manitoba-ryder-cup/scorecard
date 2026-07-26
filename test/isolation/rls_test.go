package isolation

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestRLSHidesOtherTenantsRowsFromTheAppRole drops the tenant predicate every query
// carries and asks what the database alone hands back. Failing here means RLS is inert
// and those predicates are all that prevent a leak — one forgotten WHERE away.
func TestRLSHidesOtherTenantsRowsFromTheAppRole(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Tenant A owns a tournament; tenant B owns nothing.
	seedFullFixture(t, ctx, connectSuperuser(t, ctx))
	tenantB := uuid.New()

	appConn := connectAppRole(t, ctx)

	for _, table := range tenantScopedTables {
		t.Run(table, func(t *testing.T) {
			visible := countVisibleRows(t, ctx, appConn, tenantB, table)
			if visible != 0 {
				t.Errorf("tenant B sees %d row(s) in %s with no tenant predicate; want 0 (RLS is not filtering)", visible, table)
			}
		})
	}
}

// TestRLSStillShowsATenantItsOwnRows guards the other direction — without it, revoking
// the app role's access outright would "pass" the test above.
func TestRLSStillShowsATenantItsOwnRows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fix := seedFullFixture(t, ctx, connectSuperuser(t, ctx))
	appConn := connectAppRole(t, ctx)

	for _, table := range tenantScopedTables {
		t.Run(table, func(t *testing.T) {
			visible := countVisibleRows(t, ctx, appConn, fix.TenantID, table)
			if visible == 0 {
				t.Errorf("tenant sees 0 rows in its own %s; want at least 1", table)
			}
		})
	}
}
