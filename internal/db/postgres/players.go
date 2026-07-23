package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

// PlayersDB handles player database operations
type PlayersDB struct {
	db *DB
}

// NewPlayersDB creates a new PlayersDB
func NewPlayersDB(db *DB) *PlayersDB {
	return &PlayersDB{db: db}
}

// CreatePlayer inserts a new player. photo_path starts empty (set later by the photo
// upload); a duplicate email or user_id surfaces as golf.ErrConflict via mapWriteErr.
func (p *PlayersDB) CreatePlayer(ctx context.Context, in golf.CreatePlayerInput) (*golf.Player, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Player, error) {
		player, err := q.CreatePlayer(ctx, sqlc.CreatePlayerParams{
			TenantID:  tenantID,
			UserID:    in.UserID,
			Email:     in.Email,
			FirstName: in.FirstName,
			LastName:  in.LastName,
			PhotoPath: "",
		})
		if err != nil {
			return nil, fmt.Errorf("creating player: %w", mapWriteErr(err))
		}
		pl := toDomainPlayer(player)
		return &pl, nil
	})
}

// GetPlayer retrieves a player (with their all-time record and cups) by ID.
func (p *PlayersDB) GetPlayer(ctx context.Context, id uuid.UUID) (*golf.Player, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Player, error) {
		rows, err := q.PlayerRecords(ctx, sqlc.PlayerRecordsParams{TenantID: tenantID, ID: &id})
		if err != nil {
			return nil, fmt.Errorf("getting player %s: %w", id, err)
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("getting player %s: %w", id, golf.ErrNotFound)
		}
		pl := toDomainPlayerRecord(rows[0])
		return &pl, nil
	})
}

// ListPlayers retrieves all players for the tenant, each with their record and cups.
func (p *PlayersDB) ListPlayers(ctx context.Context) ([]golf.Player, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.Player, error) {
		rows, err := q.PlayerRecords(ctx, sqlc.PlayerRecordsParams{TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing players: %w", err)
		}
		return mapSlice(rows, toDomainPlayerRecord), nil
	})
}

func toDomainPlayerRecord(r sqlc.PlayerRecordsRow) golf.Player {
	return golf.Player{
		ID: r.ID, UserID: r.UserID, Email: r.Email,
		FirstName: r.FirstName, LastName: r.LastName, PhotoPath: r.PhotoPath,
		Record:  golf.PlayerRecord{Wins: int32(r.Wins), Losses: int32(r.Losses), Ties: int32(r.Ties)},
		CupsWon: int(r.CupsWon),
	}
}

// ListPlayerTournaments returns the player's tournament history. Result is left unset
// here; the service derives it from each tournament's standings.
func (p *PlayersDB) ListPlayerTournaments(ctx context.Context, playerID uuid.UUID) ([]golf.PlayerTournamentHistory, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.PlayerTournamentHistory, error) {
		rows, err := q.ListPlayerTournaments(ctx, sqlc.ListPlayerTournamentsParams{PlayerID: playerID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing player tournaments %s: %w", playerID, err)
		}
		return mapSlice(rows, toPlayerTournamentHistory), nil
	})
}

func toPlayerTournamentHistory(row sqlc.ListPlayerTournamentsRow) golf.PlayerTournamentHistory {
	return golf.PlayerTournamentHistory{
		TournamentID:     row.TournamentID,
		Name:             row.Name,
		Location:         row.Location,
		StartDate:        row.StartDate,
		EndDate:          row.EndDate,
		CaptainFirstName: derefString(row.CaptainFirstName),
		CaptainLastName:  derefString(row.CaptainLastName),
		TeamID:           row.TeamID,
		Record: golf.PlayerRecord{
			Wins:   int32(row.Wins),
			Losses: int32(row.Losses),
			Ties:   int32(row.Ties),
		},
	}
}

// toDomainPlayer converts a sqlc Player to a domain Player. sqlc maps the nullable
// uuid column straight to *uuid.UUID, so user_id passes through with no conversion.
func toDomainPlayer(p sqlc.Player) golf.Player {
	return golf.Player{
		ID:        p.ID,
		UserID:    p.UserID,
		Email:     p.Email,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		PhotoPath: p.PhotoPath,
	}
}
