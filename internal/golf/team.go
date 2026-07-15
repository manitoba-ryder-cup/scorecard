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

// GetTeamPoints calculates total points for a team in a tournament.
// Awards: 1.0 for win, 0.5 for tie, 0.0 for loss.
//
// Algorithm:
//   - Iterate through all matches in tournament
//   - Add 1 point for each win
//   - Add 0.5 points for each tie
//   - Add 0 points for each loss
func (s *TeamService) GetTeamPoints(ctx context.Context, teamName string, tournamentID int32) (float64, error) {
	// Get all matches for tournament
	matches, err := s.MatchService.MatchDB.ListMatchesByTournament(ctx, tournamentID)
	if err != nil {
		return 0, fmt.Errorf("failed to list matches: %w", err)
	}

	var points float64
	for _, match := range matches {
		// Calculate winner for this match
		winner, err := s.MatchService.GetWinner(ctx, match.ID)
		if err != nil {
			// Skip incomplete matches
			continue
		}

		if winner == teamName {
			points += 1.0
		} else if winner == TeamTied {
			points += 0.5
		}
		// No points for loss
	}

	return points, nil
}

// GetCaptain returns the team captain for a tournament
func (s *TeamService) GetCaptain(ctx context.Context, teamID int32, tournamentID int32) (*PlayerSummary, error) {
	captain, err := s.TeamMemberDB.GetTeamCaptain(ctx, tournamentID, teamID)
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

// ListTeams retrieves all teams for the tenant
func (s *TeamService) ListTeams(ctx context.Context) ([]Team, error) {
	teams, err := s.TeamDB.ListTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	return teams, nil
}
