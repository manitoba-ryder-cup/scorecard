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
	cups, err := s.cupsWon(ctx)
	if err != nil {
		return nil, err
	}
	player.CupsWon = cups[player.ID]
	return player, nil
}

// ListPlayers retrieves all players for the tenant, each with their all-time record and
// Cups won.
func (s *PlayerService) ListPlayers(ctx context.Context) ([]Player, error) {
	players, err := s.PlayerDB.ListPlayers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list players: %w", err)
	}
	cups, err := s.cupsWon(ctx)
	if err != nil {
		return nil, err
	}
	for i := range players {
		players[i].CupsWon = cups[players[i].ID]
	}
	return players, nil
}

// cupsWon counts every player's Cup wins. Which side won a Cup is a scoring rule, so it
// is settled here from the raw standings rather than in SQL.
func (s *PlayerService) cupsWon(ctx context.Context) (map[uuid.UUID]int, error) {
	data, err := s.ResultDB.ListCupData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list cup data: %w", err)
	}
	return ComputeCupsWon(data), nil
}

// ListPlayerTournaments returns the player's tournament history. The per-event W-L-T is
// counted in the query; the verdict for their side is a scoring rule, decided here.
func (s *PlayerService) ListPlayerTournaments(ctx context.Context, playerID uuid.UUID) ([]PlayerTournamentHistory, error) {
	history, err := s.PlayerDB.ListPlayerTournaments(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list player tournaments: %w", err)
	}
	standings, err := s.ResultDB.ListAllTournamentStandings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tournament standings: %w", err)
	}
	for i := range history {
		history[i].Result = TournamentResultFor(standings[history[i].TournamentID], history[i].TeamID)
	}
	return history, nil
}
