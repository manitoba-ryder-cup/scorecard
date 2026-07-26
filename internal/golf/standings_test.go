package golf

import (
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

func won(team uuid.UUID) MatchOutcome { return MatchOutcome{Finished: true, WinnerTeamID: &team} }
func halved() MatchOutcome            { return MatchOutcome{Finished: true} }
func unplayed() MatchOutcome          { return MatchOutcome{} }

func TestComputeTeamPoints(t *testing.T) {
	teams := []uuid.UUID{teamA, teamB}

	tests := []struct {
		name     string
		outcomes []MatchOutcome
		wantA    float64
		wantB    float64
	}{
		{"a win is a point", []MatchOutcome{won(teamA)}, 1, 0},
		{"a halve is half each", []MatchOutcome{halved()}, 0.5, 0.5},
		{"an unfinished match scores nothing", []MatchOutcome{unplayed()}, 0, 0},
		{"points accumulate", []MatchOutcome{won(teamA), won(teamA), halved(), won(teamB)}, 2.5, 1.5},
		{"no matches is nil-all", nil, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeTeamPoints(tc.outcomes, teams)
			if got[teamA] != tc.wantA || got[teamB] != tc.wantB {
				t.Errorf("points = A:%v B:%v, want A:%v B:%v", got[teamA], got[teamB], tc.wantA, tc.wantB)
			}
		})
	}
}

// Every team must appear, so a side that has won nothing still reports zero rather
// than being absent from the standings.
func TestComputeTeamPoints_IncludesTeamsWithNoPoints(t *testing.T) {
	got := ComputeTeamPoints([]MatchOutcome{won(teamA)}, []uuid.UUID{teamA, teamB})

	if len(got) != 2 {
		t.Fatalf("want an entry per team, got %d", len(got))
	}
	if _, ok := got[teamB]; !ok {
		t.Error("team with no points is missing from the standings")
	}
}

func TestIsTournamentComplete(t *testing.T) {
	tests := []struct {
		name     string
		outcomes []MatchOutcome
		want     bool
	}{
		{"every match final", []MatchOutcome{won(teamA), halved()}, true},
		{"one match outstanding", []MatchOutcome{won(teamA), unplayed()}, false},
		// A tournament with no matches has not been played, so it is not complete —
		// otherwise a freshly created Cup would report itself finished.
		{"no matches", nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTournamentComplete(tc.outcomes); got != tc.want {
				t.Errorf("complete = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestComputeTournamentOutcome(t *testing.T) {
	teams := []uuid.UUID{teamA, teamB}

	tests := []struct {
		name         string
		outcomes     []MatchOutcome
		wantFinished bool
		wantWinner   *uuid.UUID
	}{
		{"sole leader wins the cup", []MatchOutcome{won(teamA), halved()}, true, &teamA},
		// A tie retains the cup for the holder, so nobody wins it outright.
		{"a tie has no winner", []MatchOutcome{won(teamA), won(teamB)}, true, nil},
		{"all halved is a tie", []MatchOutcome{halved(), halved()}, true, nil},
		// No early clinch: the Cup is undecided while any match is outstanding, even
		// when one side already leads unassailably.
		{"leading but unfinished", []MatchOutcome{won(teamA), won(teamA), unplayed()}, false, nil},
		{"no matches", nil, false, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeTournamentOutcome(tc.outcomes, teams)
			if got.Finished != tc.wantFinished {
				t.Errorf("finished = %v, want %v", got.Finished, tc.wantFinished)
			}
			switch {
			case tc.wantWinner == nil && got.WinnerTeamID != nil:
				t.Errorf("winner = %v, want none", got.WinnerTeamID)
			case tc.wantWinner != nil && (got.WinnerTeamID == nil || *got.WinnerTeamID != *tc.wantWinner):
				t.Errorf("winner = %v, want %v", got.WinnerTeamID, *tc.wantWinner)
			}
		})
	}
}

func TestComputeCupsWon(t *testing.T) {
	cup2024, cup2025, cup2026 := uuid.New(), uuid.New(), uuid.New()
	redA, blueA := uuid.New(), uuid.New()
	redB, blueB := uuid.New(), uuid.New()
	redC, blueC := uuid.New(), uuid.New()
	alice, bob := uuid.New(), uuid.New()

	standings := map[uuid.UUID]TournamentStandings{
		// Alice's side won outright.
		cup2024: {TeamIDs: []uuid.UUID{redA, blueA}, Outcomes: []MatchOutcome{won(redA), halved()}},
		// Tied — nobody wins the Cup.
		cup2025: {TeamIDs: []uuid.UUID{redB, blueB}, Outcomes: []MatchOutcome{won(redB), won(blueB)}},
		// Still in progress.
		cup2026: {TeamIDs: []uuid.UUID{redC, blueC}, Outcomes: []MatchOutcome{won(redC), unplayed()}},
	}
	memberships := []TeamMembership{
		{PlayerID: alice, TournamentID: cup2024, TeamID: redA},
		{PlayerID: alice, TournamentID: cup2025, TeamID: redB},
		{PlayerID: alice, TournamentID: cup2026, TeamID: redC},
		{PlayerID: bob, TournamentID: cup2024, TeamID: blueA},
	}

	got := ComputeCupsWon(CupData{Standings: standings, Memberships: memberships})

	if got[alice] != 1 {
		t.Errorf("alice = %d cups, want 1 (a tie and an unfinished Cup count for nobody)", got[alice])
	}
	if got[bob] != 0 {
		t.Errorf("bob = %d cups, want 0 (he was on the losing side)", got[bob])
	}
}

func TestTournamentResultFor(t *testing.T) {
	red, blue := uuid.New(), uuid.New()
	teams := []uuid.UUID{red, blue}

	tests := []struct {
		name     string
		outcomes []MatchOutcome
		team     *uuid.UUID
		want     string
	}{
		{"won", []MatchOutcome{won(red)}, &red, sdk.ResultWon},
		{"lost", []MatchOutcome{won(red)}, &blue, sdk.ResultLost},
		{"tied", []MatchOutcome{won(red), won(blue)}, &red, sdk.ResultTied},
		{"in progress", []MatchOutcome{won(red), unplayed()}, &red, sdk.ResultInProgress},
		// Entered but never drafted: the Cup still has an outcome, but not one of theirs.
		{"undrafted player", []MatchOutcome{won(red)}, nil, sdk.ResultLost},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TournamentResultFor(TournamentStandings{TeamIDs: teams, Outcomes: tc.outcomes}, tc.team)
			if got != tc.want {
				t.Errorf("result = %q, want %q", got, tc.want)
			}
		})
	}
}
