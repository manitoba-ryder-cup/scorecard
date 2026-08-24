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

func (m *MatchesDB) UpdateMatch(ctx context.Context, in golf.UpdateMatchInput) (*golf.Match, error) {
	return withTenant(ctx, m.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Match, error) {
		if err := refuseTeeSetMoveOnScoredMatch(ctx, q, in, tenantID); err != nil {
			return nil, err
		}
		match, err := q.UpdateMatch(ctx, sqlc.UpdateMatchParams{
			ID:            in.ID,
			TenantID:      tenantID,
			CourseID:      in.CourseID,
			TeeColorID:    in.TeeColorID,
			MatchFormatID: in.MatchFormatID,
			TeeTime:       in.TeeTime,
			Handicapped:   in.Handicapped,
		})
		if err != nil {
			// No row means no such match here; an unknown course or tee is the FK.
			return nil, fmt.Errorf("updating match %s: %w", in.ID, mapWriteErr(mapReadErr(err)))
		}
		dm := toDomainMatch(match)
		return &dm, nil
	})
}

// DeleteMatch removes a match, taking its lineup with it. Refused for a match that has been
// scored: losing results is a decision, not a side effect of tidying up.
func (m *MatchesDB) DeleteMatch(ctx context.Context, id uuid.UUID) error {
	return withTenantExec(ctx, m.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
		// Taken before the check, so a submission cannot land between it and the delete.
		if _, err := q.LockMatchForScoring(ctx, sqlc.LockMatchForScoringParams{
			ID:       id,
			TenantID: tenantID,
		}); err != nil {
			return fmt.Errorf("locking match %s: %w", id, mapReadErr(err))
		}
		scored, err := q.MatchHasScores(ctx, sqlc.MatchHasScoresParams{
			MatchID:  id,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("checking scores for match %s: %w", id, err)
		}
		if scored {
			return fmt.Errorf("%w: match %s", golf.ErrMatchScored, id)
		}
		if _, err := q.DeleteMatch(ctx, sqlc.DeleteMatchParams{
			ID:       id,
			TenantID: tenantID,
		}); err != nil {
			return fmt.Errorf("deleting match %s: %w", id, mapWriteErr(err))
		}
		return nil
	})
}

// refuseTeeSetMoveOnScoredMatch turns a scored match's tee set into a 409 naming the way
// through, where the foreign key underneath would answer an unexplained 400.
//
// Scores carry the course and tee they were recorded against and read their par and stroke
// index from that tee set, so moving a scored match would leave them describing a round nobody
// played.
func refuseTeeSetMoveOnScoredMatch(ctx context.Context, q *sqlc.Queries, in golf.UpdateMatchInput, tenantID uuid.UUID) error {
	if in.CourseID == nil && in.TeeColorID == nil {
		return nil
	}
	// Locked before it is read, so a score cannot land between finding the match unscored and
	// moving the tee set out from under it.
	if _, err := q.LockMatchForScoring(ctx, sqlc.LockMatchForScoringParams{ID: in.ID, TenantID: tenantID}); err != nil {
		return fmt.Errorf("locking match %s: %w", in.ID, mapReadErr(err))
	}
	current, err := q.GetMatch(ctx, sqlc.GetMatchParams{ID: in.ID, TenantID: tenantID})
	if err != nil {
		return fmt.Errorf("reading match %s: %w", in.ID, mapReadErr(err))
	}
	moves := (in.CourseID != nil && *in.CourseID != current.CourseID) ||
		(in.TeeColorID != nil && *in.TeeColorID != current.TeeColorID)
	if !moves {
		return nil
	}
	scored, err := q.MatchHasScores(ctx, sqlc.MatchHasScoresParams{MatchID: in.ID, TenantID: tenantID})
	if err != nil {
		return fmt.Errorf("checking scores for match %s: %w", in.ID, err)
	}
	if scored {
		return fmt.Errorf("%w: match %s", golf.ErrMatchScored, in.ID)
	}
	return nil
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
