package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

// SeedDB writes a tournament's advance setup.
type SeedDB struct {
	db *DB
}

func NewSeedDB(db *DB) *SeedDB {
	return &SeedDB{db: db}
}

// SeedTournament writes the whole planned setup in one transaction, so roughly eighty
// statements commit together or not at all.
//
// Spread across separate calls, a failure partway would leave a tournament with half a
// roster and no matches, and rerunning would create a second one — nothing identifies the
// event, so only the players would match anything existing.
func (s *SeedDB) SeedTournament(ctx context.Context, plan golf.SeedPlan) (*golf.SeedSummary, error) {
	return withTenant(ctx, s.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.SeedSummary, error) {
		tournament, err := createTournamentWithTeams(ctx, q, tenantID, plan.Tournament, golf.TournamentTeamColors)
		if err != nil {
			return nil, err
		}

		teamsByColor, err := seedTeamsByColor(ctx, q, tenantID, tournament.ID)
		if err != nil {
			return nil, err
		}

		// Keyed by email, so someone who plays every year is one player, not one a year.
		existing, err := seedPlayersByEmail(ctx, q, tenantID)
		if err != nil {
			return nil, err
		}

		summary := &golf.SeedSummary{TournamentID: tournament.ID}
		entered := make(map[string]uuid.UUID, len(plan.Players))

		for _, sp := range plan.Players {
			email := strings.ToLower(strings.TrimSpace(sp.Email))
			playerID, ok := existing[email]
			if !ok {
				addr := sp.Email
				created, err := q.CreatePlayer(ctx, sqlc.CreatePlayerParams{
					TenantID: tenantID, Email: &addr, FirstName: sp.FirstName, LastName: sp.LastName,
				})
				if err != nil {
					return nil, fmt.Errorf("creating player %s: %w", sp.Email, mapWriteErr(err))
				}
				playerID = created.ID
				existing[email] = playerID
			}

			if _, err := q.CreateTournamentPlayer(ctx, sqlc.CreateTournamentPlayerParams{
				TournamentID: tournament.ID, PlayerID: playerID, TenantID: tenantID,
				Tier: sp.Tier, Biography: sp.Biography, Hdcp: sp.Hdcp,
			}); err != nil {
				return nil, fmt.Errorf("entering %s: %w", sp.Email, mapWriteErr(err))
			}
			entered[email] = playerID
			summary.PlayersEntered++
		}

		// Only captains: the field is picked live, but a side needs a captain to pick.
		for color, email := range plan.Captains {
			teamID, ok := teamsByColor[color]
			if !ok {
				return nil, fmt.Errorf("tournament has no %q team", color)
			}
			captainID, ok := entered[email]
			if !ok {
				return nil, fmt.Errorf("%s captain %q is not in the roster", color, email)
			}
			if _, err := q.CreateTeamMember(ctx, sqlc.CreateTeamMemberParams{
				TeamID: teamID, PlayerID: captainID, TournamentID: tournament.ID, TenantID: tenantID,
			}); err != nil {
				return nil, fmt.Errorf("drafting %s captain: %w", color, mapWriteErr(err))
			}
			if _, err := q.SetTeamCaptain(ctx, sqlc.SetTeamCaptainParams{
				ID: teamID, TenantID: tenantID, CaptainID: &captainID,
			}); err != nil {
				return nil, fmt.Errorf("setting %s captain: %w", color, mapWriteErr(err))
			}
		}

		// Matches in schedule order, without participants — those are assigned live.
		for _, m := range plan.Matches {
			if _, err := q.CreateMatch(ctx, sqlc.CreateMatchParams{
				TournamentID: tournament.ID, CourseID: plan.CourseID, TeeColorID: plan.TeeColorID,
				MatchFormatID: m.FormatID, TenantID: tenantID, TeeTime: m.TeeTime,
			}); err != nil {
				return nil, fmt.Errorf("creating %s match: %w", m.Format, mapWriteErr(err))
			}
			summary.Matches++
		}

		return summary, nil
	})
}

// seedTeamsByColor maps the colours of the teams just created to their ids, so a captain
// lands on the right side.
func seedTeamsByColor(ctx context.Context, q *sqlc.Queries, tenantID, tournamentID uuid.UUID) (map[string]uuid.UUID, error) {
	rows, err := q.ListTeamsByTournament(ctx, sqlc.ListTeamsByTournamentParams{
		TournamentID: tournamentID, TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing teams: %w", err)
	}
	out := make(map[string]uuid.UUID, len(rows))
	for _, r := range rows {
		out[r.Color] = r.ID
	}
	return out, nil
}

// seedPlayersByEmail indexes existing players by lowercased email. Email is the only
// stable identity a setup file can carry — names are typed by hand and change.
func seedPlayersByEmail(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID) (map[string]uuid.UUID, error) {
	rows, err := q.PlayerRecords(ctx, sqlc.PlayerRecordsParams{TenantID: tenantID})
	if err != nil {
		return nil, fmt.Errorf("listing players: %w", err)
	}
	out := make(map[string]uuid.UUID, len(rows))
	for _, r := range rows {
		if r.Email != nil {
			out[strings.ToLower(strings.TrimSpace(*r.Email))] = r.ID
		}
	}
	return out, nil
}
