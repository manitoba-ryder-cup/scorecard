package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
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
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return err
	}

	return s.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
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
				return fmt.Errorf("upserting player score: %w", err)
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
			return fmt.Errorf("upserting team score: %w", err)
		}
		return nil
	})
}

func (s *ScoresDB) ListScoresByMatch(ctx context.Context, matchID uuid.UUID) ([]golf.Score, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.Score
	err = s.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		scores, err := q.ListScoresByMatch(ctx, sqlc.ListScoresByMatchParams{
			MatchID:  matchID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing scores for match %d: %w", matchID, err)
		}
		result = make([]golf.Score, len(scores))
		for i, score := range scores {
			result[i] = golf.Score{
				ID:         score.ID,
				MatchID:    score.MatchID,
				TeamID:     score.TeamID,
				PlayerID:   score.PlayerID,
				CourseID:   score.CourseID,
				TeeColorID: score.TeeColorID,
				HoleNumber: score.HoleNumber,
				Strokes:    score.Strokes,
			}
		}
		return nil
	})
	return result, err
}
