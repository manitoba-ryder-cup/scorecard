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

// SetMatchLineup replaces a match's lineup with entries. The lock spans the whole exchange:
// the scored check that decides whether the lineup may move at all is read before the write
// that acts on it, and a score landing in between would make that answer untrue.
//
// The size rule needs no lock — it is checked against the set being written, not the one
// stored — which is the reason a lineup arrives whole rather than a player at a time.
func (p *ParticipantsDB) SetMatchLineup(ctx context.Context, tournamentID, matchID uuid.UUID, entries []golf.MatchParticipant) error {
	return withTenantExec(ctx, p.db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
		if err := lockMatch(ctx, q, matchID, tenantID); err != nil {
			return err
		}
		if err := refuseIfScored(ctx, q, matchID, tenantID, golf.ErrScoredMatchLineup); err != nil {
			return err
		}
		match, err := q.GetMatch(ctx, sqlc.GetMatchParams{ID: matchID, TenantID: tenantID})
		if err != nil {
			return fmt.Errorf("reading match %s: %w", matchID, mapReadErr(err, golf.ErrMatchNotFound))
		}
		format, err := q.GetMatchFormat(ctx, match.MatchFormatID)
		if err != nil {
			return fmt.Errorf("reading format %s: %w", match.MatchFormatID, mapReadErr(err, golf.ErrNotFound))
		}
		if !golf.LineupFits(entries, format.PlayersPerSide) {
			return fmt.Errorf("%w: format %s takes %d a side", golf.ErrLineupSize, match.MatchFormatID, format.PlayersPerSide)
		}

		if err := q.DeleteMatchLineup(ctx, sqlc.DeleteMatchLineupParams{MatchID: matchID, TenantID: tenantID}); err != nil {
			return fmt.Errorf("clearing lineup for match %s: %w", matchID, mapWriteErr(err))
		}
		for _, e := range entries {
			// An undrafted or wrong-team player trips a composite FK, which mapWriteErr reads
			// as the caller's error rather than ours.
			if _, err := q.CreateMatchParticipant(ctx, sqlc.CreateMatchParticipantParams{
				TournamentID: tournamentID,
				MatchID:      matchID,
				PlayerID:     e.PlayerID,
				TeamID:       e.TeamID,
				TenantID:     tenantID,
			}); err != nil {
				return fmt.Errorf("adding %s to match %s: %w", e.PlayerID, matchID, mapWriteErr(err))
			}
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
