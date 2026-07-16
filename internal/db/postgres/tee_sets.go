package postgres

import (
	"context"
	"fmt"

	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
)

type TeeSetsDB struct {
	db *DB
}

func NewTeeSetsDB(db *DB) *TeeSetsDB {
	return &TeeSetsDB{db: db}
}

// CreateTeeSet inserts a tee set and all its holes in a single transaction, so a
// course's tee is never left with a partial hole list. A bad tee_color_id surfaces as
// ErrInvalidInput (FK violation) via mapWriteErr.
func (t *TeeSetsDB) CreateTeeSet(ctx context.Context, in golf.CreateTeeSetInput) (*golf.TeeSetWithHoles, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.TeeSetWithHoles
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		teeSet, err := q.CreateTeeSet(ctx, sqlc.CreateTeeSetParams{
			CourseID:   in.CourseID,
			TeeColorID: in.TeeColorID,
			TenantID:   tenantID,
			Slope:      in.Slope,
			Rating:     in.Rating,
		})
		if err != nil {
			return fmt.Errorf("creating tee set: %w", mapWriteErr(err))
		}

		holes := make([]golf.Hole, len(in.Holes))
		for i, h := range in.Holes {
			hole, err := q.CreateHole(ctx, sqlc.CreateHoleParams{
				CourseID:   in.CourseID,
				TeeColorID: in.TeeColorID,
				Number:     h.Number,
				TenantID:   tenantID,
				Par:        h.Par,
				Hdcp:       h.Hdcp,
				Yards:      h.Yards,
			})
			if err != nil {
				return fmt.Errorf("creating hole %d: %w", h.Number, mapWriteErr(err))
			}
			holes[i] = toDomainHole(hole)
		}

		result = &golf.TeeSetWithHoles{TeeSet: toDomainTeeSet(teeSet), Holes: holes}
		return nil
	})
	return result, err
}

func toDomainTeeSet(ts sqlc.TeeSet) golf.TeeSet {
	return golf.TeeSet{
		CourseID:   ts.CourseID,
		TeeColorID: ts.TeeColorID,
		Slope:      ts.Slope,
		Rating:     ts.Rating,
	}
}

func toDomainHole(h sqlc.Hole) golf.Hole {
	return golf.Hole{
		CourseID:   h.CourseID,
		TeeColorID: h.TeeColorID,
		Number:     h.Number,
		Par:        h.Par,
		Hdcp:       h.Hdcp,
		Yards:      h.Yards,
	}
}
