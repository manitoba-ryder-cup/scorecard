package golf

import (
	"context"
	"fmt"
)

// RosterService manages a tournament's roster: entering players with their
// per-tournament attributes (tier, biography, handicap). The team draft
// (team_members) is a separate concern handled elsewhere.
type RosterService struct {
	TournamentPlayerDB tournamentPlayerDB
	Logger             logger
}

// EnterPlayerInput is the intent to enter a player in a tournament (or update their
// per-tournament attributes). Shape validation happens at the API boundary.
type EnterPlayerInput struct {
	TournamentID int32
	PlayerID     int32
	Tier         string
	Biography    string
	Hdcp         float32
}

// EnterPlayer enters a player in a tournament with their attributes. A duplicate
// entry surfaces as ErrConflict; an unknown player/tournament as ErrInvalidInput.
func (s *RosterService) EnterPlayer(ctx context.Context, in EnterPlayerInput) (*TournamentPlayer, error) {
	entry, err := s.TournamentPlayerDB.CreateTournamentPlayer(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to enter tournament player: %w", err)
	}
	return entry, nil
}

// UpdatePlayer updates an entered player's attributes. ErrNotFound if the player has
// not been entered in the tournament.
func (s *RosterService) UpdatePlayer(ctx context.Context, in EnterPlayerInput) (*TournamentPlayer, error) {
	entry, err := s.TournamentPlayerDB.UpdateTournamentPlayer(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to update tournament player: %w", err)
	}
	return entry, nil
}

// ListPlayers returns the tournament's entered players with their identity.
func (s *RosterService) ListPlayers(ctx context.Context, tournamentID int32) ([]TournamentPlayerDetail, error) {
	entries, err := s.TournamentPlayerDB.ListTournamentPlayers(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tournament players: %w", err)
	}
	return entries, nil
}
