package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

// MatchesDB handles match database operations
type MatchesDB struct {
	db *DB
}

// NewMatchesDB creates a new MatchesDB
func NewMatchesDB(db *DB) *MatchesDB {
	return &MatchesDB{db: db}
}

// CreateMatch inserts a new match. Unknown course/tee/format references (or a tee not
// configured for the course) surface as ErrInvalidInput via mapWriteErr (FK violation).
func (m *MatchesDB) CreateMatch(ctx context.Context, in golf.CreateMatchInput) (*golf.Match, error) {
	return withTenant(ctx, m.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Match, error) {
		match, err := q.CreateMatch(ctx, sqlc.CreateMatchParams{
			TournamentID:  in.TournamentID,
			CourseID:      in.CourseID,
			TeeColorID:    in.TeeColorID,
			MatchFormatID: in.MatchFormatID,
			TenantID:      tenantID,
			TeeTime:       in.TeeTime,
			Handicapped:   in.Handicapped,
		})
		if err != nil {
			return nil, fmt.Errorf("creating match: %w", mapWriteErr(err))
		}
		dm := toDomainMatch(match)
		return &dm, nil
	})
}

// GetMatch retrieves a match by ID with tenant isolation
func (m *MatchesDB) GetMatch(ctx context.Context, id uuid.UUID) (*golf.Match, error) {
	return withTenant(ctx, m.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Match, error) {
		match, err := q.GetMatch(ctx, sqlc.GetMatchParams{ID: id, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("getting match %s: %w", id, mapReadErr(err))
		}
		dm := toDomainMatch(match)
		return &dm, nil
	})
}

// ListMatchesByTournament retrieves all matches for a tournament
func (m *MatchesDB) ListMatchesByTournament(ctx context.Context, tournamentID uuid.UUID) ([]golf.Match, error) {
	return withTenant(ctx, m.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.Match, error) {
		matches, err := q.ListMatchesByTournament(ctx, sqlc.ListMatchesByTournamentParams{
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing matches for tournament %s: %w", tournamentID, err)
		}
		return mapSlice(matches, toDomainMatch), nil
	})
}

func (m *MatchesDB) ListMatchDetailsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]golf.MatchDetail, error) {
	return withTenant(ctx, m.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.MatchDetail, error) {
		rows, err := q.ListMatchesWithDetailsByTournament(ctx, sqlc.ListMatchesWithDetailsByTournamentParams{
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing match details for tournament %s: %w", tournamentID, err)
		}
		return mapSlice(rows, toDomainMatchDetail), nil
	})
}

// toDomainMatch converts a sqlc Match to a domain Match
func toDomainMatch(m sqlc.Match) golf.Match {
	return golf.Match{
		ID:            m.ID,
		TournamentID:  m.TournamentID,
		CourseID:      m.CourseID,
		TeeColorID:    m.TeeColorID,
		MatchFormatID: m.MatchFormatID,
		TeeTime:       m.TeeTime,
		Handicapped:   m.Handicapped,
	}
}

func toDomainMatchDetail(m sqlc.ListMatchesWithDetailsByTournamentRow) golf.MatchDetail {
	return golf.MatchDetail{
		Match: golf.Match{
			ID:            m.ID,
			TournamentID:  m.TournamentID,
			CourseID:      m.CourseID,
			TeeColorID:    m.TeeColorID,
			MatchFormatID: m.MatchFormatID,
			TeeTime:       m.TeeTime,
			Handicapped:   m.Handicapped,
		},
		FormatName: m.FormatName,
		CourseName: m.CourseName,
	}
}
