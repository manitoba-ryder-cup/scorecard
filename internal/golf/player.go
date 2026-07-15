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

// GetHandicapStrokes calculates the number of strokes a player receives on a hole
// based on their overall handicap and the hole's difficulty rating.
//
// Algorithm:
//   - Base strokes: playerHdcp / 18 (integer division)
//   - Additional stroke: +1 if (playerHdcp % 18) >= holeHdcp
//
// Example: A 12.5 handicap player on a hole with hdcp=5
//   - Base: 12.5 / 18 = 0 (integer division)
//   - Remainder: 12 % 18 = 12
//   - Since 12 >= 5, add 1 stroke
//   - Total: 0 + 1 = 1 stroke
func GetHandicapStrokes(playerHdcp float32, holeHdcp int32) int32 {
	base := int32(playerHdcp / 18)
	remainder := int32(playerHdcp) % 18

	if remainder >= holeHdcp {
		return base + 1
	}
	return base
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
