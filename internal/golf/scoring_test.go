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

	want := []TeamHoleScore{
		{TeamID: teamA, Strokes: 4, PlayerScores: []PlayerHoleScore{{PlayerID: playerA, Strokes: 5}, {PlayerID: playerA2, Strokes: 4}}},
		{TeamID: teamB, Strokes: 5, PlayerScores: []PlayerHoleScore{{PlayerID: playerB, Strokes: 5}, {PlayerID: playerB2, Strokes: 6}}},
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0].TeamScores, want) || got[0].Lead != 1 || *got[0].LeaderTeamID != teamA {
		t.Errorf("want teamA 4 teamB 5, leader A lead 1, got %+v", got[0])
	}
}

func TestComputeMatchProgress_PlayerScoresAreOrderedByPlayerID(t *testing.T) {
	// Scores come back from the store ordered by hole only, so the per-player breakdown
	// is sorted by player id to keep the response stable between reads.
	scores := []Score{
		{TeamID: teamA, PlayerID: pUUID(playerA2), HoleNumber: 1, Strokes: 4},
		{TeamID: teamA, PlayerID: pUUID(playerA), HoleNumber: 1, Strokes: 5},
		{TeamID: teamB, PlayerID: pUUID(playerB), HoleNumber: 1, Strokes: 6},
	}

	got := ComputeMatchProgress(scores, teamA, teamB)

	want := []PlayerHoleScore{{PlayerID: playerA, Strokes: 5}, {PlayerID: playerA2, Strokes: 4}}
	if len(got) != 1 || !reflect.DeepEqual(got[0].TeamScores[0].PlayerScores, want) {
		t.Errorf("want player scores in id order %+v, got %+v", want, got[0].TeamScores[0].PlayerScores)
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
	// No player recorded the score, so there is no breakdown to report — the team score
	// is the whole truth for a one-ball format.
	if got[0].TeamScores[0].PlayerScores != nil {
		t.Errorf("want no player breakdown for a one-ball format, got %+v", got[0].TeamScores[0].PlayerScores)
	}
}

func TestComputeMatchProgress_SinglesPlayerScoreMatchesTeamScore(t *testing.T) {
	// One player per side: the breakdown names the player behind the team's score, which
	// is what lets a client re-open a scored hole with the right value per player.
	scores := []Score{
		{TeamID: teamA, PlayerID: pUUID(playerA), HoleNumber: 1, Strokes: 4},
		{TeamID: teamB, PlayerID: pUUID(playerB), HoleNumber: 1, Strokes: 5},
	}

	got := ComputeMatchProgress(scores, teamA, teamB)

	want := []TeamHoleScore{
		{TeamID: teamA, Strokes: 4, PlayerScores: []PlayerHoleScore{{PlayerID: playerA, Strokes: 4}}},
		{TeamID: teamB, Strokes: 5, PlayerScores: []PlayerHoleScore{{PlayerID: playerB, Strokes: 5}}},
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0].TeamScores, want) {
		t.Errorf("got %+v, want %+v", got[0].TeamScores, want)
	}
}

func TestComputeMatchProgress_LeadChangesHands(t *testing.T) {
	// A wins hole 1, then B wins holes 2 and 3: the lead crosses zero and the leader
	// flips from A to B — a swing the monotonic tests never exercise.
	scores := []Score{
		{TeamID: teamA, HoleNumber: 1, Strokes: 4}, {TeamID: teamB, HoleNumber: 1, Strokes: 5}, // A 1 up
		{TeamID: teamA, HoleNumber: 2, Strokes: 5}, {TeamID: teamB, HoleNumber: 2, Strokes: 4}, // all square
		{TeamID: teamA, HoleNumber: 3, Strokes: 5}, {TeamID: teamB, HoleNumber: 3, Strokes: 4}, // B 1 up
	}

	got := ComputeMatchProgress(scores, teamA, teamB)

	if len(got) != 3 {
		t.Fatalf("want 3 holes, got %d", len(got))
	}
	if got[0].LeaderTeamID == nil || *got[0].LeaderTeamID != teamA || got[0].Lead != 1 {
		t.Errorf("hole 1: want A 1 up, got %+v", got[0])
	}
	if got[1].LeaderTeamID != nil || got[1].Lead != 0 {
		t.Errorf("hole 2: want all square, got %+v", got[1])
	}
	if got[2].LeaderTeamID == nil || *got[2].LeaderTeamID != teamB || got[2].Lead != 1 {
		t.Errorf("hole 3: want B 1 up, got %+v", got[2])
	}
}

func TestComputeMatchProgress_DormieThenClosesOut(t *testing.T) {
	// Halve holes 1-14, then A wins 15, 16, 17. After 16, A is 2 up with 2 to play —
	// dormie (lead == holes remaining) is NOT decided; the opponent can still halve. A
	// wins 17 to go 3 up with 1 to play, which closes it out ("3 & 1"). This exercises
	// the exact close-out boundary (decided the moment lead first exceeds remaining),
	// unlike the existing test's wide 10 & 8 margin.
	var scores []Score
	for h := int32(1); h <= 14; h++ {
		scores = append(scores, Score{TeamID: teamA, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: teamB, HoleNumber: h, Strokes: 4})
	}
	for h := int32(15); h <= 17; h++ {
		scores = append(scores, Score{TeamID: teamA, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: teamB, HoleNumber: h, Strokes: 5})
	}

	got := ComputeMatchProgress(scores, teamA, teamB)

	if len(got) != 17 {
		t.Fatalf("want 17 holes (stops at close-out), got %d", len(got))
	}
	dormie := got[15] // hole 16
	if dormie.Decided || dormie.Lead != 2 || dormie.HolesRemaining != 2 || *dormie.LeaderTeamID != teamA {
		t.Errorf("hole 16: want dormie (A 2 up, 2 to play, undecided), got %+v", dormie)
	}
	last := got[16] // hole 17
	if !last.Decided || last.Lead != 3 || last.HolesRemaining != 1 || *last.LeaderTeamID != teamA {
		t.Errorf("hole 17: want decided A 3 up, 1 to play, got %+v", last)
	}
}

func TestComputeMatchProgress_EighteenHolesIsDecided(t *testing.T) {
	// Halve holes 1-17, then A wins 18. The lead never exceeds the holes remaining, so
	// nothing closes out early — but the match is over once the 18th is played.
	var scores []Score
	for h := int32(1); h <= 17; h++ {
		scores = append(scores, Score{TeamID: teamA, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: teamB, HoleNumber: h, Strokes: 4})
	}
	scores = append(scores, Score{TeamID: teamA, HoleNumber: 18, Strokes: 4})
	scores = append(scores, Score{TeamID: teamB, HoleNumber: 18, Strokes: 5})

	got := ComputeMatchProgress(scores, teamA, teamB)

	last := got[len(got)-1]
	if !last.Decided {
		t.Errorf("hole 18: want decided (every hole played), got %+v", last)
	}
}

func TestComputeMatchProgress_HalvedAfterEighteenIsDecided(t *testing.T) {
	// All square after 18: the match is halved, which is still over.
	var scores []Score
	for h := int32(1); h <= 18; h++ {
		scores = append(scores, Score{TeamID: teamA, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: teamB, HoleNumber: h, Strokes: 4})
	}

	got := ComputeMatchProgress(scores, teamA, teamB)

	last := got[len(got)-1]
	if !last.Decided || last.LeaderTeamID != nil || last.Lead != 0 {
		t.Errorf("hole 18: want decided and all square, got %+v", last)
	}
}
