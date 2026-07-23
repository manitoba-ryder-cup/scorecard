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

func (r *ResultsDB) UpsertMatchResult(ctx context.Context, matchID, tournamentID uuid.UUID, res golf.StoredResult) error {
	return withTenantExec(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
		_, err := q.UpsertMatchResult(ctx, sqlc.UpsertMatchResultParams{
			MatchID:        matchID,
			TournamentID:   tournamentID,
			TenantID:       tenantID,
			Finished:       res.Finished,
			LeaderTeamID:   res.LeaderTeamID,
			Lead:           int32(res.Lead),
			HolesRemaining: int32(res.HolesRemaining),
		})
		if err != nil {
			return fmt.Errorf("upserting match result %s: %w", matchID, mapWriteErr(err))
		}
		return nil
	})
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

func (r *ResultsDB) IsTournamentFinished(ctx context.Context, tournamentID uuid.UUID) (bool, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (bool, error) {
		finished, err := q.IsTournamentFinished(ctx, sqlc.IsTournamentFinishedParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			return false, fmt.Errorf("checking tournament finished: %w", err)
		}
		return finished, nil
	})
}

func (r *ResultsDB) GetTournamentWinner(ctx context.Context, tournamentID uuid.UUID) (*uuid.UUID, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*uuid.UUID, error) {
		teamID, err := q.GetTournamentWinner(ctx, sqlc.GetTournamentWinnerParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil // unfinished or tied — no winner
			}
			return nil, fmt.Errorf("getting tournament winner: %w", err)
		}
		return &teamID, nil
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
