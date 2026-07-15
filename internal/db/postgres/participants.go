package postgres

import (
	"context"
	"fmt"

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

func (p *ParticipantsDB) ListMatchParticipants(ctx context.Context, matchID int32) ([]golf.MatchParticipant, error) {
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
				TenantID:     participant.TenantID,
			}
		}
		return nil
	})
	return result, err
}

// ListMatchParticipantsWithTeam returns a map of playerID -> teamName
func (p *ParticipantsDB) ListMatchParticipantsWithTeam(ctx context.Context, matchID int32) (map[int32]string, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[int32]string)
	err = p.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		participants, err := q.ListMatchParticipantsWithTeam(ctx, sqlc.ListMatchParticipantsWithTeamParams{
			MatchID:  matchID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("listing match participants with team: %w", err)
		}
		for _, participant := range participants {
			result[participant.PlayerID] = participant.TeamName
		}
		return nil
	})
	return result, err
}
