package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type MatchFormatsDB struct {
	db *DB
}

func NewMatchFormatsDB(db *DB) *MatchFormatsDB {
	return &MatchFormatsDB{db: db}
}

// GetMatchFormat reads one of the global, seeded formats. No sentinel on the miss: the ids
// come off rows that reference them, so an absent format is a broken row rather than a caller
// asking for something that does not exist, and it must not answer as a 404.
func (m *MatchFormatsDB) GetMatchFormat(ctx context.Context, id uuid.UUID) (*golf.MatchFormat, error) {
	f, err := m.db.Queries().GetMatchFormat(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("reading match format %s: %w", id, err)
	}
	return &golf.MatchFormat{ID: f.ID, Name: f.Name, PlayersPerSide: f.PlayersPerSide, ScoresPerPlayer: f.ScoresPerPlayer}, nil
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
		result[i] = golf.MatchFormat{ID: f.ID, Name: f.Name, PlayersPerSide: f.PlayersPerSide, ScoresPerPlayer: f.ScoresPerPlayer}
	}
	return result, nil
}
