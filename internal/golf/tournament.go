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
var TournamentTeamColors = []string{sdk.TeamColorRed, sdk.TeamColorBlue}

// CreateTournament persists a new tournament together with its two teams (Red and
// Blue) in a single atomic operation.
func (s *TournamentService) CreateTournament(ctx context.Context, in CreateTournamentInput) (*Tournament, error) {
	tournament, err := s.TournamentDB.CreateTournamentWithTeams(ctx, in, TournamentTeamColors)
	if err != nil {
		return nil, fmt.Errorf("failed to create tournament: %w", err)
	}
	tournament.Phase = sdk.PhaseUpcoming // it has no matches yet, let alone scores
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
// unfinished or tied). The teams are only needed to settle a winner, so an in-progress
// Cup — the common case during an event — costs a single query.
func (s *TournamentService) GetOutcome(ctx context.Context, tournamentID uuid.UUID) (TournamentOutcome, error) {
	outcomes, err := s.ResultDB.ListMatchOutcomes(ctx, tournamentID)
	if err != nil {
		return TournamentOutcome{}, fmt.Errorf("failed to list match outcomes: %w", err)
	}
	if !IsTournamentComplete(outcomes) {
		return TournamentOutcome{}, nil
	}
	teams, err := s.TeamService.ListTeamsByTournament(ctx, tournamentID)
	if err != nil {
		return TournamentOutcome{}, fmt.Errorf("failed to list teams: %w", err)
	}
	return ComputeTournamentOutcome(outcomes, teamIDs(teams)), nil
}

func teamIDs(teams []TeamWithCaptain) []uuid.UUID {
	ids := make([]uuid.UUID, len(teams))
	for i, t := range teams {
		ids[i] = t.ID
	}
	return ids
}

// GetTeamsData builds each team's summary (color, captain, points) for a tournament, and
// reports where the cup stands. The phase is a by-product, not extra work: it comes from
// the same outcomes the points do.
func (s *TournamentService) GetTeamsData(ctx context.Context, tournamentID uuid.UUID) ([]TeamData, sdk.TournamentPhase, error) {
	teams, err := s.TeamService.ListTeamsByTournament(ctx, tournamentID)
	if err != nil {
		return nil, sdk.PhaseUpcoming, fmt.Errorf("failed to list teams: %w", err)
	}
	outcomes, err := s.ResultDB.ListMatchOutcomes(ctx, tournamentID)
	if err != nil {
		return nil, sdk.PhaseUpcoming, fmt.Errorf("failed to list match outcomes: %w", err)
	}
	ids := teamIDs(teams)
	points := ComputeTeamPoints(outcomes, ids)

	result := []TeamData{}
	for _, team := range teams {
		result = append(result, TeamData{
			ID:      team.ID,
			Color:   team.Color,
			Captain: team.Captain,
			Points:  points[team.ID],
		})
	}
	return result, ComputePhase(outcomes), nil
}

// GetTournament retrieves a tournament by ID, in the phase its matches put it in.
func (s *TournamentService) GetTournament(ctx context.Context, tournamentID uuid.UUID) (*Tournament, error) {
	tournament, err := s.TournamentDB.GetTournament(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament: %w", err)
	}
	outcomes, err := s.ResultDB.ListMatchOutcomes(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list match outcomes: %w", err)
	}
	tournament.Phase = ComputePhase(outcomes)
	return tournament, nil
}

// ListTournaments retrieves all tournaments for the tenant. The outcomes behind their
// phases are fetched in one query rather than one per cup.
func (s *TournamentService) ListTournaments(ctx context.Context) ([]Tournament, error) {
	tournaments, err := s.TournamentDB.ListTournaments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tournaments: %w", err)
	}
	outcomes, err := s.ResultDB.ListAllMatchOutcomes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list match outcomes: %w", err)
	}
	for i := range tournaments {
		tournaments[i].Phase = ComputePhase(outcomes[tournaments[i].ID])
	}
	return tournaments, nil
}
