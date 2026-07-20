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

// GetPlayer retrieves a player by ID with tenant isolation
func (p *PlayersDB) GetPlayer(ctx context.Context, id uuid.UUID) (*golf.Player, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Player, error) {
		player, err := q.GetPlayer(ctx, sqlc.GetPlayerParams{ID: id, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("getting player %s: %w", id, mapReadErr(err))
		}
		pl := toDomainPlayer(player)
		return &pl, nil
	})
}

// ListPlayers retrieves all players for the tenant
func (p *PlayersDB) ListPlayers(ctx context.Context) ([]golf.Player, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.Player, error) {
		players, err := q.ListPlayers(ctx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("listing players: %w", err)
		}
		return mapSlice(players, toDomainPlayer), nil
	})
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
