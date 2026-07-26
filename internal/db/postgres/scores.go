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

// SaveScoreAndRecompute upserts one hole score and rewrites the match's materialized
// result — the single write path keeping match_results in sync. The scoring rule stays
// in the domain (recompute); the repository supplies only the transaction and lock that
// make the pair atomic.
func (s *ScoresDB) SaveScoreAndRecompute(
	ctx context.Context,
	score golf.Score,
	tournamentID uuid.UUID,
	recompute func([]golf.Score) golf.StoredResult,
) error {
	return withTenantExec(ctx, s.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
		// Before the write, so the recompute below always sees a committed, complete set.
		if _, err := q.LockMatchForScoring(ctx, sqlc.LockMatchForScoringParams{
			ID:       score.MatchID,
			TenantID: tenantID,
		}); err != nil {
			return fmt.Errorf("locking match %s: %w", score.MatchID, mapReadErr(err))
		}

		if err := upsertScore(ctx, q, score, tenantID); err != nil {
			return err
		}

		rows, err := q.ListScoresByMatch(ctx, sqlc.ListScoresByMatchParams{
			MatchID:  score.MatchID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing scores for match %s: %w", score.MatchID, err)
		}

		result := recompute(mapSlice(rows, toDomainScore))
		if _, err := q.UpsertMatchResult(ctx, sqlc.UpsertMatchResultParams{
			MatchID:        score.MatchID,
			TournamentID:   tournamentID,
			TenantID:       tenantID,
			Finished:       result.Finished,
			LeaderTeamID:   result.LeaderTeamID,
			Lead:           int32(result.Lead),
			HolesRemaining: int32(result.HolesRemaining),
		}); err != nil {
			return fmt.Errorf("upserting match result %s: %w", score.MatchID, mapWriteErr(err))
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
