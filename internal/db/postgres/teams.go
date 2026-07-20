package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
)

type TeamsDB struct {
	db *DB
}

func NewTeamsDB(db *DB) *TeamsDB {
	return &TeamsDB{db: db}
}

func (t *TeamsDB) GetTeam(ctx context.Context, id uuid.UUID) (*golf.Team, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Team
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		team, err := q.GetTeam(ctx, sqlc.GetTeamParams{ID: id, TenantID: tenantID})
		if err != nil {
			return fmt.Errorf("getting team %s: %w", id, mapReadErr(err))
		}
		team2 := toDomainTeam(team)
		result = &team2
		return nil
	})
	return result, err
}

func (t *TeamsDB) ListTeamsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]golf.TeamWithCaptain, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.TeamWithCaptain
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		rows, err := q.ListTeamsByTournament(ctx, sqlc.ListTeamsByTournamentParams{
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing teams for tournament %s: %w", tournamentID, err)
		}
		result = make([]golf.TeamWithCaptain, len(rows))
		for i, row := range rows {
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
			result[i] = twc
		}
		return nil
	})
	return result, err
}

// SetTeamCaptain assigns a team's captain. An unknown team yields no rows (ErrNotFound);
// an unknown player trips the captain_id FK (ErrInvalidInput).
func (t *TeamsDB) SetTeamCaptain(ctx context.Context, teamID, captainID uuid.UUID) (*golf.Team, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Team
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		team, err := q.SetTeamCaptain(ctx, sqlc.SetTeamCaptainParams{
			ID:        teamID,
			TenantID:  tenantID,
			CaptainID: &captainID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("setting captain for team %s: %w", teamID, mapReadErr(err))
			}
			return fmt.Errorf("setting captain for team %s: %w", teamID, mapWriteErr(err))
		}
		team2 := toDomainTeam(team)
		result = &team2
		return nil
	})
	return result, err
}

func toDomainTeam(t sqlc.Team) golf.Team {
	return golf.Team{
		ID:           t.ID,
		TournamentID: t.TournamentID,
		Color:        t.Color,
		CaptainID:    t.CaptainID,
	}
}

// derefString returns the string a pointer holds, or "" if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
