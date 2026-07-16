package postgres

import (
	"context"
	"fmt"

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

func (t *TeamsDB) GetTeam(ctx context.Context, id int32) (*golf.Team, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Team
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		team, err := q.GetTeam(ctx, sqlc.GetTeamParams{ID: id, TenantID: tenantID})
		if err != nil {
			return fmt.Errorf("getting team %d: %w", id, err)
		}
		team2 := toDomainTeam(team)
		result = &team2
		return nil
	})
	return result, err
}

func (t *TeamsDB) ListTeamsByTournament(ctx context.Context, tournamentID int32) ([]golf.Team, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.Team
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		teams, err := q.ListTeamsByTournament(ctx, sqlc.ListTeamsByTournamentParams{
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing teams for tournament %d: %w", tournamentID, err)
		}
		result = make([]golf.Team, len(teams))
		for i, team := range teams {
			result[i] = toDomainTeam(team)
		}
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
