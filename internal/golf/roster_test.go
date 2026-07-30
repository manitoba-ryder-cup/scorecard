package golf

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type fakeTournamentPlayerDB struct {
	created []EnterPlayerInput
	updated []UpdateRosterEntryInput
}

func (f *fakeTournamentPlayerDB) CreateTournamentPlayer(ctx context.Context, in EnterPlayerInput) (*TournamentPlayer, error) {
	f.created = append(f.created, in)
	return &TournamentPlayer{TournamentID: in.TournamentID, PlayerID: in.PlayerID, Tier: in.Tier}, nil
}

func (f *fakeTournamentPlayerDB) UpdateTournamentPlayer(ctx context.Context, in UpdateRosterEntryInput) (*TournamentPlayer, error) {
	f.updated = append(f.updated, in)
	tp := &TournamentPlayer{TournamentID: in.TournamentID, PlayerID: in.PlayerID}
	if in.Tier != nil {
		tp.Tier = *in.Tier
	}
	return tp, nil
}

func (f *fakeTournamentPlayerDB) ListTournamentPlayers(ctx context.Context, tournamentID uuid.UUID) ([]TournamentPlayer, error) {
	return nil, nil
}

func (f *fakeTournamentPlayerDB) ListTournamentPlayersByTeam(ctx context.Context, teamID uuid.UUID) ([]TournamentPlayer, error) {
	return nil, nil
}

// Defaulting an omitted tier is a rule about roster entries, so it belongs here rather
// than in each caller — the API and the seed CLI both enter players.
func TestEnterPlayer_DefaultsAnOmittedTier(t *testing.T) {
	db := &fakeTournamentPlayerDB{}
	svc := &RosterService{TournamentPlayerDB: db}

	if _, err := svc.EnterPlayer(context.Background(), EnterPlayerInput{
		TournamentID: tournamentID, PlayerID: playerA, Tier: "",
	}); err != nil {
		t.Fatalf("EnterPlayer: %v", err)
	}

	if len(db.created) != 1 {
		t.Fatalf("want 1 entry created, got %d", len(db.created))
	}
	if db.created[0].Tier != sdk.DefaultTier {
		t.Errorf("tier = %q, want %q", db.created[0].Tier, sdk.DefaultTier)
	}
}

func TestEnterPlayer_KeepsASuppliedTier(t *testing.T) {
	db := &fakeTournamentPlayerDB{}
	svc := &RosterService{TournamentPlayerDB: db}

	if _, err := svc.EnterPlayer(context.Background(), EnterPlayerInput{
		TournamentID: tournamentID, PlayerID: playerA, Tier: "gold",
	}); err != nil {
		t.Fatalf("EnterPlayer: %v", err)
	}

	if db.created[0].Tier != "gold" {
		t.Errorf("tier = %q, want gold", db.created[0].Tier)
	}
}

// Entering a player defaults an omitted tier; updating one must not. On an update an
// omitted field means leave it alone, so defaulting would quietly demote every entry
// whose biography someone edited.
func TestUpdatePlayer_LeavesAnOmittedTierAlone(t *testing.T) {
	db := &fakeTournamentPlayerDB{}
	svc := &RosterService{TournamentPlayerDB: db}

	bio := "Holed out from the car park."
	if _, err := svc.UpdatePlayer(context.Background(), UpdateRosterEntryInput{
		TournamentID: tournamentID, PlayerID: playerA, Biography: &bio,
	}); err != nil {
		t.Fatalf("UpdatePlayer: %v", err)
	}

	if len(db.updated) != 1 {
		t.Fatalf("want 1 entry updated, got %d", len(db.updated))
	}
	if db.updated[0].Tier != nil {
		t.Errorf("tier = %v, want nil so the stored one survives", *db.updated[0].Tier)
	}
	if db.updated[0].Hdcp != nil {
		t.Errorf("hdcp = %v, want nil so the stored one survives", *db.updated[0].Hdcp)
	}
}
