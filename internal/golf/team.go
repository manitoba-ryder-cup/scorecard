package golf

import (
	"context"
	"fmt"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// CreateTeamInput is the validated intent to create one of a tournament's two sides.
// CaptainID is optional (a team can be created before its captain is chosen).
type CreateTeamInput struct {
	TournamentID int32
	Color        string
	CaptainID    *int32
}

// CreateTeam validates and persists a team. Color must be one of the two Ryder Cup
// sides; the database additionally caps a tournament at one team per color.
func (s *TeamService) CreateTeam(ctx context.Context, in CreateTeamInput) (*Team, error) {
	if !sdk.IsValidTeamColor(in.Color) {
		return nil, fmt.Errorf("%w: team color must be %q or %q", ErrInvalidInput, sdk.TeamColorRed, sdk.TeamColorBlue)
	}
	team, err := s.TeamDB.CreateTeam(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}
	return team, nil
}

// TeamService handles team reads.
type TeamService struct {
	TeamDB       teamDB
	TeamMemberDB teamMemberDB
	Logger       logger
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
