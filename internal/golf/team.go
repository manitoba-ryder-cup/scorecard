package golf

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// TeamService handles team reads and captain assignment.
type TeamService struct {
	TeamDB teamDB
	Logger logger
}

// SetCaptain assigns a team's captain. The captain must be an existing player (enforced
// by the FK): an unknown team surfaces as ErrNotFound, an unknown player as
// ErrInvalidInput.
func (s *TeamService) SetCaptain(ctx context.Context, teamID, captainID uuid.UUID) (*Team, error) {
	team, err := s.TeamDB.SetTeamCaptain(ctx, teamID, captainID)
	if err != nil {
		return nil, fmt.Errorf("failed to set captain: %w", err)
	}
	return team, nil
}

// ListTeamsByTournament retrieves a tournament's two teams with their captains resolved.
func (s *TeamService) ListTeamsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]TeamWithCaptain, error) {
	teams, err := s.TeamDB.ListTeamsByTournament(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	return teams, nil
}
