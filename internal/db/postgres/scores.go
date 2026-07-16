package postgres

import (
	"context"
	"fmt"

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

func (s *ScoresDB) ListScoresByMatch(ctx context.Context, matchID int32) ([]golf.Score, error) {
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
