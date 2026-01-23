package postgres

import (
	"context"
	"fmt"

	"github.com/travisbale/knowhere/identity"
	"github.com/travisbale/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/travisbale/scorecard/internal/golf"
)

type TournamentsDB struct {
	db *DB
}

func NewTournamentsDB(db *DB) *TournamentsDB {
	return &TournamentsDB{db: db}
}

func (t *TournamentsDB) GetTournament(ctx context.Context, id int32) (*golf.Tournament, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Tournament
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		tournament, err := q.GetTournament(ctx, sqlc.GetTournamentParams{ID: id, TenantID: tenantID})
		if err != nil {
			return fmt.Errorf("getting tournament %d: %w", id, err)
		}
		result = &golf.Tournament{
			ID:        tournament.ID,
			TenantID:  tournament.TenantID,
			Name:      tournament.Name,
			StartDate: tournament.StartDate,
			EndDate:   tournament.EndDate,
			Location:  tournament.Location,
		}
		return nil
	})
	return result, err
}

func (t *TournamentsDB) ListTournaments(ctx context.Context) ([]golf.Tournament, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.Tournament
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		tournaments, err := q.ListTournaments(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("listing tournaments: %w", err)
		}
		result = make([]golf.Tournament, len(tournaments))
		for i, tournament := range tournaments {
			result[i] = golf.Tournament{
				ID:        tournament.ID,
				TenantID:  tournament.TenantID,
				Name:      tournament.Name,
				StartDate: tournament.StartDate,
				EndDate:   tournament.EndDate,
				Location:  tournament.Location,
			}
		}
		return nil
	})
	return result, err
}
