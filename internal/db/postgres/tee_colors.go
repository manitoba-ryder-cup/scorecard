package postgres

import (
	"context"
	"fmt"

	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
)

type TeeColorsDB struct {
	db *DB
}

func NewTeeColorsDB(db *DB) *TeeColorsDB {
	return &TeeColorsDB{db: db}
}

func (t *TeeColorsDB) CreateTeeColor(ctx context.Context, in golf.CreateTeeColorInput) (*golf.TeeColor, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.TeeColor
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		teeColor, err := q.CreateTeeColor(ctx, sqlc.CreateTeeColorParams{TenantID: tenantID, Color: in.Color})
		if err != nil {
			return fmt.Errorf("creating tee color: %w", mapWriteErr(err))
		}
		tc := toDomainTeeColor(teeColor)
		result = &tc
		return nil
	})
	return result, err
}

func (t *TeeColorsDB) ListTeeColors(ctx context.Context) ([]golf.TeeColor, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.TeeColor
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		teeColors, err := q.ListTeeColors(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("listing tee colors: %w", err)
		}
		result = make([]golf.TeeColor, len(teeColors))
		for i, tc := range teeColors {
			result[i] = toDomainTeeColor(tc)
		}
		return nil
	})
	return result, err
}

func toDomainTeeColor(tc sqlc.TeeColor) golf.TeeColor {
	return golf.TeeColor{ID: tc.ID, Color: tc.Color}
}
