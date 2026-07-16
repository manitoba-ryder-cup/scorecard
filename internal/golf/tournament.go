package golf

import (
	"context"
	"fmt"
)

// TournamentService handles tournament reads and standings, derived from the
// materialized match_results.
type TournamentService struct {
	TournamentDB tournamentDB
	ResultDB     resultDB
	TeamService  *TeamService
	Logger       logger
}

// IsFinished reports whether all of a tournament's matches are complete.
func (s *TournamentService) IsFinished(ctx context.Context, tournamentID int32) (bool, error) {
	return s.ResultDB.IsTournamentFinished(ctx, tournamentID)
}

// GetWinningTeam returns the tournament's winning team ID, or nil if undecided
// (not finished) or tied.
func (s *TournamentService) GetWinningTeam(ctx context.Context, tournamentID int32) (*int32, error) {
	finished, err := s.IsFinished(ctx, tournamentID)
	if err != nil || !finished {
		return nil, err
	}
	points, err := s.ResultDB.ListTeamPoints(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list team points: %w", err)
	}
	return winnerFromPoints(points), nil
}

// GetTeamsData builds each team's summary (color, captain, points) for a tournament.
func (s *TournamentService) GetTeamsData(ctx context.Context, tournamentID int32) ([]TeamData, error) {
	teams, err := s.TeamService.ListTeamsByTournament(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	points, err := s.ResultDB.ListTeamPoints(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list team points: %w", err)
	}

	result := []TeamData{}
	for _, team := range teams {
		captain, err := s.TeamService.GetCaptain(ctx, team.ID)
		if err != nil {
			s.Logger.Error("failed to get captain for team", "team_id", team.ID, "error", err)
			captain = nil
		}
		result = append(result, TeamData{
			ID:      team.ID,
			Color:   team.Color,
			Captain: captain,
			Points:  points[team.ID],
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

// winnerFromPoints returns the team with the unique highest points, or nil on a tie.
func winnerFromPoints(points map[int32]float64) *int32 {
	var bestID int32
	var best float64
	count := 0
	first := true
	for id, p := range points {
		switch {
		case first || p > best:
			best, bestID, count, first = p, id, 1, false
		case p == best:
			count++
		}
	}
	if count != 1 {
		return nil
	}
	id := bestID
	return &id
}
