package golf

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type fakeTournamentDB struct {
	tournament  *Tournament // returned by GetTournament; nil unless a test needs one
	tournaments []Tournament
	created     *CreateTournamentInput
	teamColors  []string
}

func (f *fakeTournamentDB) GetTournament(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	return f.tournament, nil
}
func (f *fakeTournamentDB) ListTournaments(ctx context.Context) ([]Tournament, error) {
	return f.tournaments, nil
}
func (f *fakeTournamentDB) CreateTournamentWithTeams(ctx context.Context, in CreateTournamentInput, teamColors []string) (*Tournament, error) {
	f.created = &in
	f.teamColors = teamColors
	return &Tournament{ID: tournamentID, Name: in.Name, StartDate: in.StartDate, EndDate: in.EndDate, Location: in.Location}, nil
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
	if got.ID != tournamentID || got.Name != "Manitoba Ryder Cup" {
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

// A cup is created months before anyone tees off, and its phase says so without the
// service having to go looking for matches it cannot have yet.
func TestCreateTournament_IsUpcoming(t *testing.T) {
	svc := &TournamentService{TournamentDB: &fakeTournamentDB{}}

	got, err := svc.CreateTournament(context.Background(), CreateTournamentInput{
		Name: "Manitoba Ryder Cup", StartDate: date(2026, 7, 1), EndDate: date(2026, 7, 3), Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("CreateTournament: %v", err)
	}
	if got.Phase != sdk.PhaseUpcoming {
		t.Errorf("phase = %q, want upcoming", got.Phase)
	}
}

// Phase is derived on read, so the record the repository returns carries none — the
// service has to put it there or every read publishes an empty string.
func TestGetTournament_CarriesThePhase(t *testing.T) {
	db := &fakeTournamentDB{tournament: &Tournament{ID: tournamentID, Name: "Manitoba Ryder Cup"}}
	results := &fakeResultDB{outcomes: []MatchOutcome{{Started: true}, {}}}
	svc := &TournamentService{TournamentDB: db, ResultDB: results}

	got, err := svc.GetTournament(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("GetTournament: %v", err)
	}
	if got.Phase != sdk.PhaseLive {
		t.Errorf("phase = %q, want live", got.Phase)
	}
}

// Each cup in the list gets its own phase, from one batched read of every outcome — the
// History page lists eighteen of them at once.
func TestListTournaments_PhasesEachCup(t *testing.T) {
	played, scheduled := uuid.New(), uuid.New()
	db := &fakeTournamentDB{tournaments: []Tournament{{ID: played}, {ID: scheduled}}}
	results := &fakeResultDB{allOutcomes: map[uuid.UUID][]MatchOutcome{
		played: {{Started: true, Finished: true}},
	}}
	svc := &TournamentService{TournamentDB: db, ResultDB: results}

	got, err := svc.ListTournaments(context.Background())
	if err != nil {
		t.Fatalf("ListTournaments: %v", err)
	}
	if got[0].Phase != sdk.PhaseFinished {
		t.Errorf("played cup: phase = %q, want finished", got[0].Phase)
	}
	// No matches at all, so nothing to derive a phase from but the absence itself.
	if got[1].Phase != sdk.PhaseUpcoming {
		t.Errorf("scheduled cup: phase = %q, want upcoming", got[1].Phase)
	}
}
