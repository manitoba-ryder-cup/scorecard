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

func (t *TeamMembersDB) ListTeamMembers(ctx context.Context, tournamentID int32, teamID int32) ([]golf.TeamMember, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.TeamMember
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		members, err := q.ListTeamMembersByTeam(ctx, sqlc.ListTeamMembersByTeamParams{
			TournamentID: tournamentID,
			TeamID:       teamID,
			TenantID:     tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing team members: %w", err)
		}
		result = make([]golf.TeamMember, len(members))
		for i, member := range members {
			result[i] = golf.TeamMember{
				TournamentID: member.TournamentID,
				PlayerID:     member.PlayerID,
				TeamID:       member.TeamID,
				TenantID:     member.TenantID,
				IsCaptain:    member.IsCaptain,
			}
		}
		return nil
	})
	return result, err
}

// GetTeamCaptain returns the captain for a team in a tournament
func (t *TeamMembersDB) GetTeamCaptain(ctx context.Context, tournamentID int32, teamID int32) (*golf.Player, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Player
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		captain, err := q.GetTeamCaptain(ctx, sqlc.GetTeamCaptainParams{
			TournamentID: tournamentID,
			TeamID:       teamID,
			TenantID:     tenantID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// No captain found
				return nil
			}
			return fmt.Errorf("getting team captain: %w", err)
		}

		result = &golf.Player{
			ID:        captain.PlayerID,
			TenantID:  captain.TenantID,
			FirstName: captain.FirstName,
			LastName:  captain.LastName,
			Email:     captain.Email,
		}
		return nil
	})
	return result, err
}
