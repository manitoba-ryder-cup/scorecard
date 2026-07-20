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
		f, err := q.IsTournamentFinished(ctx, sqlc.IsTournamentFinishedParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			return false, fmt.Errorf("checking tournament finished: %w", err)
		}
		return f != nil && *f, nil
	})
}

func (r *ResultsDB) GetPlayerRecord(ctx context.Context, playerID uuid.UUID) (golf.PlayerRecord, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (golf.PlayerRecord, error) {
		row, err := q.GetPlayerRecord(ctx, sqlc.GetPlayerRecordParams{PlayerID: playerID, TenantID: tenantID})
		if err != nil {
			return golf.PlayerRecord{}, fmt.Errorf("getting player record %s: %w", playerID, err)
		}
		return golf.PlayerRecord{
			Wins:   int32(row.Wins),
			Losses: int32(row.Losses),
			Ties:   int32(row.Ties),
		}, nil
	})
}
