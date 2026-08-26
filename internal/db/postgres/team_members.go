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

// DeleteTeamMember undrafts a player from a team. ErrTeamMemberNotFound if they weren't a
// member. A lineup place references this row, so a player named to play a match is refused by
// the foreign key rather than by a read taken beforehand — nothing here has to be locked,
// because the rule is the reference itself.
func (t *TeamMembersDB) DeleteTeamMember(ctx context.Context, teamID, playerID uuid.UUID) error {
	return withTenantExec(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
		rows, err := q.DeleteTeamMember(ctx, sqlc.DeleteTeamMemberParams{
			TeamID:   teamID,
			PlayerID: playerID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("deleting team member: %w", mapDeleteErr(err, golf.ErrAssignedPlayerUndraft))
		}
		if rows == 0 {
			return fmt.Errorf("deleting team member: %w", golf.ErrTeamMemberNotFound)
		}
		return nil
	})
}
