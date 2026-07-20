package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type TournamentsDB struct {
	db *DB
}

func NewTournamentsDB(db *DB) *TournamentsDB {
	return &TournamentsDB{db: db}
}

func (t *TournamentsDB) CreateTournamentWithTeams(ctx context.Context, in golf.CreateTournamentInput, teamColors []string) (*golf.Tournament, error) {
	// A single withTenant closure is one transaction, so the tournament and its teams
	// commit together or not at all — no tournament ever exists half-created.
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Tournament, error) {
		tournament, err := q.CreateTournament(ctx, sqlc.CreateTournamentParams{
			TenantID:  tenantID,
			Name:      in.Name,
			StartDate: in.StartDate,
			EndDate:   in.EndDate,
			Location:  in.Location,
		})
		if err != nil {
			return nil, fmt.Errorf("creating tournament: %w", mapWriteErr(err))
		}
		for _, color := range teamColors {
			if _, err := q.CreateTeam(ctx, sqlc.CreateTeamParams{
				TenantID:     tenantID,
				TournamentID: tournament.ID,
				Color:        color,
				CaptainID:    nil, // captain is assigned later, once the roster is set
			}); err != nil {
				return nil, fmt.Errorf("creating %s team: %w", color, mapWriteErr(err))
			}
		}
		td := toDomainTournament(tournament)
		return &td, nil
	})
}

func (t *TournamentsDB) GetTournament(ctx context.Context, id uuid.UUID) (*golf.Tournament, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Tournament, error) {
		tournament, err := q.GetTournament(ctx, sqlc.GetTournamentParams{ID: id, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("getting tournament %s: %w", id, mapReadErr(err))
		}
		td := toDomainTournament(tournament)
		return &td, nil
	})
}

func (t *TournamentsDB) ListTournaments(ctx context.Context) ([]golf.Tournament, error) {
	return withTenant(ctx, t.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.Tournament, error) {
		tournaments, err := q.ListTournaments(ctx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("listing tournaments: %w", err)
		}
		return mapSlice(tournaments, toDomainTournament), nil
	})
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
