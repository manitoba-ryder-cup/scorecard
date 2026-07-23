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

// SaveScore upserts one hole score. PlayerID present -> per-player row (singles/
// fourball); nil -> one team-attributable row (alt shot/scramble). The two paths hit
// different partial unique indexes, so the write must pick the matching statement.
func (s *ScoresDB) SaveScore(ctx context.Context, score golf.Score) error {
	return withTenantExec(ctx, s.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
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
	})
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
		return mapSlice(scores, toDomainScoreWithHole), nil
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

// ListScoresByMatch returns extra hole columns (par/hdcp/yards) the domain Score
// drops; toDomainScoreWithHole reads only the fields the scoring engine needs.
func toDomainScoreWithHole(s sqlc.ListScoresByMatchRow) golf.Score {
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
