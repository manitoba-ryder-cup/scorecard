package postgres

import (
	"context"
	"fmt"

	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
)

type TournamentsDB struct {
	db *DB
}

func NewTournamentsDB(db *DB) *TournamentsDB {
	return &TournamentsDB{db: db}
}

func (t *TournamentsDB) CreateTournamentWithTeams(ctx context.Context, in golf.CreateTournamentInput, teamColors []string) (*golf.Tournament, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Tournament
	// A single WithTenantContext closure is one transaction, so the tournament and its
	// teams commit together or not at all — no tournament ever exists half-created.
	err = t.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		tournament, err := q.CreateTournament(ctx, sqlc.CreateTournamentParams{
			TenantID:  tenantID,
			Name:      in.Name,
			StartDate: in.StartDate,
			EndDate:   in.EndDate,
			Location:  in.Location,
		})
		if err != nil {
			return fmt.Errorf("creating tournament: %w", mapWriteErr(err))
		}
		for _, color := range teamColors {
			if _, err := q.CreateTeam(ctx, sqlc.CreateTeamParams{
				TenantID:     tenantID,
				TournamentID: tournament.ID,
				Color:        color,
				CaptainID:    nil, // captain is assigned later, once the roster is set
			}); err != nil {
				return fmt.Errorf("creating %s team: %w", color, mapWriteErr(err))
			}
		}
		td := toDomainTournament(tournament)
		result = &td
		return nil
	})
	return result, err
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
		td := toDomainTournament(tournament)
		result = &td
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
			result[i] = toDomainTournament(tournament)
		}
		return nil
	})
	return result, err
}

func toDomainTournament(t sqlc.Tournament) golf.Tournament {
	return golf.Tournament{
		ID:        t.ID,
		Name:      t.Name,
		StartDate: t.StartDate,
		EndDate:   t.EndDate,
		Location:  t.Location,
	}
}
