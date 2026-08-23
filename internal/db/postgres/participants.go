package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type ParticipantsDB struct {
	db *DB
}

func NewParticipantsDB(db *DB) *ParticipantsDB {
	return &ParticipantsDB{db: db}
}

// CreateMatchParticipant adds a player (on a team) to a match. The composite FKs to
// team_members and teams reject an undrafted or wrong-team player as a FK violation,
// which mapWriteErr turns into ErrInvalidInput; a duplicate is ErrConflict.
func (p *ParticipantsDB) CreateMatchParticipant(ctx context.Context, tournamentID, matchID, playerID, teamID uuid.UUID) (*golf.MatchParticipant, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.MatchParticipant, error) {
		participant, err := q.CreateMatchParticipant(ctx, sqlc.CreateMatchParticipantParams{
			TournamentID: tournamentID,
			MatchID:      matchID,
			PlayerID:     playerID,
			TeamID:       teamID,
			TenantID:     tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("creating match participant: %w", mapWriteErr(err))
		}
		mp := toDomainParticipant(participant)
		return &mp, nil
	})
}

// DeleteMatchParticipant removes a player from a match. ErrNotFound if they weren't in it.
//
// The guard decides, from the match's committed scores, whether the removal is allowed. It
// runs behind the same lock the score write path takes, and in the same transaction as the
// delete, because otherwise a score landing between the two would be orphaned — the foreign
// key catches that for a per-player score and cannot see a one-ball one.
func (p *ParticipantsDB) DeleteMatchParticipant(
	ctx context.Context,
	matchID, playerID uuid.UUID,
	guard func(scores []golf.Score) error,
) error {
	return withTenantExec(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
		if _, err := q.LockMatchForScoring(ctx, sqlc.LockMatchForScoringParams{
			ID:       matchID,
			TenantID: tenantID,
		}); err != nil {
			return fmt.Errorf("locking match %s: %w", matchID, mapReadErr(err))
		}
		scores, err := q.ListScoresByMatch(ctx, sqlc.ListScoresByMatchParams{
			MatchID:  matchID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing scores for match %s: %w", matchID, err)
		}
		if err := guard(mapSlice(scores, toDomainScore)); err != nil {
			return err
		}
		rows, err := q.DeleteMatchParticipant(ctx, sqlc.DeleteMatchParticipantParams{
			MatchID:  matchID,
			PlayerID: playerID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("deleting match participant: %w", mapWriteErr(err))
		}
		if rows == 0 {
			return fmt.Errorf("deleting match participant: %w", golf.ErrNotFound)
		}
		return nil
	})
}

func (p *ParticipantsDB) ListMatchParticipants(ctx context.Context, matchID uuid.UUID) ([]golf.MatchParticipant, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.MatchParticipant, error) {
		participants, err := q.ListMatchParticipants(ctx, sqlc.ListMatchParticipantsParams{
			MatchID:  matchID,
			TenantID: tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing match participants: %w", err)
		}
		return mapSlice(participants, toDomainParticipant), nil
	})
}

func (p *ParticipantsDB) ListParticipantsWithPlayersByTournament(ctx context.Context, tournamentID uuid.UUID) ([]golf.MatchParticipantPlayer, error) {
	return withTenant(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.MatchParticipantPlayer, error) {
		rows, err := q.ListParticipantsWithPlayersByTournament(ctx, sqlc.ListParticipantsWithPlayersByTournamentParams{
			TournamentID: tournamentID,
			TenantID:     tenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing participants for tournament %s: %w", tournamentID, err)
		}
		return mapSlice(rows, toDomainParticipantPlayer), nil
	})
}

func toDomainParticipantPlayer(p sqlc.ListParticipantsWithPlayersByTournamentRow) golf.MatchParticipantPlayer {
	return golf.MatchParticipantPlayer{
		MatchID:   p.MatchID,
		TeamID:    p.TeamID,
		PlayerID:  p.PlayerID,
		FirstName: p.FirstName,
		LastName:  p.LastName,
	}
}

func toDomainParticipant(p sqlc.MatchParticipant) golf.MatchParticipant {
	return golf.MatchParticipant{
		TournamentID: p.TournamentID,
		MatchID:      p.MatchID,
		PlayerID:     p.PlayerID,
		TeamID:       p.TeamID,
	}
}
