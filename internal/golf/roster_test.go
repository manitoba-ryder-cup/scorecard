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

type fakeTeamDB struct {
	captainCleared []uuid.UUID
}

func (f *fakeTeamDB) GetTeam(ctx context.Context, id uuid.UUID) (*Team, error) { return nil, nil }
func (f *fakeTeamDB) ListTeamsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]TeamWithCaptain, error) {
	return nil, nil
}
func (f *fakeTeamDB) SetTeamCaptain(ctx context.Context, teamID, captainID uuid.UUID) (*Team, error) {
	return nil, nil
}
func (f *fakeTeamDB) ClearCaptainForPlayer(ctx context.Context, teamID, playerID uuid.UUID) error {
	f.captainCleared = append(f.captainCleared, playerID)
	return nil
}
func (f *fakeTeamDB) ClearCaptain(ctx context.Context, teamID uuid.UUID) error { return nil }

type fakeTeamMemberDB struct {
	undrafted []uuid.UUID
}

func (f *fakeTeamMemberDB) CreateTeamMember(ctx context.Context, teamID, playerID, tournamentID uuid.UUID) (*TeamMember, error) {
	return nil, nil
}
func (f *fakeTeamMemberDB) DeleteTeamMember(ctx context.Context, teamID, playerID uuid.UUID) error {
	f.undrafted = append(f.undrafted, playerID)
	return nil
}

// Being in a lineup is not being played: a draft is edited right up to the first tee.
func TestUndraftPlayer_AllowedBeforeTheyHaveBeenScored(t *testing.T) {
	db := &fakeTeamMemberDB{}
	svc := &RosterService{TeamMemberDB: db, TeamDB: &fakeTeamDB{}}

	if err := svc.UndraftPlayer(context.Background(), teamA, playerA); err != nil {
		t.Fatalf("UndraftPlayer: %v", err)
	}
	if len(db.undrafted) != 1 || db.undrafted[0] != playerA {
		t.Errorf("undrafted = %v, want the player", db.undrafted)
	}
}

// A side's name derives from its captain, so one who has left the team cannot stay it.
func TestUndraftPlayer_ClearsThemAsCaptain(t *testing.T) {
	teams := &fakeTeamDB{}
	svc := &RosterService{TeamMemberDB: &fakeTeamMemberDB{}, TeamDB: teams}

	if err := svc.UndraftPlayer(context.Background(), teamA, playerA); err != nil {
		t.Fatalf("UndraftPlayer: %v", err)
	}
	if len(teams.captainCleared) != 1 || teams.captainCleared[0] != playerA {
		t.Errorf("captain cleared for = %v, want the undrafted player", teams.captainCleared)
	}
}
