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

// IsFinished reports whether all of a tournament's matches are complete.
// TODO(step 4): derive from materialized match results.
func (s *TournamentService) IsFinished(ctx context.Context, tournamentID int32) (bool, error) {
	return false, nil
}

// GetWinningTeam returns the tournament's winning team ID, or nil if undecided.
// TODO(step 4): compare materialized team points.
func (s *TournamentService) GetWinningTeam(ctx context.Context, tournamentID int32) (*int32, error) {
	return nil, nil
}

// GetTeamsData builds each team's summary (color, captain, points) for a tournament.
func (s *TournamentService) GetTeamsData(ctx context.Context, tournamentID int32) ([]TeamData, error) {
	teams, err := s.TeamService.ListTeamsByTournament(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	result := []TeamData{}
	for _, team := range teams {
		captain, err := s.TeamService.GetCaptain(ctx, team.ID)
		if err != nil {
			s.Logger.Error("failed to get captain for team", "team_id", team.ID, "error", err)
			captain = nil
		}

		points, err := s.TeamService.GetTeamPoints(ctx, team.Color, tournamentID)
		if err != nil {
			s.Logger.Error("failed to get points for team", "team_color", team.Color, "error", err)
			points = 0
		}

		result = append(result, TeamData{
			ID:      team.ID,
			Color:   team.Color,
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
