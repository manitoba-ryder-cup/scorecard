package golf

import (
	"context"
	"fmt"
)

// TeamService handles team-related business logic
type TeamService struct {
	TeamDB       teamDB
	TeamMemberDB teamMemberDB
	MatchService *MatchService
	Logger       logger
}

// GetTeamPoints returns a team's Ryder-Cup points for a tournament (1 per win,
// 0.5 per tie). TODO(step 4): read from materialized match results instead of
// recomputing across every match/score.
func (s *TeamService) GetTeamPoints(ctx context.Context, teamColor string, tournamentID int32) (float64, error) {
	return 0, nil
}

// GetCaptain returns the captain of a team (teams.captain_id), or nil if unset.
func (s *TeamService) GetCaptain(ctx context.Context, teamID int32) (*PlayerSummary, error) {
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
func (s *TeamService) GetTeam(ctx context.Context, teamID int32) (*Team, error) {
	team, err := s.TeamDB.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}
	return team, nil
}

// ListTeamsByTournament retrieves a tournament's two teams
func (s *TeamService) ListTeamsByTournament(ctx context.Context, tournamentID int32) ([]Team, error) {
	teams, err := s.TeamDB.ListTeamsByTournament(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	return teams, nil
}
