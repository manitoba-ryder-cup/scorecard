package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

// ResultsDB reads/writes match_results and the aggregates derived from it.
type ResultsDB struct {
	db *DB
}

func NewResultsDB(db *DB) *ResultsDB {
	return &ResultsDB{db: db}
}

func (r *ResultsDB) GetMatchResult(ctx context.Context, matchID uuid.UUID) (*golf.StoredResult, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.StoredResult, error) {
		row, err := q.GetMatchResult(ctx, sqlc.GetMatchResultParams{MatchID: matchID, TenantID: tenantID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil // no result yet
			}
			return nil, fmt.Errorf("getting match result %s: %w", matchID, err)
		}
		return &golf.StoredResult{
			Finished:       row.Finished,
			LeaderTeamID:   row.LeaderTeamID,
			Lead:           int(row.Lead),
			HolesRemaining: int(row.HolesRemaining),
		}, nil
	})
}

func (r *ResultsDB) ListMatchOutcomes(ctx context.Context, tournamentID uuid.UUID) ([]golf.MatchOutcome, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.MatchOutcome, error) {
		rows, err := q.ListMatchOutcomes(ctx, sqlc.ListMatchOutcomesParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing match outcomes: %w", err)
		}
		return mapSlice(rows, func(row sqlc.ListMatchOutcomesRow) golf.MatchOutcome {
			return golf.MatchOutcome{Finished: row.Finished, WinnerTeamID: row.LeaderTeamID}
		}), nil
	})
}

func (r *ResultsDB) ListTournamentPlayerRecords(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]golf.PlayerRecord, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (map[uuid.UUID]golf.PlayerRecord, error) {
		rows, err := q.ListTournamentPlayerRecords(ctx, sqlc.ListTournamentPlayerRecordsParams{TournamentID: tournamentID, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("listing tournament player records: %w", err)
		}
		records := make(map[uuid.UUID]golf.PlayerRecord, len(rows))
		for _, row := range rows {
			records[row.PlayerID] = golf.PlayerRecord{
				Wins:   int32(row.Wins),
				Losses: int32(row.Losses),
				Ties:   int32(row.Ties),
			}
		}
		return records, nil
	})
}

// ListAllTournamentStandings returns every tournament's teams and match outcomes in one
// transaction, so the domain can settle each Cup without a query per tournament.
func (r *ResultsDB) ListAllTournamentStandings(ctx context.Context) (map[uuid.UUID]golf.TournamentStandings, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (map[uuid.UUID]golf.TournamentStandings, error) {
		return tournamentStandings(ctx, q, tenantID)
	})
}

func tournamentStandings(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID) (map[uuid.UUID]golf.TournamentStandings, error) {
	teams, err := q.ListAllTeams(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing teams: %w", err)
	}
	outcomes, err := q.ListAllMatchOutcomes(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing match outcomes: %w", err)
	}

	standings := make(map[uuid.UUID]golf.TournamentStandings)
	for _, t := range teams {
		s := standings[t.TournamentID]
		s.TeamIDs = append(s.TeamIDs, t.TeamID)
		standings[t.TournamentID] = s
	}
	for _, o := range outcomes {
		s := standings[o.TournamentID]
		s.Outcomes = append(s.Outcomes, golf.MatchOutcome{Finished: o.Finished, WinnerTeamID: o.LeaderTeamID})
		standings[o.TournamentID] = s
	}
	return standings, nil
}

// ListCupData fetches the standings and the draft records together, so counting Cups is
// one round trip rather than two.
func (r *ResultsDB) ListCupData(ctx context.Context) (golf.CupData, error) {
	return withTenant(ctx, r.db, func(q *sqlc.Queries, tenantID uuid.UUID) (golf.CupData, error) {
		standings, err := tournamentStandings(ctx, q, tenantID)
		if err != nil {
			return golf.CupData{}, err
		}
		rows, err := q.ListTeamMemberships(ctx, tenantID)
		if err != nil {
			return golf.CupData{}, fmt.Errorf("listing team memberships: %w", err)
		}
		return golf.CupData{
			Standings: standings,
			Memberships: mapSlice(rows, func(row sqlc.ListTeamMembershipsRow) golf.TeamMembership {
				return golf.TeamMembership{PlayerID: row.PlayerID, TournamentID: row.TournamentID, TeamID: row.TeamID}
			}),
		}, nil
	})
}
