package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
)

// ResultsDB reads/writes match_results and the aggregates derived from it.
type ResultsDB struct {
	db *DB
}

func NewResultsDB(db *DB) *ResultsDB {
	return &ResultsDB{db: db}
}

func (r *ResultsDB) UpsertMatchResult(ctx context.Context, matchID, tournamentID uuid.UUID, res golf.StoredResult) error {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return err
	}
	return r.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
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
			return fmt.Errorf("upserting match result %s: %w", matchID, err)
		}
		return nil
	})
}

func (r *ResultsDB) GetMatchResult(ctx context.Context, matchID uuid.UUID) (*golf.StoredResult, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}
	var result *golf.StoredResult
	err = r.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		row, err := q.GetMatchResult(ctx, sqlc.GetMatchResultParams{MatchID: matchID, TenantID: tenantID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil // no result yet
			}
			return fmt.Errorf("getting match result %s: %w", matchID, err)
		}
		result = &golf.StoredResult{
			Finished:       row.Finished,
			LeaderTeamID:   row.LeaderTeamID,
			Lead:           int(row.Lead),
			HolesRemaining: int(row.HolesRemaining),
		}
		return nil
	})
	return result, err
}

func (r *ResultsDB) ListTeamPoints(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]float64, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}
	points := make(map[uuid.UUID]float64)
	err = r.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		rows, err := q.ListTeamPoints(ctx, sqlc.ListTeamPointsParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			return fmt.Errorf("listing team points: %w", err)
		}
		for _, row := range rows {
			points[row.TeamID] = row.Points
		}
		return nil
	})
	return points, err
}

func (r *ResultsDB) IsTournamentFinished(ctx context.Context, tournamentID uuid.UUID) (bool, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return false, err
	}
	var finished bool
	err = r.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		f, err := q.IsTournamentFinished(ctx, sqlc.IsTournamentFinishedParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			return fmt.Errorf("checking tournament finished: %w", err)
		}
		if f != nil {
			finished = *f
		}
		return nil
	})
	return finished, err
}

func (r *ResultsDB) GetPlayerRecord(ctx context.Context, playerID uuid.UUID) (golf.PlayerRecord, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return golf.PlayerRecord{}, err
	}
	var record golf.PlayerRecord
	err = r.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		row, err := q.GetPlayerRecord(ctx, sqlc.GetPlayerRecordParams{PlayerID: playerID, TenantID: tenantID})
		if err != nil {
			return fmt.Errorf("getting player record %d: %w", playerID, err)
		}
		record = golf.PlayerRecord{
			Wins:   int32(row.Wins),
			Losses: int32(row.Losses),
			Ties:   int32(row.Ties),
		}
		return nil
	})
	return record, err
}
