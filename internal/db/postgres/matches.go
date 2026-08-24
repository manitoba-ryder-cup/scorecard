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

// UpdateMatch applies in, having first shown guard the match as it stands and whether it has
// been scored. The lock is held across both, so the answer guard gives cannot be raced by a
// score landing before the update.
func (m *MatchesDB) UpdateMatch(
	ctx context.Context,
	in golf.UpdateMatchInput,
	guard func(current golf.Match, scored bool) error,
) (*golf.Match, error) {
	return withTenant(ctx, m.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Match, error) {
		if err := lockMatchForScoring(ctx, q, in.ID, tenantID); err != nil {
			return nil, err
		}
		current, err := q.GetMatch(ctx, sqlc.GetMatchParams{ID: in.ID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("reading match %s: %w", in.ID, mapReadErr(err, golf.ErrMatchNotFound))
		}
		scored, err := matchHasScores(ctx, q, in.ID, tenantID)
		if err != nil {
			return nil, err
		}
		if err := guard(toDomainMatch(current), scored); err != nil {
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
			return nil, fmt.Errorf("updating match %s: %w", in.ID, mapWriteErr(mapReadErr(err, golf.ErrMatchNotFound)))
		}
		dm := toDomainMatch(match)
		return &dm, nil
	})
}

// lockMatchForScoring takes the match's FOR UPDATE lock. Held before the first read, so a
// score cannot land between a guard deciding the match is unscored and the write acting on
// that decision.
func lockMatchForScoring(ctx context.Context, q *sqlc.Queries, matchID, tenantID uuid.UUID) error {
	if _, err := q.LockMatchForScoring(ctx, sqlc.LockMatchForScoringParams{
		ID:       matchID,
		TenantID: tenantID,
	}); err != nil {
		return fmt.Errorf("locking match %s: %w", matchID, mapReadErr(err, golf.ErrMatchNotFound))
	}
	return nil
}

func matchHasScores(ctx context.Context, q *sqlc.Queries, matchID, tenantID uuid.UUID) (bool, error) {
	scored, err := q.MatchHasScores(ctx, sqlc.MatchHasScoresParams{
		MatchID:  matchID,
		TenantID: tenantID,
	})
	if err != nil {
		return false, fmt.Errorf("checking scores for match %s: %w", matchID, err)
	}
	return scored, nil
}

// refuseIfScored returns refusal if the match has scores. Kept apart from the lock because a
// tee set move has to read the match and decide the write moves it before this applies.
func refuseIfScored(ctx context.Context, q *sqlc.Queries, matchID, tenantID uuid.UUID, refusal error) error {
	scored, err := matchHasScores(ctx, q, matchID, tenantID)
	if err != nil {
		return err
	}
	if scored {
		return fmt.Errorf("%w: match %s", refusal, matchID)
	}
	return nil
}

// DeleteMatch removes a match, taking its lineup with it. Refused for a match that has been
// scored: losing results is a decision, not a side effect of tidying up.
func (m *MatchesDB) DeleteMatch(ctx context.Context, id uuid.UUID) error {
	return withTenantExec(ctx, m.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
		if err := lockMatchForScoring(ctx, q, id, tenantID); err != nil {
			return err
		}
		if err := refuseIfScored(ctx, q, id, tenantID, golf.ErrScoredMatchDelete); err != nil {
			return err
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

// GetMatch retrieves a match by ID with tenant isolation
func (m *MatchesDB) GetMatch(ctx context.Context, id uuid.UUID) (*golf.Match, error) {
	return withTenant(ctx, m.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Match, error) {
		match, err := q.GetMatch(ctx, sqlc.GetMatchParams{ID: id, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("getting match %s: %w", id, mapReadErr(err, golf.ErrMatchNotFound))
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
