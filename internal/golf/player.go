package golf

import (
	"context"
	"fmt"
)

// PlayerService handles player-related business logic
type PlayerService struct {
	PlayerDB playerDB
	Logger   logger
}

// GetPlayer retrieves a player by ID
func (s *PlayerService) GetPlayer(ctx context.Context, playerID int32) (*Player, error) {
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
