package golf

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type fakeTournamentDB struct {
	created    *CreateTournamentInput
	teamColors []string
}

func (f *fakeTournamentDB) GetTournament(ctx context.Context, id int32) (*Tournament, error) {
	return nil, nil
}
func (f *fakeTournamentDB) ListTournaments(ctx context.Context) ([]Tournament, error) {
	return nil, nil
}
func (f *fakeTournamentDB) CreateTournamentWithTeams(ctx context.Context, in CreateTournamentInput, teamColors []string) (*Tournament, error) {
	f.created = &in
	f.teamColors = teamColors
	return &Tournament{ID: 1, Name: in.Name, StartDate: in.StartDate, EndDate: in.EndDate, Location: in.Location}, nil
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestCreateTournament_Valid(t *testing.T) {
	db := &fakeTournamentDB{}
	svc := &TournamentService{TournamentDB: db}

	got, err := svc.CreateTournament(context.Background(), CreateTournamentInput{
		Name: "Manitoba Ryder Cup", StartDate: date(2026, 7, 1), EndDate: date(2026, 7, 3), Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("CreateTournament: %v", err)
	}
	if got.ID != 1 || got.Name != "Manitoba Ryder Cup" {
		t.Errorf("unexpected result: %+v", got)
	}
	if db.created == nil || db.created.Location != "Winnipeg" {
		t.Errorf("input not passed through: %+v", db.created)
	}
	// The tournament is seeded with exactly its two sides, Red and Blue.
	if len(db.teamColors) != 2 || db.teamColors[0] != sdk.TeamColorRed || db.teamColors[1] != sdk.TeamColorBlue {
		t.Errorf("want [Red Blue] teams seeded, got %v", db.teamColors)
	}
}

func TestCreateTournament_RejectsEmptyName(t *testing.T) {
	db := &fakeTournamentDB{}
	svc := &TournamentService{TournamentDB: db}

	_, err := svc.CreateTournament(context.Background(), CreateTournamentInput{
		Name: "  ", StartDate: date(2026, 7, 1), EndDate: date(2026, 7, 3),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if db.created != nil {
		t.Error("must not write on validation failure")
	}
}

func TestCreateTournament_RejectsEndBeforeStart(t *testing.T) {
	db := &fakeTournamentDB{}
	svc := &TournamentService{TournamentDB: db}

	_, err := svc.CreateTournament(context.Background(), CreateTournamentInput{
		Name: "Cup", StartDate: date(2026, 7, 3), EndDate: date(2026, 7, 1),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if db.created != nil {
		t.Error("must not write on validation failure")
	}
}
