package postgres

import (
	"context"
	"fmt"

	"github.com/travisbale/knowhere/identity"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
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
		result = &golf.Team{ID: team.ID, TenantID: team.TenantID, Name: team.Name}
		return nil
	})
	return result, err
}

func (t *TeamsDB) ListTeams(ctx context.Context) ([]golf.Team, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.Team
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		teams, err := q.ListTeams(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("listing teams: %w", err)
		}
		result = make([]golf.Team, len(teams))
		for i, team := range teams {
			result[i] = golf.Team{ID: team.ID, TenantID: team.TenantID, Name: team.Name}
		}
		return nil
	})
	return result, err
}
