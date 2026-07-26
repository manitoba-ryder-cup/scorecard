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
}

// GetOutcome reports whether every match is final and which team won (nil when
// unfinished or tied).
func (s *TournamentService) GetOutcome(ctx context.Context, tournamentID uuid.UUID) (TournamentOutcome, error) {
	outcome, err := s.ResultDB.GetTournamentOutcome(ctx, tournamentID)
	if err != nil {
		return TournamentOutcome{}, fmt.Errorf("failed to get tournament outcome: %w", err)
	}
	return outcome, nil
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
