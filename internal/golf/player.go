package golf

import (
	"context"
	"fmt"
	"strings"

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
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
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

// UpdatePlayerInput changes a player's own attributes. A nil field keeps its stored value.
type UpdatePlayerInput struct {
	ID        uuid.UUID
	FirstName *string
	LastName  *string
	Email     *string
	PhotoPath *string
}

// UpdatePlayer changes a player's attributes. ErrPlayerNotFound if there is no such player.
func (s *PlayerService) UpdatePlayer(ctx context.Context, in UpdatePlayerInput) (*Player, error) {
	player, err := s.PlayerDB.UpdatePlayer(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to update player: %w", err)
	}
	return player, nil
}

// GetPlayer retrieves a player by ID
func (s *PlayerService) GetPlayer(ctx context.Context, playerID uuid.UUID) (*Player, error) {
	player, err := s.PlayerDB.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}
	cups, err := cupsWon(ctx, s.ResultDB)
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
	cups, err := cupsWon(ctx, s.ResultDB)
	if err != nil {
		return nil, err
	}
	for i := range players {
		players[i].CupsWon = cups[players[i].ID]
	}
	return players, nil
}

// cupsWon counts every player's Cup wins. Which side won a Cup is a scoring rule, so it
// is settled from the raw standings here rather than in SQL. Shared by the player and
// roster services, which both report Cups.
func cupsWon(ctx context.Context, db resultDB) (map[uuid.UUID]int, error) {
	data, err := db.ListCupData(ctx)
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

// PointsFor is the cup's own currency: a win is a point, a half is half a point, and a
// loss is nothing. Defined once here because every stat that talks about points has to
// agree with the standings the leaderboard shows.
func PointsFor(r PlayerRecord) float64 {
	return float64(r.Wins) + float64(r.Ties)/2
}

// PlayerStats gathers a career the ways a captain and a player each want to read it. The
// three breakdowns are independent aggregates over the same matches, so they're fetched
// together rather than leaving a client to make one request per section.
//
// Points is reported with the number of cups it was earned over rather than as a rate:
// both numbers are worth seeing, and how to round the division is a display decision.
func (s *PlayerService) PlayerStats(ctx context.Context, playerID uuid.UUID) (*PlayerStats, error) {
	rows, err := s.PlayerDB.PlayerStatsRows(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to read player stats: %w", err)
	}
	// Summed from the format split, so a career total cannot disagree with the rows beneath.
	var points float64
	for _, f := range rows.ByFormat {
		points += PointsFor(f.Record)
	}
	return &PlayerStats{
		ByFormat:     rows.ByFormat,
		Teammates:    rows.Teammates,
		Opponents:    rows.Opponents,
		Points:       points,
		CupsPlayed:   len(rows.History),
		LastHole:     rows.LastHole,
		DecidedEarly: rows.DecidedEarly,
		BestWin:      rows.BestWin,
		HeaviestLoss: rows.HeaviestLoss,
	}, nil
}
