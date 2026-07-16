package postgres

import (
	"context"
	"fmt"

	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
)

type TournamentPlayersDB struct {
	db *DB
}

func NewTournamentPlayersDB(db *DB) *TournamentPlayersDB {
	return &TournamentPlayersDB{db: db}
}

func (t *TournamentPlayersDB) CreateTournamentPlayer(ctx context.Context, in golf.EnterPlayerInput) (*golf.TournamentPlayer, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.TournamentPlayer
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		entry, err := q.CreateTournamentPlayer(ctx, sqlc.CreateTournamentPlayerParams{
			TournamentID: in.TournamentID,
			PlayerID:     in.PlayerID,
			TenantID:     tenantID,
			Tier:         in.Tier,
			Biography:    in.Biography,
			Hdcp:         in.Hdcp,
		})
		if err != nil {
			return fmt.Errorf("entering tournament player: %w", mapWriteErr(err))
		}
		tp := toDomainTournamentPlayer(entry)
		result = &tp
		return nil
	})
	return result, err
}

func (t *TournamentPlayersDB) UpdateTournamentPlayer(ctx context.Context, in golf.EnterPlayerInput) (*golf.TournamentPlayer, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.TournamentPlayer
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		entry, err := q.UpdateTournamentPlayer(ctx, sqlc.UpdateTournamentPlayerParams{
			TournamentID: in.TournamentID,
			PlayerID:     in.PlayerID,
			TenantID:     tenantID,
			Tier:         in.Tier,
			Biography:    in.Biography,
			Hdcp:         in.Hdcp,
		})
		if err != nil {
			// No row means the player was never entered -> ErrNotFound (404).
			return fmt.Errorf("updating tournament player: %w", mapReadErr(err))
		}
		tp := toDomainTournamentPlayer(entry)
		result = &tp
		return nil
	})
	return result, err
}

func (t *TournamentPlayersDB) ListTournamentPlayers(ctx context.Context, tournamentID int32) ([]golf.TournamentPlayerDetail, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.TournamentPlayerDetail
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		rows, err := q.ListTournamentPlayers(ctx, sqlc.ListTournamentPlayersParams{
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing tournament players: %w", err)
		}
		result = make([]golf.TournamentPlayerDetail, len(rows))
		for i, row := range rows {
			result[i] = golf.TournamentPlayerDetail{
				TournamentPlayer: golf.TournamentPlayer{
					TournamentID: row.TournamentID,
					PlayerID:     row.PlayerID,
					Tier:         row.Tier,
					Biography:    row.Biography,
					Hdcp:         row.Hdcp,
				},
				FirstName: row.FirstName,
				LastName:  row.LastName,
				Email:     row.Email,
				PhotoPath: row.PhotoPath,
			}
		}
		return nil
	})
	return result, err
}

func toDomainTournamentPlayer(tp sqlc.TournamentPlayer) golf.TournamentPlayer {
	return golf.TournamentPlayer{
		TournamentID: tp.TournamentID,
		PlayerID:     tp.PlayerID,
		Tier:         tp.Tier,
		Biography:    tp.Biography,
		Hdcp:         tp.Hdcp,
	}
}
