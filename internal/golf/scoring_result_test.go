package golf

import "testing"

func TestComputeMatchResult_InProgressNotFinished(t *testing.T) {
	scores := []Score{
		{TeamID: 1, HoleNumber: 1, Strokes: 4},
		{TeamID: 2, HoleNumber: 1, Strokes: 5},
	}

	got := ComputeMatchResult(scores, 1, 2)

	if got.Finished {
		t.Errorf("want not finished, got %+v", got)
	}
}

func TestComputeMatchResult_ClosedOutHasWinner(t *testing.T) {
	// Team 1 wins every hole — decided at hole 10.
	var scores []Score
	for h := int32(1); h <= 18; h++ {
		scores = append(scores, Score{TeamID: 1, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: 2, HoleNumber: h, Strokes: 5})
	}

	got := ComputeMatchResult(scores, 1, 2)

	if !got.Finished || got.WinnerTeamID == nil || *got.WinnerTeamID != 1 {
		t.Errorf("want finished with winner 1, got %+v", got)
	}
}

func TestComputeMatchResult_AllSquareAfter18IsTie(t *testing.T) {
	// Every hole halved: 18 holes, all square, finished, no winner.
	var scores []Score
	for h := int32(1); h <= 18; h++ {
		scores = append(scores, Score{TeamID: 1, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: 2, HoleNumber: h, Strokes: 4})
	}

	got := ComputeMatchResult(scores, 1, 2)

	if !got.Finished || got.WinnerTeamID != nil {
		t.Errorf("want finished tie (no winner), got %+v", got)
	}
}

func TestComputeMatchResult_OneUpAfter18Wins(t *testing.T) {
	// Team 1 wins hole 1, halves 2-18: 1 up through 18, finished, winner 1.
	scores := []Score{
		{TeamID: 1, HoleNumber: 1, Strokes: 4},
		{TeamID: 2, HoleNumber: 1, Strokes: 5},
	}
	for h := int32(2); h <= 18; h++ {
		scores = append(scores, Score{TeamID: 1, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: 2, HoleNumber: h, Strokes: 4})
	}

	got := ComputeMatchResult(scores, 1, 2)

	if !got.Finished || got.WinnerTeamID == nil || *got.WinnerTeamID != 1 {
		t.Errorf("want finished with winner 1, got %+v", got)
	}
}
