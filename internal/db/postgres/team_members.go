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

type TeamMembersDB struct {
	db *DB
}

func NewTeamMembersDB(db *DB) *TeamMembersDB {
	return &TeamMembersDB{db: db}
}

// CreateTeamMember drafts a player onto a team.
func (t *TeamMembersDB) CreateTeamMember(ctx context.Context, teamID, playerID, tournamentID uuid.UUID) (*golf.TeamMember, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.TeamMember
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		member, err := q.CreateTeamMember(ctx, sqlc.CreateTeamMemberParams{
			TeamID:       teamID,
			PlayerID:     playerID,
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return fmt.Errorf("creating team member: %w", mapWriteErr(err))
		}
		result = &golf.TeamMember{
			TeamID:       member.TeamID,
			PlayerID:     member.PlayerID,
			TournamentID: member.TournamentID,
		}
		return nil
	})
	return result, err
}

// GetTeamCaptain returns the captain of a team (teams.captain_id), or nil if unset.
func (t *TeamMembersDB) GetTeamCaptain(ctx context.Context, teamID uuid.UUID) (*golf.Player, error) {
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
