package golf

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// CreateTournamentInput is the intent to create a tournament — only the caller-supplied
// fields (no ID, no tenant). Request-shape validation (name present, dates valid and
// ordered) happens at the API boundary; the domain owns the two-team invariant below.
type CreateTournamentInput struct {
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Location  string
}

// tournamentTeamColors are the two sides every tournament is created with. A Ryder
// Cup has exactly two teams, no more no less, so they're seeded with the tournament
// rather than added by an admin — there is no valid state with zero or one team.
var tournamentTeamColors = []string{sdk.TeamColorRed, sdk.TeamColorBlue}

// CreateTournament persists a new tournament together with its two teams (Red and
// Blue) in a single atomic operation.
func (s *TournamentService) CreateTournament(ctx context.Context, in CreateTournamentInput) (*Tournament, error) {
	tournament, err := s.TournamentDB.CreateTournamentWithTeams(ctx, in, tournamentTeamColors)
	if err != nil {
		return nil, fmt.Errorf("failed to create tournament: %w", err)
	}
	return tournament, nil
}

// TournamentService handles tournament reads and standings, derived from the
// materialized match_results.
type TournamentService struct {
	TournamentDB tournamentDB
	ResultDB     resultDB
	TeamService  *TeamService
	Logger       logger
}

// IsFinished reports whether all of a tournament's matches are complete.
func (s *TournamentService) IsFinished(ctx context.Context, tournamentID uuid.UUID) (bool, error) {
	return s.ResultDB.IsTournamentFinished(ctx, tournamentID)
}

// GetWinningTeam returns the tournament's winning team ID, or nil if undecided
// (not finished) or tied.
func (s *TournamentService) GetWinningTeam(ctx context.Context, tournamentID uuid.UUID) (*uuid.UUID, error) {
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
func (s *TournamentService) GetTeamsData(ctx context.Context, tournamentID uuid.UUID) ([]TeamData, error) {
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
		result = append(result, TeamData{
			ID:      team.ID,
			Color:   team.Color,
			Captain: team.Captain,
			Points:  points[team.ID],
		})
	}
	return result, nil
}

// GetTournament retrieves a tournament by ID
func (s *TournamentService) GetTournament(ctx context.Context, tournamentID uuid.UUID) (*Tournament, error) {
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
func winnerFromPoints(points map[uuid.UUID]float64) *uuid.UUID {
	var bestID uuid.UUID
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
