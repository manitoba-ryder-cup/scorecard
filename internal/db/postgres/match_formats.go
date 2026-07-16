package postgres

import (
	"context"
	"fmt"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type MatchFormatsDB struct {
	db *DB
}

func NewMatchFormatsDB(db *DB) *MatchFormatsDB {
	return &MatchFormatsDB{db: db}
}

// ListMatchFormats reads the global, seeded formats. match_formats is code-defined
// reference data shared across tenants (no tenant_id, no RLS), so it queries the pool
// directly rather than through WithTenantContext.
func (m *MatchFormatsDB) ListMatchFormats(ctx context.Context) ([]golf.MatchFormat, error) {
	formats, err := m.db.Queries().ListMatchFormats(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing match formats: %w", err)
	}
	result := make([]golf.MatchFormat, len(formats))
	for i, f := range formats {
		result[i] = golf.MatchFormat{ID: f.ID, Name: f.Name}
	}
	return result, nil
}
