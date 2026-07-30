package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type TournamentPlayersDB struct {
	db *DB
}

func NewTournamentPlayersDB(db *DB) *TournamentPlayersDB {
	return &TournamentPlayersDB{db: db}
}

func (t *TournamentPlayersDB) CreateTournamentPlayer(ctx context.Context, in golf.EnterPlayerInput) (*golf.TournamentPlayer, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.TournamentPlayer, error) {
		row, err := q.CreateTournamentPlayer(ctx, sqlc.CreateTournamentPlayerParams{
			TournamentID: in.TournamentID,
			PlayerID:     in.PlayerID,
			TenantID:     tenantID,
			Tier:         in.Tier,
			Biography:    in.Biography,
			Hdcp:         in.Hdcp,
		})
		if err != nil {
			return nil, fmt.Errorf("entering tournament player: %w", mapWriteErr(err))
		}
		tp := toTournamentPlayer(row.TournamentID, row.PlayerID, row.Tier, row.Biography, row.Hdcp,
			row.FirstName, row.LastName, row.PhotoPath, row.TeamID)
		return &tp, nil
	})
}

// UpdateTournamentPlayer writes the supplied attributes, then reads the entry back
// enriched. Two statements in one transaction rather than one CTE — see the query file for
// why — so the read cannot observe anything but this write.
func (t *TournamentPlayersDB) UpdateTournamentPlayer(ctx context.Context, in golf.UpdateRosterEntryInput) (*golf.TournamentPlayer, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.TournamentPlayer, error) {
		if _, err := q.UpdateTournamentPlayer(ctx, sqlc.UpdateTournamentPlayerParams{
			TournamentID: in.TournamentID,
			PlayerID:     in.PlayerID,
			TenantID:     tenantID,
			Tier:         in.Tier,
			Biography:    in.Biography,
			Hdcp:         in.Hdcp,
		}); err != nil {
			// No row means the player was never entered -> ErrNotFound (404).
			return nil, fmt.Errorf("updating tournament player: %w", mapReadErr(err))
		}
		row, err := q.GetTournamentPlayer(ctx, sqlc.GetTournamentPlayerParams{
			TournamentID: in.TournamentID,
			PlayerID:     in.PlayerID,
			TenantID:     tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("reading back tournament player: %w", mapReadErr(err))
		}
		tp := toTournamentPlayer(row.TournamentID, row.PlayerID, row.Tier, row.Biography, row.Hdcp,
			row.FirstName, row.LastName, row.PhotoPath, row.TeamID)
		return &tp, nil
	})
}

func (t *TournamentPlayersDB) ListTournamentPlayers(ctx context.Context, tournamentID uuid.UUID) ([]golf.TournamentPlayer, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.TournamentPlayer, error) {
		rows, err := q.ListTournamentPlayers(ctx, sqlc.ListTournamentPlayersParams{
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing tournament players: %w", err)
		}
		result := make([]golf.TournamentPlayer, len(rows))
		for i, row := range rows {
			// team_id is nullable here (LEFT JOIN): nil when undrafted.
			result[i] = toTournamentPlayer(row.TournamentID, row.PlayerID, row.Tier, row.Biography, row.Hdcp,
				row.FirstName, row.LastName, row.PhotoPath, row.TeamID)
		}
		return result, nil
	})
}

func (t *TournamentPlayersDB) ListTournamentPlayersByTeam(ctx context.Context, teamID uuid.UUID) ([]golf.TournamentPlayer, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.TournamentPlayer, error) {
		rows, err := q.ListTournamentPlayersByTeam(ctx, sqlc.ListTournamentPlayersByTeamParams{
			TeamID:   teamID,
			TenantID: tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing team members: %w", err)
		}
		result := make([]golf.TournamentPlayer, len(rows))
		for i, row := range rows {
			tid := row.TeamID // non-null here (INNER JOIN)
			result[i] = toTournamentPlayer(row.TournamentID, row.PlayerID, row.Tier, row.Biography, row.Hdcp,
				row.FirstName, row.LastName, row.PhotoPath, &tid)
		}
		return result, nil
	})
}

// toTournamentPlayer assembles the unified roster entry shared by every read/write.
func toTournamentPlayer(tournamentID, playerID uuid.UUID, tier, biography string, hdcp float32, first, last, photo string, teamID *uuid.UUID) golf.TournamentPlayer {
	return golf.TournamentPlayer{
		TournamentID: tournamentID,
		PlayerID:     playerID,
		Tier:         tier,
		Biography:    biography,
		Hdcp:         hdcp,
		FirstName:    first,
		LastName:     last,
		PhotoPath:    photo,
		TeamID:       teamID,
	}
}
