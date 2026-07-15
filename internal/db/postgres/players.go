package postgres

import (
	"context"
	"fmt"

	"github.com/travisbale/knowhere/identity"
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

// toDomainPlayer converts a sqlc Player to a domain Player
func toDomainPlayer(p sqlc.Player) golf.Player {
	return golf.Player{
		ID:        p.ID,
		TenantID:  p.TenantID,
		Email:     p.Email,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		Hdcp:      p.Hdcp,
		PhotoPath: p.PhotoPath,
		Biography: p.Biography,
		Tier:      p.Tier,
		Cups:      p.Cups,
		Wins:      p.Wins,
		Ties:      p.Ties,
		Losses:    p.Losses,
	}
}
