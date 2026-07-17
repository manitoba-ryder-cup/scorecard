package golf

import (
	"reflect"
	"testing"
)

func TestComputeMatchProgress_LeaderWinsHole(t *testing.T) {
	// Team A posts 4, Team B posts 5 on hole 1 — lower wins, so Team A goes 1 up.
	scores := []Score{
		{TeamID: teamA, HoleNumber: 1, Strokes: 4},
		{TeamID: teamB, HoleNumber: 1, Strokes: 5},
	}

	got := ComputeMatchProgress(scores, teamA, teamB)

	want := []HoleResult{
		{
			HoleNumber:     1,
			TeamScores:     []TeamHoleScore{{TeamID: teamA, Strokes: 4}, {TeamID: teamB, Strokes: 5}},
			LeaderTeamID:   pUUID(teamA),
			Lead:           1,
			HolesRemaining: 17,
			Decided:        false,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestComputeMatchProgress_HalvedHoleIsAllSquare(t *testing.T) {
	scores := []Score{
		{TeamID: teamA, HoleNumber: 1, Strokes: 4},
		{TeamID: teamB, HoleNumber: 1, Strokes: 4},
	}

	got := ComputeMatchProgress(scores, teamA, teamB)

	if len(got) != 1 {
		t.Fatalf("got %d holes, want 1", len(got))
	}
	if got[0].Lead != 0 || got[0].LeaderTeamID != nil || got[0].Decided {
		t.Errorf("want lead 0, no leader, undecided, got %+v", got[0])
	}
}

func TestComputeMatchProgress_ClosesOutAndStops(t *testing.T) {
	// Team A wins every hole. It's decided at hole 10 (lead 10 with 8 to play),
	// so the progression stops there even though 18 holes were scored.
	var scores []Score
	for h := int32(1); h <= 18; h++ {
		scores = append(scores, Score{TeamID: teamA, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: teamB, HoleNumber: h, Strokes: 5})
	}

	got := ComputeMatchProgress(scores, teamA, teamB)

	if len(got) != 10 {
		t.Fatalf("want 10 holes (stops at close-out), got %d", len(got))
	}
	last := got[len(got)-1]
	if !last.Decided || last.Lead != 10 || last.HolesRemaining != 8 || *last.LeaderTeamID != teamA {
		t.Errorf("want decided, leader A, lead 10, rem 8, got %+v", last)
	}
}

func TestComputeMatchProgress_FourballUsesBestBall(t *testing.T) {
	// Two players per team; the team's hole score is the better (min) of the two.
	scores := []Score{
		{TeamID: teamA, PlayerID: pUUID(playerA), HoleNumber: 1, Strokes: 5},
		{TeamID: teamA, PlayerID: pUUID(playerA2), HoleNumber: 1, Strokes: 4}, // Team A best = 4
		{TeamID: teamB, PlayerID: pUUID(playerB), HoleNumber: 1, Strokes: 5},  // Team B best = 5
		{TeamID: teamB, PlayerID: pUUID(playerB2), HoleNumber: 1, Strokes: 6},
	}

	got := ComputeMatchProgress(scores, teamA, teamB)

	want := []TeamHoleScore{{TeamID: teamA, Strokes: 4}, {TeamID: teamB, Strokes: 5}}
	if len(got) != 1 || !reflect.DeepEqual(got[0].TeamScores, want) || got[0].Lead != 1 || *got[0].LeaderTeamID != teamA {
		t.Errorf("want teamA 4 teamB 5, leader A lead 1, got %+v", got[0])
	}
}

func TestComputeMatchProgress_OneBallTeamScore(t *testing.T) {
	// Alt shot / scramble: one score per team (player_id nil).
	scores := []Score{
		{TeamID: teamA, PlayerID: nil, HoleNumber: 1, Strokes: 4},
		{TeamID: teamB, PlayerID: nil, HoleNumber: 1, Strokes: 5},
	}

	got := ComputeMatchProgress(scores, teamA, teamB)

	if len(got) != 1 || got[0].Lead != 1 || *got[0].LeaderTeamID != teamA {
		t.Errorf("want lead 1 leader A, got %+v", got[0])
	}
}
