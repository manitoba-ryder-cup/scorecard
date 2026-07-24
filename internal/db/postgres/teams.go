package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type TeamsDB struct {
	db *DB
}

func NewTeamsDB(db *DB) *TeamsDB {
	return &TeamsDB{db: db}
}

func (t *TeamsDB) GetTeam(ctx context.Context, id uuid.UUID) (*golf.Team, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Team, error) {
		team, err := q.GetTeam(ctx, sqlc.GetTeamParams{ID: id, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("getting team %s: %w", id, mapReadErr(err))
		}
		td := toDomainTeam(team)
		return &td, nil
	})
}

func (t *TeamsDB) ListTeamsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]golf.TeamWithCaptain, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.TeamWithCaptain, error) {
		rows, err := q.ListTeamsByTournament(ctx, sqlc.ListTeamsByTournamentParams{
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing teams for tournament %s: %w", tournamentID, err)
		}
		return mapSlice(rows, toTeamWithCaptain), nil
	})
}

// SetTeamCaptain assigns a team's captain. An unknown team yields no rows (ErrNotFound);
// an unknown player trips the captain_id FK (ErrInvalidInput).
func (t *TeamsDB) SetTeamCaptain(ctx context.Context, teamID, captainID uuid.UUID) (*golf.Team, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Team, error) {
		team, err := q.SetTeamCaptain(ctx, sqlc.SetTeamCaptainParams{
			ID:        teamID,
			TenantID:  tenantID,
			CaptainID: &captainID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("setting captain for team %s: %w", teamID, mapReadErr(err))
			}
			return nil, fmt.Errorf("setting captain for team %s: %w", teamID, mapWriteErr(err))
		}
		td := toDomainTeam(team)
		return &td, nil
	})
}

// ClearCaptainForPlayer removes the player as a team's captain, if they are it. A no-op
// otherwise. Used when a captain leaves the team (undrafted) so the role never goes stale.
func (t *TeamsDB) ClearCaptainForPlayer(ctx context.Context, teamID, playerID uuid.UUID) error {
	return withTenantExec(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
		if err := q.ClearTeamCaptainForPlayer(ctx, sqlc.ClearTeamCaptainForPlayerParams{
			ID:        teamID,
			CaptainID: &playerID,
			TenantID:  tenantID,
		}); err != nil {
			return fmt.Errorf("clearing captain for team %s: %w", teamID, err)
		}
		return nil
	})
}

func toDomainTeam(t sqlc.Team) golf.Team {
	return golf.Team{
		ID:           t.ID,
		TournamentID: t.TournamentID,
		Color:        t.Color,
		CaptainID:    t.CaptainID,
	}
}

func toTeamWithCaptain(row sqlc.ListTeamsByTournamentRow) golf.TeamWithCaptain {
	twc := golf.TeamWithCaptain{
		Team: golf.Team{
			ID:           row.ID,
			TournamentID: row.TournamentID,
			Color:        row.Color,
			CaptainID:    row.CaptainID,
		},
	}
	// The LEFT JOIN leaves the captain columns NULL when captain_id is unset.
	if row.CaptainID != nil {
		twc.Captain = &golf.PlayerSummary{
			ID:        *row.CaptainID,
			FirstName: derefString(row.CaptainFirstName),
			LastName:  derefString(row.CaptainLastName),
			Email:     row.CaptainEmail,
		}
	}
	return twc
}

// derefString returns the string a pointer holds, or "" if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
