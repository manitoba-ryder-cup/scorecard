package golf

import (
	"reflect"
	"testing"
)

func pInt32(v int32) *int32 { return &v }

func TestComputeMatchProgress_RedWinsHole(t *testing.T) {
	// Red posts 4, Blue posts 5 on hole 1 — lower score wins, so Red goes 1 up.
	scores := []Score{
		{TeamID: 1, HoleNumber: 1, Strokes: 4},
		{TeamID: 2, HoleNumber: 1, Strokes: 5},
	}

	got := ComputeMatchProgress(scores, 1, 2)

	want := []HoleResult{
		{HoleNumber: 1, RedStrokes: 4, BlueStrokes: 5, Lead: 1, HolesRemaining: 17, Decided: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestComputeMatchProgress_HalvedHoleIsAllSquare(t *testing.T) {
	scores := []Score{
		{TeamID: 1, HoleNumber: 1, Strokes: 4},
		{TeamID: 2, HoleNumber: 1, Strokes: 4},
	}

	got := ComputeMatchProgress(scores, 1, 2)

	if len(got) != 1 {
		t.Fatalf("got %d holes, want 1", len(got))
	}
	if got[0].Lead != 0 || got[0].Decided {
		t.Errorf("want lead 0 and undecided, got %+v", got[0])
	}
}

func TestComputeMatchProgress_ClosesOutAndStops(t *testing.T) {
	// Red wins every hole. It's mathematically decided at hole 10 (lead 10 with
	// 8 to play), so the progression stops there even though 18 holes were scored.
	var scores []Score
	for h := int32(1); h <= 18; h++ {
		scores = append(scores, Score{TeamID: 1, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: 2, HoleNumber: h, Strokes: 5})
	}

	got := ComputeMatchProgress(scores, 1, 2)

	if len(got) != 10 {
		t.Fatalf("want 10 holes (stops at close-out), got %d", len(got))
	}
	last := got[len(got)-1]
	if !last.Decided || last.Lead != 10 || last.HolesRemaining != 8 {
		t.Errorf("want decided lead 10 rem 8, got %+v", last)
	}
}

func TestComputeMatchProgress_FourballUsesBestBall(t *testing.T) {
	// Two players per team; the team's hole score is the better (min) of the two.
	scores := []Score{
		{TeamID: 1, PlayerID: pInt32(10), HoleNumber: 1, Strokes: 5},
		{TeamID: 1, PlayerID: pInt32(11), HoleNumber: 1, Strokes: 4}, // Red best = 4
		{TeamID: 2, PlayerID: pInt32(20), HoleNumber: 1, Strokes: 5}, // Blue best = 5
		{TeamID: 2, PlayerID: pInt32(21), HoleNumber: 1, Strokes: 6},
	}

	got := ComputeMatchProgress(scores, 1, 2)

	if len(got) != 1 || got[0].RedStrokes != 4 || got[0].BlueStrokes != 5 || got[0].Lead != 1 {
		t.Errorf("want red 4 blue 5 lead 1, got %+v", got[0])
	}
}

func TestComputeMatchProgress_OneBallTeamScore(t *testing.T) {
	// Alt shot / scramble: one score per team (player_id nil).
	scores := []Score{
		{TeamID: 1, PlayerID: nil, HoleNumber: 1, Strokes: 4},
		{TeamID: 2, PlayerID: nil, HoleNumber: 1, Strokes: 5},
	}

	got := ComputeMatchProgress(scores, 1, 2)

	if len(got) != 1 || got[0].RedStrokes != 4 || got[0].Lead != 1 {
		t.Errorf("want red 4 lead 1, got %+v", got[0])
	}
}
