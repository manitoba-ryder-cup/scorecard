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
// DeleteTeamMember undrafts a player. ErrNotFound if they weren't on the team.
//
// The guard is told whether they have been scored, and runs behind a lock on every match
// they are in — undrafting cascades to those lineups and on to their scores, so without the
// lock a score landing mid-check would be orphaned. Ordered by match id, so two undrafts
// cannot take the same locks in opposite orders.
func (t *TeamMembersDB) DeleteTeamMember(
	ctx context.Context,
	teamID, playerID uuid.UUID,
	guard func(scored bool) error,
) error {
	return withTenantExec(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
		if _, err := q.LockPlayerMatchesForScoring(ctx, sqlc.LockPlayerMatchesForScoringParams{
			TeamID:   teamID,
			PlayerID: playerID,
			TenantID: tenantID,
		}); err != nil {
			return fmt.Errorf("locking matches for player %s: %w", playerID, err)
		}
		scored, err := q.PlayerHasScoredMatches(ctx, sqlc.PlayerHasScoredMatchesParams{
			TeamID:   teamID,
			PlayerID: playerID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("checking scored matches for player %s: %w", playerID, err)
		}
		if err := guard(scored); err != nil {
			return err
		}
		rows, err := q.DeleteTeamMember(ctx, sqlc.DeleteTeamMemberParams{
			TeamID:   teamID,
			PlayerID: playerID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("deleting team member: %w", mapWriteErr(err))
		}
		if rows == 0 {
			return fmt.Errorf("deleting team member: %w", golf.ErrNotFound)
		}
		return nil
	})
}
