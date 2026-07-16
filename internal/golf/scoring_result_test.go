package golf

import "testing"

func TestComputeStoredResult_NoScores(t *testing.T) {
	got := ComputeStoredResult(nil, 1, 2)
	if got.Finished || got.LeaderTeamID != nil || got.HolesRemaining != 18 {
		t.Errorf("want unfinished, no leader, 18 to play, got %+v", got)
	}
}

func TestComputeStoredResult_InProgress(t *testing.T) {
	scores := []Score{
		{TeamID: 1, HoleNumber: 1, Strokes: 4},
		{TeamID: 2, HoleNumber: 1, Strokes: 5},
	}

	got := ComputeStoredResult(scores, 1, 2)

	if got.Finished || got.LeaderTeamID == nil || *got.LeaderTeamID != 1 || got.Lead != 1 {
		t.Errorf("want unfinished, leader 1, lead 1, got %+v", got)
	}
}

func TestComputeStoredResult_ClosedOut(t *testing.T) {
	// Team 1 wins every hole — decided at hole 10 (lead 10, 8 to play).
	var scores []Score
	for h := int32(1); h <= 18; h++ {
		scores = append(scores, Score{TeamID: 1, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: 2, HoleNumber: h, Strokes: 5})
	}

	got := ComputeStoredResult(scores, 1, 2)

	if !got.Finished || *got.LeaderTeamID != 1 || got.Lead != 10 || got.HolesRemaining != 8 {
		t.Errorf("want finished, leader 1, lead 10, rem 8, got %+v", got)
	}
}

func TestComputeStoredResult_AllSquareAfter18(t *testing.T) {
	var scores []Score
	for h := int32(1); h <= 18; h++ {
		scores = append(scores, Score{TeamID: 1, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: 2, HoleNumber: h, Strokes: 4})
	}

	got := ComputeStoredResult(scores, 1, 2)

	if !got.Finished || got.LeaderTeamID != nil {
		t.Errorf("want finished tie (no leader), got %+v", got)
	}
}

func TestComputeStoredResult_OneUpAfter18(t *testing.T) {
	scores := []Score{
		{TeamID: 1, HoleNumber: 1, Strokes: 4},
		{TeamID: 2, HoleNumber: 1, Strokes: 5},
	}
	for h := int32(2); h <= 18; h++ {
		scores = append(scores, Score{TeamID: 1, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: 2, HoleNumber: h, Strokes: 4})
	}

	got := ComputeStoredResult(scores, 1, 2)

	if !got.Finished || got.LeaderTeamID == nil || *got.LeaderTeamID != 1 || got.Lead != 1 {
		t.Errorf("want finished, leader 1, lead 1, got %+v", got)
	}
}
