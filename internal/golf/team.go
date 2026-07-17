package golf

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// TeamService handles team reads.
type TeamService struct {
	TeamDB       teamDB
	TeamMemberDB teamMemberDB
	Logger       logger
}

// GetCaptain returns the captain of a team (teams.captain_id), or nil if unset.
func (s *TeamService) GetCaptain(ctx context.Context, teamID uuid.UUID) (*PlayerSummary, error) {
	captain, err := s.TeamMemberDB.GetTeamCaptain(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get captain: %w", err)
	}
	if captain == nil {
		return nil, nil
	}
	return &PlayerSummary{
		ID:        captain.ID,
		FirstName: captain.FirstName,
		LastName:  captain.LastName,
		Email:     captain.Email,
	}, nil
}

// GetTeam retrieves a team by ID
func (s *TeamService) GetTeam(ctx context.Context, teamID uuid.UUID) (*Team, error) {
	team, err := s.TeamDB.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}
	return team, nil
}

// ListTeamsByTournament retrieves a tournament's two teams
func (s *TeamService) ListTeamsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Team, error) {
	teams, err := s.TeamDB.ListTeamsByTournament(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	return teams, nil
}
