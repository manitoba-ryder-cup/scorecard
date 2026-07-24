package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type TeamMembersDB struct {
	db *DB
}

func NewTeamMembersDB(db *DB) *TeamMembersDB {
	return &TeamMembersDB{db: db}
}

// CreateTeamMember drafts a player onto a team.
func (t *TeamMembersDB) CreateTeamMember(ctx context.Context, teamID, playerID, tournamentID uuid.UUID) (*golf.TeamMember, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.TeamMember, error) {
		member, err := q.CreateTeamMember(ctx, sqlc.CreateTeamMemberParams{
			TeamID:       teamID,
			PlayerID:     playerID,
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("creating team member: %w", mapWriteErr(err))
		}
		return &golf.TeamMember{
			TeamID:       member.TeamID,
			PlayerID:     member.PlayerID,
			TournamentID: member.TournamentID,
		}, nil
	})
}

// DeleteTeamMember undrafts a player from a team. ErrNotFound if they weren't a member.
func (t *TeamMembersDB) DeleteTeamMember(ctx context.Context, teamID, playerID uuid.UUID) error {
	rows, err := withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) (int64, error) {
		return q.DeleteTeamMember(ctx, sqlc.DeleteTeamMemberParams{
			TeamID:   teamID,
			PlayerID: playerID,
			TenantID: tenantID,
		})
	})
	if err != nil {
		return fmt.Errorf("deleting team member: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("deleting team member: %w", golf.ErrNotFound)
	}
	return nil
}
