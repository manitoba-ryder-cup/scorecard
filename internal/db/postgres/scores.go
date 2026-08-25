package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type ScoresDB struct {
	db *DB
}

func NewScoresDB(db *DB) *ScoresDB {
	return &ScoresDB{db: db}
}

// SaveScoresAndRecompute upserts a hole's scores and rewrites the match's materialized
// result — the single write path keeping match_results in sync. The scoring rule stays
// in the domain (guard/recompute); the repository supplies only the transaction and lock
// that make the whole hole atomic.
func (s *ScoresDB) SaveScoresAndRecompute(
	ctx context.Context,
	matchID uuid.UUID,
	scores []golf.Score,
	tournamentID uuid.UUID,
	guard func([]golf.Score) error,
	recompute func([]golf.Score) golf.StoredResult,
) (golf.StoredResult, error) {
	return withTenant(ctx, s.db, func(q *sqlc.Queries, tenantID uuid.UUID) (golf.StoredResult, error) {
		var zero golf.StoredResult
		list := func() ([]golf.Score, error) {
			rows, err := q.ListScoresByMatch(ctx, sqlc.ListScoresByMatchParams{
				MatchID:  matchID,
				TenantID: tenantID,
			})
			if err != nil {
				return nil, fmt.Errorf("listing scores for match %s: %w", matchID, err)
			}
			return mapSlice(rows, toDomainScore), nil
		}

		// Before the write, so the guard's decision cannot be raced by another writer.
		if _, err := q.LockMatch(ctx, sqlc.LockMatchParams{
			ID:       matchID,
			TenantID: tenantID,
		}); err != nil {
			return zero, fmt.Errorf("locking match %s: %w", matchID, mapReadErr(err, golf.ErrMatchNotFound))
		}

		before, err := list()
		if err != nil {
			return zero, err
		}
		if err := guard(before); err != nil {
			return zero, err
		}

		for _, score := range scores {
			if err := upsertScore(ctx, q, score, tenantID); err != nil {
				return zero, err
			}
		}

		after, err := list()
		if err != nil {
			return zero, err
		}

		result := recompute(after)
		if _, err := q.UpsertMatchResult(ctx, sqlc.UpsertMatchResultParams{
			MatchID:        matchID,
			TournamentID:   tournamentID,
			TenantID:       tenantID,
			Finished:       result.Finished,
			LeaderTeamID:   result.LeaderTeamID,
			Lead:           int32(result.Lead),
			HolesRemaining: int32(result.HolesRemaining),
		}); err != nil {
			return zero, fmt.Errorf("upserting match result %s: %w", matchID, mapWriteErr(err))
		}
		return result, nil
	})
}

func (s *ScoresDB) ResetMatch(ctx context.Context, matchID uuid.UUID) error {
	return withTenantExec(ctx, s.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
		if _, err := q.LockMatch(ctx, sqlc.LockMatchParams{
			ID:       matchID,
			TenantID: tenantID,
		}); err != nil {
			return fmt.Errorf("locking match %s: %w", matchID, mapReadErr(err, golf.ErrMatchNotFound))
		}
		if _, err := q.DeleteScoresByMatch(ctx, sqlc.DeleteScoresByMatchParams{
			MatchID:  matchID,
			TenantID: tenantID,
		}); err != nil {
			return fmt.Errorf("deleting scores for match %s: %w", matchID, mapWriteErr(err))
		}
		if _, err := q.DeleteMatchResult(ctx, sqlc.DeleteMatchResultParams{
			MatchID:  matchID,
			TenantID: tenantID,
		}); err != nil {
			return fmt.Errorf("deleting match result %s: %w", matchID, mapWriteErr(err))
		}
		return nil
	})
}

// upsertScore writes one hole score. PlayerID present -> per-player row (singles/
// fourball); nil -> one team row (alt shot/scramble). The two grains hit different
// partial unique indexes, so the write must pick the matching statement.
func upsertScore(ctx context.Context, q *sqlc.Queries, score golf.Score, tenantID uuid.UUID) error {
	if score.PlayerID != nil {
		_, err := q.UpsertPlayerScore(ctx, sqlc.UpsertPlayerScoreParams{
			MatchID:    score.MatchID,
			TeamID:     score.TeamID,
			PlayerID:   score.PlayerID,
			CourseID:   score.CourseID,
			TeeColorID: score.TeeColorID,
			HoleNumber: score.HoleNumber,
			TenantID:   tenantID,
			Strokes:    score.Strokes,
		})
		if err != nil {
			// A bad player_id (not a participant) trips the composite FK -> 400, not 500.
			return fmt.Errorf("upserting player score: %w", mapWriteErr(err))
		}
		return nil
	}
	_, err := q.UpsertTeamScore(ctx, sqlc.UpsertTeamScoreParams{
		MatchID:    score.MatchID,
		TeamID:     score.TeamID,
		CourseID:   score.CourseID,
		TeeColorID: score.TeeColorID,
		HoleNumber: score.HoleNumber,
		TenantID:   tenantID,
		Strokes:    score.Strokes,
	})
	if err != nil {
		return fmt.Errorf("upserting team score: %w", mapWriteErr(err))
	}
	return nil
}

func (s *ScoresDB) ListScoresByMatch(ctx context.Context, matchID uuid.UUID) ([]golf.Score, error) {
	return withTenant(ctx, s.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.Score, error) {
		scores, err := q.ListScoresByMatch(ctx, sqlc.ListScoresByMatchParams{
			MatchID:  matchID,
			TenantID: tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing scores for match %s: %w", matchID, err)
		}
		return mapSlice(scores, toDomainScore), nil
	})
}

func (s *ScoresDB) ListScoresByTournament(ctx context.Context, tournamentID uuid.UUID) ([]golf.Score, error) {
	return withTenant(ctx, s.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.Score, error) {
		scores, err := q.ListScoresByTournament(ctx, sqlc.ListScoresByTournamentParams{
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing scores for tournament %s: %w", tournamentID, err)
		}
		return mapSlice(scores, toDomainScore), nil
	})
}

func toDomainScore(s sqlc.Score) golf.Score {
	return golf.Score{
		ID:         s.ID,
		MatchID:    s.MatchID,
		TeamID:     s.TeamID,
		PlayerID:   s.PlayerID,
		CourseID:   s.CourseID,
		TeeColorID: s.TeeColorID,
		HoleNumber: s.HoleNumber,
		Strokes:    s.Strokes,
	}
}
