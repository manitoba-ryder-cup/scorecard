package postgres

import (
	"context"
	"fmt"

	"github.com/travisbale/knowhere/identity"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
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
				MatchID:    score.MatchID,
				PlayerID:   score.PlayerID,
				CourseID:   score.CourseID,
				TeeColorID: score.TeeColorID,
				HoleNumber: score.HoleNumber,
				TenantID:   score.TenantID,
				Strokes:    score.Strokes,
			}
		}
		return nil
	})
	return result, err
}

func (s *ScoresDB) ListScoresByMatchAndPlayer(ctx context.Context, matchID int32, playerID int32) ([]golf.Score, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.Score
	err = s.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		scores, err := q.ListScoresByMatchAndPlayer(ctx, sqlc.ListScoresByMatchAndPlayerParams{
			MatchID:  matchID,
			PlayerID: playerID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing scores for match %d player %d: %w", matchID, playerID, err)
		}
		result = make([]golf.Score, len(scores))
		for i, score := range scores {
			result[i] = golf.Score{
				MatchID:    score.MatchID,
				PlayerID:   score.PlayerID,
				CourseID:   score.CourseID,
				TeeColorID: score.TeeColorID,
				HoleNumber: score.HoleNumber,
				TenantID:   score.TenantID,
				Strokes:    score.Strokes,
			}
		}
		return nil
	})
	return result, err
}

func (s *ScoresDB) GetHoleHandicap(ctx context.Context, courseID int32, teeColorID int32, holeNumber int32) (int32, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return 0, err
	}

	var result int32
	err = s.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		hole, err := q.GetHole(ctx, sqlc.GetHoleParams{
			CourseID:   courseID,
			TeeColorID: teeColorID,
			Number:     holeNumber,
			TenantID:   tenantID,
		})
		if err != nil {
			return fmt.Errorf("getting hole handicap: %w", err)
		}
		result = hole.Hdcp
		return nil
	})
	return result, err
}
