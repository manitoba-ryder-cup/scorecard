package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
)

type TeamMembersDB struct {
	db *DB
}

func NewTeamMembersDB(db *DB) *TeamMembersDB {
	return &TeamMembersDB{db: db}
}

// GetTeamCaptain returns the captain of a team (teams.captain_id), or nil if unset.
func (t *TeamMembersDB) GetTeamCaptain(ctx context.Context, teamID int32) (*golf.Player, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Player
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		captain, err := q.GetTeamCaptain(ctx, sqlc.GetTeamCaptainParams{ID: teamID, TenantID: tenantID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil // no captain set
			}
			return fmt.Errorf("getting team captain: %w", err)
		}
		p := toDomainPlayer(captain)
		result = &p
		return nil
	})
	return result, err
}
