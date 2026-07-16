package postgres

import (
	"context"
	"fmt"

	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
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
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Player
	err = p.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		player, err := q.CreatePlayer(ctx, sqlc.CreatePlayerParams{
			TenantID:  tenantID,
			UserID:    in.UserID,
			Email:     in.Email,
			FirstName: in.FirstName,
			LastName:  in.LastName,
			PhotoPath: "",
		})
		if err != nil {
			return fmt.Errorf("creating player: %w", mapWriteErr(err))
		}
		pl := toDomainPlayer(player)
		result = &pl
		return nil
	})
	return result, err
}

// GetPlayer retrieves a player by ID with tenant isolation
func (p *PlayersDB) GetPlayer(ctx context.Context, id int32) (*golf.Player, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Player

	err = p.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		player, err := q.GetPlayer(ctx, sqlc.GetPlayerParams{
			ID:       id,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("getting player %d: %w", id, err)
		}

		p := toDomainPlayer(player)
		result = &p
		return nil
	})

	return result, err
}

// ListPlayers retrieves all players for the tenant
func (p *PlayersDB) ListPlayers(ctx context.Context) ([]golf.Player, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.Player

	err = p.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		players, err := q.ListPlayers(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("listing players: %w", err)
		}

		result = make([]golf.Player, len(players))
		for i, player := range players {
			result[i] = toDomainPlayer(player)
		}
		return nil
	})

	return result, err
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
