package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type TeeColorsDB struct {
	db *DB
}

func NewTeeColorsDB(db *DB) *TeeColorsDB {
	return &TeeColorsDB{db: db}
}

func (t *TeeColorsDB) CreateTeeColor(ctx context.Context, in golf.CreateTeeColorInput) (*golf.TeeColor, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.TeeColor, error) {
		teeColor, err := q.CreateTeeColor(ctx, sqlc.CreateTeeColorParams{TenantID: tenantID, Color: in.Color})
		if err != nil {
			return nil, fmt.Errorf("creating tee color: %w", mapWriteErr(err))
		}
		tc := toDomainTeeColor(teeColor)
		return &tc, nil
	})
}

func (t *TeeColorsDB) ListTeeColors(ctx context.Context) ([]golf.TeeColor, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.TeeColor, error) {
		teeColors, err := q.ListTeeColors(ctx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("listing tee colors: %w", err)
		}
		return mapSlice(teeColors, toDomainTeeColor), nil
	})
}

func toDomainTeeColor(tc sqlc.TeeColor) golf.TeeColor {
	return golf.TeeColor{ID: tc.ID, Color: tc.Color}
}
