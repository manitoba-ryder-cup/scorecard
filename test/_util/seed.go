package util

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Fixture holds the IDs of a seeded singles match: two teams (Red/Blue), one player
// per side, on an 18-hole course. Enough to exercise the score-entry loop end to end.
type Fixture struct {
	TenantID   uuid.UUID
	CourseID   int32
	TeeColorID int32
	MatchID    int32
	TeamRed    int32
	TeamBlue   int32
	RedPlayer  int32
	BluePlayer int32
}

// Connect opens a single pgx connection for seeding. The caller closes it.
func Connect(ctx context.Context, databaseURL string) (*pgx.Conn, error) {
	return pgx.Connect(ctx, databaseURL)
}

// SeedPlayer inserts one player under the given tenant and returns its ID. Used to
// stage data for anonymous public-read tests (which read a specific tenant with no
// token).
func SeedPlayer(ctx context.Context, conn *pgx.Conn, tenantID uuid.UUID, first, last string) (int32, error) {
	var id int32
	err := conn.QueryRow(ctx,
		`INSERT INTO players (tenant_id, first_name, last_name) VALUES ($1, $2, $3) RETURNING id`,
		tenantID, first, last,
	).Scan(&id)
	return id, err
}

// SeedSinglesMatch inserts a complete fixture under a fresh random tenant and returns
// its IDs. It runs as the superuser, so RLS is bypassed and tenant_id is set
// explicitly on every row. Each call uses a new tenant, so fixtures never collide and
// no inter-test cleanup is needed. The course is a flat par-4 18 (stroke indexes 1-18).
func SeedSinglesMatch(ctx context.Context, conn *pgx.Conn) (*Fixture, error) {
	f := &Fixture{TenantID: uuid.New()}
	t := f.TenantID

	if err := conn.QueryRow(ctx,
		`INSERT INTO tee_colors (tenant_id, color) VALUES ($1, 'White') RETURNING id`, t,
	).Scan(&f.TeeColorID); err != nil {
		return nil, fmt.Errorf("seed tee_colors: %w", err)
	}
	if err := conn.QueryRow(ctx,
		`INSERT INTO courses (tenant_id, name) VALUES ($1, 'Test GC') RETURNING id`, t,
	).Scan(&f.CourseID); err != nil {
		return nil, fmt.Errorf("seed courses: %w", err)
	}
	if _, err := conn.Exec(ctx,
		`INSERT INTO tee_sets (course_id, tee_color_id, tenant_id, slope, rating) VALUES ($1, $2, $3, 113, 72.0)`,
		f.CourseID, f.TeeColorID, t,
	); err != nil {
		return nil, fmt.Errorf("seed tee_sets: %w", err)
	}
	// Flat par-4 course; hdcp (stroke index) must be unique 1-18 per tee set.
	for n := int32(1); n <= 18; n++ {
		if _, err := conn.Exec(ctx,
			`INSERT INTO holes (course_id, tee_color_id, number, tenant_id, par, hdcp, yards)
			 VALUES ($1, $2, $3, $4, 4, $3, 400)`,
			f.CourseID, f.TeeColorID, n, t,
		); err != nil {
			return nil, fmt.Errorf("seed hole %d: %w", n, err)
		}
	}

	if err := conn.QueryRow(ctx,
		`INSERT INTO players (tenant_id, first_name, last_name) VALUES ($1, 'Red', 'Player') RETURNING id`, t,
	).Scan(&f.RedPlayer); err != nil {
		return nil, fmt.Errorf("seed red player: %w", err)
	}
	if err := conn.QueryRow(ctx,
		`INSERT INTO players (tenant_id, first_name, last_name) VALUES ($1, 'Blue', 'Player') RETURNING id`, t,
	).Scan(&f.BluePlayer); err != nil {
		return nil, fmt.Errorf("seed blue player: %w", err)
	}

	var tournamentID int32
	if err := conn.QueryRow(ctx,
		`INSERT INTO tournaments (tenant_id, name, start_date, end_date, location)
		 VALUES ($1, 'Test Cup', '2026-07-01', '2026-07-03', 'Winnipeg') RETURNING id`, t,
	).Scan(&tournamentID); err != nil {
		return nil, fmt.Errorf("seed tournament: %w", err)
	}

	if err := conn.QueryRow(ctx,
		`INSERT INTO teams (tenant_id, tournament_id, color, captain_id) VALUES ($1, $2, 'Red', $3) RETURNING id`,
		t, tournamentID, f.RedPlayer,
	).Scan(&f.TeamRed); err != nil {
		return nil, fmt.Errorf("seed red team: %w", err)
	}
	if err := conn.QueryRow(ctx,
		`INSERT INTO teams (tenant_id, tournament_id, color, captain_id) VALUES ($1, $2, 'Blue', $3) RETURNING id`,
		t, tournamentID, f.BluePlayer,
	).Scan(&f.TeamBlue); err != nil {
		return nil, fmt.Errorf("seed blue team: %w", err)
	}

	// Players must be entered in the tournament (tournament_players) before they can be
	// drafted onto a team (team_members FK requires it).
	for _, playerID := range []int32{f.RedPlayer, f.BluePlayer} {
		if _, err := conn.Exec(ctx,
			`INSERT INTO tournament_players (tournament_id, player_id, tenant_id) VALUES ($1, $2, $3)`,
			tournamentID, playerID, t,
		); err != nil {
			return nil, fmt.Errorf("seed tournament_player %d: %w", playerID, err)
		}
	}

	if _, err := conn.Exec(ctx,
		`INSERT INTO team_members (team_id, player_id, tournament_id, tenant_id) VALUES ($1, $2, $3, $4)`,
		f.TeamRed, f.RedPlayer, tournamentID, t,
	); err != nil {
		return nil, fmt.Errorf("seed red member: %w", err)
	}
	if _, err := conn.Exec(ctx,
		`INSERT INTO team_members (team_id, player_id, tournament_id, tenant_id) VALUES ($1, $2, $3, $4)`,
		f.TeamBlue, f.BluePlayer, tournamentID, t,
	); err != nil {
		return nil, fmt.Errorf("seed blue member: %w", err)
	}

	// Match formats are global seeded reference data, not tenant-scoped — reference the
	// pre-seeded 'Singles' format rather than inserting one.
	var formatID int32
	if err := conn.QueryRow(ctx,
		`SELECT id FROM match_formats WHERE name = 'Singles'`,
	).Scan(&formatID); err != nil {
		return nil, fmt.Errorf("look up Singles match_format: %w", err)
	}

	if err := conn.QueryRow(ctx,
		`INSERT INTO matches (tournament_id, course_id, tee_color_id, match_format_id, tenant_id)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		tournamentID, f.CourseID, f.TeeColorID, formatID, t,
	).Scan(&f.MatchID); err != nil {
		return nil, fmt.Errorf("seed match: %w", err)
	}

	if _, err := conn.Exec(ctx,
		`INSERT INTO match_participants (tournament_id, match_id, player_id, team_id, tenant_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		tournamentID, f.MatchID, f.RedPlayer, f.TeamRed, t,
	); err != nil {
		return nil, fmt.Errorf("seed red participant: %w", err)
	}
	if _, err := conn.Exec(ctx,
		`INSERT INTO match_participants (tournament_id, match_id, player_id, team_id, tenant_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		tournamentID, f.MatchID, f.BluePlayer, f.TeamBlue, t,
	); err != nil {
		return nil, fmt.Errorf("seed blue participant: %w", err)
	}

	return f, nil
}
