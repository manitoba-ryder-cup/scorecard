package golf

import (
	"context"
	"fmt"
)

// TournamentService handles tournament-related business logic
type TournamentService struct {
	TournamentDB tournamentDB
	MatchService *MatchService
	TeamService  *TeamService
	Logger       logger
}

// IsFinished checks if all matches in a tournament are complete.
//
// Returns true only if all matches are finished, false otherwise.
func (s *TournamentService) IsFinished(ctx context.Context, tournamentID int32) (bool, error) {
	matches, err := s.MatchService.MatchDB.ListMatchesByTournament(ctx, tournamentID)
	if err != nil {
		return false, fmt.Errorf("failed to list matches: %w", err)
	}

	if len(matches) == 0 {
		return false, nil
	}

	// Check if all matches are finished
	for _, match := range matches {
		finished, err := s.MatchService.IsFinished(ctx, match.ID)
		if err != nil {
			return false, fmt.Errorf("failed to check if match %d finished: %w", match.ID, err)
		}
		if !finished {
			return false, nil
		}
	}

	return true, nil
}

// GetWinningTeam determines the tournament winner by comparing team points.
// Requires the tournament to be finished.
func (s *TournamentService) GetWinningTeam(ctx context.Context, tournamentID int32) (string, error) {
	finished, err := s.IsFinished(ctx, tournamentID)
	if err != nil {
		return "", fmt.Errorf("failed to check if tournament finished: %w", err)
	}
	if !finished {
		return "", ErrMatchNotFinished
	}

	redPoints, err := s.TeamService.GetTeamPoints(ctx, TeamRed, tournamentID)
	if err != nil {
		return "", fmt.Errorf("failed to get Red team points: %w", err)
	}

	bluePoints, err := s.TeamService.GetTeamPoints(ctx, TeamBlue, tournamentID)
	if err != nil {
		return "", fmt.Errorf("failed to get Blue team points: %w", err)
	}

	if redPoints > bluePoints {
		return TeamRed, nil
	} else if bluePoints > redPoints {
		return TeamBlue, nil
	}
	return TeamTied, nil
}

// GetTeamsData builds team summary with captain and points for a tournament.
// Returns data for all teams participating in the tournament.
func (s *TournamentService) GetTeamsData(ctx context.Context, tournamentID int32) ([]TeamData, error) {
	teams, err := s.TeamService.ListTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	result := []TeamData{}
	for _, team := range teams {
		// Get captain
		captain, err := s.TeamService.GetCaptain(ctx, team.ID, tournamentID)
		if err != nil {
			s.Logger.Error("failed to get captain for team", "team_id", team.ID, "error", err)
			// Continue without captain
			captain = nil
		}

		// Get points
		points, err := s.TeamService.GetTeamPoints(ctx, team.Name, tournamentID)
		if err != nil {
			s.Logger.Error("failed to get points for team", "team_name", team.Name, "error", err)
			points = 0
		}

		result = append(result, TeamData{
			ID:      team.ID,
			Name:    team.Name,
			Captain: captain,
			Points:  points,
		})
	}

	return result, nil
}

// GetTournament retrieves a tournament by ID
func (s *TournamentService) GetTournament(ctx context.Context, tournamentID int32) (*Tournament, error) {
	tournament, err := s.TournamentDB.GetTournament(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament: %w", err)
	}
	return tournament, nil
}

// ListTournaments retrieves all tournaments for the tenant
func (s *TournamentService) ListTournaments(ctx context.Context) ([]Tournament, error) {
	tournaments, err := s.TournamentDB.ListTournaments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tournaments: %w", err)
	}
	return tournaments, nil
}
