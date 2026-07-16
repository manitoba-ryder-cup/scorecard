package golf

import (
	"context"
	"errors"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type fakeTeamDB struct {
	created *CreateTeamInput
}

func (f *fakeTeamDB) GetTeam(ctx context.Context, id int32) (*Team, error) { return nil, nil }
func (f *fakeTeamDB) ListTeamsByTournament(ctx context.Context, tournamentID int32) ([]Team, error) {
	return nil, nil
}
func (f *fakeTeamDB) CreateTeam(ctx context.Context, in CreateTeamInput) (*Team, error) {
	f.created = &in
	return &Team{ID: 9, TournamentID: in.TournamentID, Color: in.Color, CaptainID: in.CaptainID}, nil
}

func TestCreateTeam_Valid(t *testing.T) {
	db := &fakeTeamDB{}
	svc := &TeamService{TeamDB: db}

	got, err := svc.CreateTeam(context.Background(), CreateTeamInput{TournamentID: 3, Color: sdk.TeamColorRed})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if got.ID != 9 || got.Color != sdk.TeamColorRed || got.TournamentID != 3 {
		t.Errorf("unexpected result: %+v", got)
	}
	if db.created == nil {
		t.Error("input not passed through")
	}
}

func TestCreateTeam_RejectsInvalidColor(t *testing.T) {
	db := &fakeTeamDB{}
	svc := &TeamService{TeamDB: db}

	_, err := svc.CreateTeam(context.Background(), CreateTeamInput{TournamentID: 3, Color: "Green"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if db.created != nil {
		t.Error("must not write on validation failure")
	}
}
