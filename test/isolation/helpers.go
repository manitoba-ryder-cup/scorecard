// Package isolation holds the cross-tenant isolation suite. Scorecard defends tenant
// boundaries twice: an explicit `tenant_id = $n` predicate on every query (covered by
// api_test.go, through the SDK) and RLS behind it (covered by rls_test.go, which drops
// the predicate). The split matters — an API-level test passes whether or not RLS works,
// because the predicate already filtered the rows.
package isolation

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
)

// tenantScopedTables carry a tenant_id and a tenant_isolation_policy. match_formats and
// schema_migrations are absent: both are global, tenant-free.
var tenantScopedTables = []string{
	"tee_colors",
	"courses",
	"tee_sets",
	"holes",
	"players",
	"tournaments",
	"teams",
	"tournament_players",
	"team_members",
	"matches",
	"match_participants",
	"scores",
	"match_results",
}

// connectSuperuser bypasses RLS, so fixtures can be staged under any tenant.
func connectSuperuser(t *testing.T, ctx context.Context) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(ctx, util.LoadConfig().DatabaseURL)
	if err != nil {
		t.Fatalf("connect as superuser: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

// connectAppRole connects as the role the server uses. RLS is only observable here —
// the superuser bypasses every policy.
func connectAppRole(t *testing.T, ctx context.Context) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(ctx, util.LoadConfig().AppDatabaseURL)
	if err != nil {
		t.Fatalf("connect as app role: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

// countVisibleRows counts rows visible with the tenant context set and no tenant
// predicate. The missing predicate is the point: it isolates what RLS alone enforces.
func countVisibleRows(t *testing.T, ctx context.Context, conn *pgx.Conn, tenantID uuid.UUID, table string) int {
	t.Helper()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// SET LOCAL mirrors how knowhere's WithTenantContext scopes a request.
	if _, err := tx.Exec(ctx, "SET LOCAL app.current_tenant_id = '"+tenantID.String()+"'"); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}

	var count int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

// seedFullFixture stages a tournament under a fresh tenant, touching every
// tenant-scoped table so the sweep has a row to find in each.
func seedFullFixture(t *testing.T, ctx context.Context, conn *pgx.Conn) *util.Fixture {
	t.Helper()

	fix, err := util.SeedSinglesMatch(ctx, conn)
	if err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	// SeedSinglesMatch stops short of scoring, so add one of each.
	if _, err := conn.Exec(ctx,
		`INSERT INTO scores (match_id, team_id, player_id, course_id, tee_color_id, hole_number, tenant_id, strokes)
		 VALUES ($1, $2, $3, $4, $5, 1, $6, 4)`,
		fix.MatchID, fix.TeamRed, fix.RedPlayer, fix.CourseID, fix.TeeColorID, fix.TenantID,
	); err != nil {
		t.Fatalf("seed score: %v", err)
	}
	if _, err := conn.Exec(ctx,
		`INSERT INTO match_results (match_id, tournament_id, tenant_id, finished, leader_team_id, lead, holes_remaining)
		 VALUES ($1, $2, $3, false, $4, 1, 17)`,
		fix.MatchID, fix.TournamentID, fix.TenantID, fix.TeamRed,
	); err != nil {
		t.Fatalf("seed match result: %v", err)
	}

	return fix
}
