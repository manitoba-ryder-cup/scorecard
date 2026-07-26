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

// ResultsDB reads/writes match_results and the aggregates derived from it.
type ResultsDB struct {
	db *DB
}

func NewResultsDB(db *DB) *ResultsDB {
	return &ResultsDB{db: db}
}

func (r *ResultsDB) GetMatchResult(ctx context.Context, matchID uuid.UUID) (*golf.StoredResult, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.StoredResult, error) {
		row, err := q.GetMatchResult(ctx, sqlc.GetMatchResultParams{MatchID: matchID, TenantID: tenantID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil // no result yet
			}
			return nil, fmt.Errorf("getting match result %s: %w", matchID, err)
		}
		return &golf.StoredResult{
			Finished:       row.Finished,
			LeaderTeamID:   row.LeaderTeamID,
			Lead:           int(row.Lead),
			HolesRemaining: int(row.HolesRemaining),
		}, nil
	})
}

func (r *ResultsDB) ListTeamPoints(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]float64, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (map[uuid.UUID]float64, error) {
		rows, err := q.ListTeamPoints(ctx, sqlc.ListTeamPointsParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing team points: %w", err)
		}
		points := make(map[uuid.UUID]float64, len(rows))
		for _, row := range rows {
			points[row.TeamID] = row.Points
		}
		return points, nil
	})
}

// GetTournamentOutcome reports whether every match is final and which team won. An
// unknown tournament reads as unfinished rather than an error, matching what the
// status and winner endpoints have always returned for one.
func (r *ResultsDB) GetTournamentOutcome(ctx context.Context, tournamentID uuid.UUID) (golf.TournamentOutcome, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (golf.TournamentOutcome, error) {
		row, err := q.GetTournamentOutcome(ctx, sqlc.GetTournamentOutcomeParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return golf.TournamentOutcome{}, nil
			}
			return golf.TournamentOutcome{}, fmt.Errorf("getting tournament outcome: %w", err)
		}
		return golf.TournamentOutcome{Finished: row.Finished, WinnerTeamID: row.WinnerTeamID}, nil
	})
}

func (r *ResultsDB) ListTournamentPlayerRecords(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]golf.PlayerRecord, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (map[uuid.UUID]golf.PlayerRecord, error) {
		rows, err := q.ListTournamentPlayerRecords(ctx, sqlc.ListTournamentPlayerRecordsParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing tournament player records: %w", err)
		}
		records := make(map[uuid.UUID]golf.PlayerRecord, len(rows))
		for _, row := range rows {
			records[row.PlayerID] = golf.PlayerRecord{
				Wins:   int32(row.Wins),
				Losses: int32(row.Losses),
				Ties:   int32(row.Ties),
			}
		}
		return records, nil
	})
}

func (r *ResultsDB) ListTournamentPlayerCups(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]int, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (map[uuid.UUID]int, error) {
		rows, err := q.ListTournamentPlayerCups(ctx, sqlc.ListTournamentPlayerCupsParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing tournament player cups: %w", err)
		}
		cups := make(map[uuid.UUID]int, len(rows))
		for _, row := range rows {
			cups[row.PlayerID] = int(row.CupsWon)
		}
		return cups, nil
	})
}
