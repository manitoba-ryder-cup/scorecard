package postgres

import (
	"context"
	"fmt"

	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
)

// MatchesDB handles match database operations
type MatchesDB struct {
	db *DB
}

// NewMatchesDB creates a new MatchesDB
func NewMatchesDB(db *DB) *MatchesDB {
	return &MatchesDB{db: db}
}

// GetMatch retrieves a match by ID with tenant isolation
func (m *MatchesDB) GetMatch(ctx context.Context, id int32) (*golf.Match, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Match
	err = m.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		match, err := q.GetMatch(ctx, sqlc.GetMatchParams{ID: id, TenantID: tenantID})
		if err != nil {
			return fmt.Errorf("getting match %d: %w", id, err)
		}
		dm := toDomainMatch(match)
		result = &dm
		return nil
	})

	return result, err
}

// ListMatchesByTournament retrieves all matches for a tournament
func (m *MatchesDB) ListMatchesByTournament(ctx context.Context, tournamentID int32) ([]golf.Match, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.Match
	err = m.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		matches, err := q.ListMatchesByTournament(ctx, sqlc.ListMatchesByTournamentParams{
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing matches for tournament %d: %w", tournamentID, err)
		}
		result = make([]golf.Match, len(matches))
		for i, match := range matches {
			result[i] = golf.Match{
				ID:            match.ID,
				TournamentID:  match.TournamentID,
				CourseID:      match.CourseID,
				TeeColorID:    match.TeeColorID,
				MatchFormatID: match.MatchFormatID,
				TeeTime:       pgTimestampToPtr(match.TeeTime),
				Handicapped:   match.Handicapped,
			}
		}
		return nil
	})

	return result, err
}

// toDomainMatch converts a sqlc Match to a domain Match
func toDomainMatch(m sqlc.Match) golf.Match {
	return golf.Match{
		ID:            m.ID,
		TournamentID:  m.TournamentID,
		CourseID:      m.CourseID,
		TeeColorID:    m.TeeColorID,
		MatchFormatID: m.MatchFormatID,
		TeeTime:       pgTimestampToPtr(m.TeeTime),
		Handicapped:   m.Handicapped,
	}
}
