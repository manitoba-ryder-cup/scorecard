package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// UpdatePlayer writes the supplied attributes and leaves the rest alone.
func (p *PlayersDB) UpdatePlayer(ctx context.Context, in golf.UpdatePlayerInput) (*golf.Player, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Player, error) {
		player, err := q.UpdatePlayer(ctx, sqlc.UpdatePlayerParams{
			ID:        in.ID,
			TenantID:  tenantID,
			FirstName: in.FirstName,
			LastName:  in.LastName,
			Email:     in.Email,
			PhotoPath: in.PhotoPath,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("updating player %s: %w", in.ID, golf.ErrPlayerNotFound)
			}
			// A duplicate address is the caller's doing: two players cannot share one.
			return nil, fmt.Errorf("updating player %s: %w", in.ID, mapWriteErr(err))
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
			return nil, fmt.Errorf("getting player %s: %w", id, golf.ErrPlayerNotFound)
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
		Record: golf.PlayerRecord{Wins: int32(r.Wins), Losses: int32(r.Losses), Ties: int32(r.Ties)},
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
		Tier:             row.Tier,
		Biography:        row.Biography,
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

// PlayerStatsRows runs every aggregate the stats page needs inside one tenant-scoped
// transaction. These used to be six separate calls, and because each withTenant opens its
// own transaction that meant six rounds of BEGIN / SET LOCAL / COMMIT — around two dozen
// round trips for queries that each execute in well under a millisecond. With the
// database in another region the round trips were nearly the whole response.
func (p *PlayersDB) PlayerStatsRows(ctx context.Context, playerID uuid.UUID) (*golf.PlayerStatsRows, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.PlayerStatsRows, error) {
		out := &golf.PlayerStatsRows{}

		byFormat, err := q.PlayerRecordByFormat(ctx, sqlc.PlayerRecordByFormatParams{PlayerID: playerID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing format records for player %s: %w", playerID, err)
		}
		out.ByFormat = mapSlice(byFormat, func(r sqlc.PlayerRecordByFormatRow) golf.FormatRecord {
			return golf.FormatRecord{
				FormatName: r.FormatName,
				Record:     golf.PlayerRecord{Wins: int32(r.Wins), Losses: int32(r.Losses), Ties: int32(r.Ties)},
			}
		})

		teammates, err := q.PlayerRecordByTeammate(ctx, sqlc.PlayerRecordByTeammateParams{PlayerID: playerID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing teammate records for player %s: %w", playerID, err)
		}
		out.Teammates = mapSlice(teammates, func(r sqlc.PlayerRecordByTeammateRow) golf.PairRecord {
			return golf.PairRecord{
				PlayerID: r.PlayerID, FirstName: r.FirstName, LastName: r.LastName, Matches: int(r.Matches),
				Record: golf.PlayerRecord{Wins: int32(r.Wins), Losses: int32(r.Losses), Ties: int32(r.Ties)},
			}
		})

		opponents, err := q.PlayerRecordByOpponent(ctx, sqlc.PlayerRecordByOpponentParams{PlayerID: playerID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing opponent records for player %s: %w", playerID, err)
		}
		out.Opponents = mapSlice(opponents, func(r sqlc.PlayerRecordByOpponentRow) golf.PairRecord {
			return golf.PairRecord{
				PlayerID: r.PlayerID, FirstName: r.FirstName, LastName: r.LastName, Matches: int(r.Matches),
				Record: golf.PlayerRecord{Wins: int32(r.Wins), Losses: int32(r.Losses), Ties: int32(r.Ties)},
			}
		})

		closeness, err := q.PlayerRecordByCloseness(ctx, sqlc.PlayerRecordByClosenessParams{PlayerID: playerID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("reading closeness split for player %s: %w", playerID, err)
		}
		out.LastHole = golf.PlayerRecord{Wins: int32(closeness.LastHoleWins), Losses: int32(closeness.LastHoleLosses), Ties: int32(closeness.LastHoleTies)}
		out.DecidedEarly = golf.PlayerRecord{Wins: int32(closeness.EarlyWins), Losses: int32(closeness.EarlyLosses), Ties: int32(closeness.EarlyTies)}

		extremes, err := q.PlayerMarginExtremes(ctx, sqlc.PlayerMarginExtremesParams{PlayerID: playerID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("reading margin extremes for player %s: %w", playerID, err)
		}
		for _, r := range extremes {
			m := &golf.NotableMatch{
				Year:           r.StartDate.Format("2006"),
				Lead:           r.Lead,
				HolesRemaining: r.HolesRemaining,
				Opponents:      r.Opponents,
			}
			if r.Kind == "win" {
				out.BestWin = m
			} else {
				out.HeaviestLoss = m
			}
		}

		history, err := q.ListPlayerTournaments(ctx, sqlc.ListPlayerTournamentsParams{PlayerID: playerID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing player tournaments %s: %w", playerID, err)
		}
		out.History = mapSlice(history, toPlayerTournamentHistory)

		return out, nil
	})
}
