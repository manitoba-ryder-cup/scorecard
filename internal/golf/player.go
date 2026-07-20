package golf

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CreatePlayerInput is the intent to add a player to the roster. Email and UserID are
// optional: roster-only players have neither, while a player with a heimdall login
// carries UserID. photo_path is set later via the photo upload. Request-shape
// validation (names present, email well-formed) happens at the API boundary.
type CreatePlayerInput struct {
	FirstName string
	LastName  string
	Email     *string
	UserID    *uuid.UUID
}

// CreatePlayer persists a new player. A duplicate email or user_id surfaces as
// ErrConflict from the repository.
func (s *PlayerService) CreatePlayer(ctx context.Context, in CreatePlayerInput) (*Player, error) {
	player, err := s.PlayerDB.CreatePlayer(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to create player: %w", err)
	}
	return player, nil
}

// PlayerService handles player reads.
type PlayerService struct {
	PlayerDB playerDB
	ResultDB resultDB
}

// GetPlayer retrieves a player by ID
func (s *PlayerService) GetPlayer(ctx context.Context, playerID uuid.UUID) (*Player, error) {
	player, err := s.PlayerDB.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}
	return player, nil
}

// ListPlayers retrieves all players for the tenant
func (s *PlayerService) ListPlayers(ctx context.Context) ([]Player, error) {
	players, err := s.PlayerDB.ListPlayers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list players: %w", err)
	}
	return players, nil
}

// GetPlayerRecord returns a player's win/loss/tie record, derived from match_results.
func (s *PlayerService) GetPlayerRecord(ctx context.Context, playerID uuid.UUID) (PlayerRecord, error) {
	record, err := s.ResultDB.GetPlayerRecord(ctx, playerID)
	if err != nil {
		return PlayerRecord{}, fmt.Errorf("failed to get player record: %w", err)
	}
	return record, nil
}
