package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
)

type ParticipantsDB struct {
	db *DB
}

func NewParticipantsDB(db *DB) *ParticipantsDB {
	return &ParticipantsDB{db: db}
}

func (p *ParticipantsDB) ListMatchParticipants(ctx context.Context, matchID uuid.UUID) ([]golf.MatchParticipant, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.MatchParticipant
	err = p.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		participants, err := q.ListMatchParticipants(ctx, sqlc.ListMatchParticipantsParams{
			MatchID:  matchID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing match participants: %w", err)
		}
		result = make([]golf.MatchParticipant, len(participants))
		for i, participant := range participants {
			result[i] = golf.MatchParticipant{
				TournamentID: participant.TournamentID,
				MatchID:      participant.MatchID,
				PlayerID:     participant.PlayerID,
				TeamID:       participant.TeamID,
			}
		}
		return nil
	})
	return result, err
}
