package golf

import (
	"reflect"
	"testing"
)

func TestComputeStoredResult_NoScores(t *testing.T) {
	got := ComputeStoredResult(nil, teamA, teamB)
	if got.Finished || got.LeaderTeamID != nil || got.HolesRemaining != 18 {
		t.Errorf("want unfinished, no leader, 18 to play, got %+v", got)
	}
}

func TestComputeStoredResult_InProgress(t *testing.T) {
	scores := []Score{
		{TeamID: teamA, HoleNumber: 1, Strokes: 4},
		{TeamID: teamB, HoleNumber: 1, Strokes: 5},
	}

	got := ComputeStoredResult(scores, teamA, teamB)

	if got.Finished || got.LeaderTeamID == nil || *got.LeaderTeamID != teamA || got.Lead != 1 {
		t.Errorf("want unfinished, leader 1, lead 1, got %+v", got)
	}
}

func TestComputeStoredResult_ClosedOut(t *testing.T) {
	// Team 1 wins every hole — decided at hole 10 (lead 10, 8 to play).
	var scores []Score
	for h := int32(1); h <= 18; h++ {
		scores = append(scores, Score{TeamID: teamA, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: teamB, HoleNumber: h, Strokes: 5})
	}

	got := ComputeStoredResult(scores, teamA, teamB)

	if !got.Finished || *got.LeaderTeamID != teamA || got.Lead != 10 || got.HolesRemaining != 8 {
		t.Errorf("want finished, leader 1, lead 10, rem 8, got %+v", got)
	}
}

func TestComputeStoredResult_AllSquareAfter18(t *testing.T) {
	var scores []Score
	for h := int32(1); h <= 18; h++ {
		scores = append(scores, Score{TeamID: teamA, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: teamB, HoleNumber: h, Strokes: 4})
	}

	got := ComputeStoredResult(scores, teamA, teamB)

	if !got.Finished || got.LeaderTeamID != nil {
		t.Errorf("want finished tie (no leader), got %+v", got)
	}
}

func TestComputeStoredResult_OneUpAfter18(t *testing.T) {
	scores := []Score{
		{TeamID: teamA, HoleNumber: 1, Strokes: 4},
		{TeamID: teamB, HoleNumber: 1, Strokes: 5},
	}
	for h := int32(2); h <= 18; h++ {
		scores = append(scores, Score{TeamID: teamA, HoleNumber: h, Strokes: 4})
		scores = append(scores, Score{TeamID: teamB, HoleNumber: h, Strokes: 4})
	}

	got := ComputeStoredResult(scores, teamA, teamB)

	if !got.Finished || got.LeaderTeamID == nil || *got.LeaderTeamID != teamA || got.Lead != 1 {
		t.Errorf("want finished, leader 1, lead 1, got %+v", got)
	}
}

// The write publishes ComputeMatchState and the read publishes ComputeMatchProgress. If
// those ever disagree, a client caching the write's answer holds something the next read
// contradicts, so the agreement is asserted rather than left to the shared call.
func TestComputeMatchState_AgreesWithTheFunctionsItReplaces(t *testing.T) {
	scores := []Score{
		{TeamID: teamA, HoleNumber: 1, Strokes: 4},
		{TeamID: teamB, HoleNumber: 1, Strokes: 5},
		{TeamID: teamA, HoleNumber: 2, Strokes: 3},
		{TeamID: teamB, HoleNumber: 2, Strokes: 4},
	}

	got := ComputeMatchState(scores, teamA, teamB)

	if !reflect.DeepEqual(got.Holes, ComputeMatchProgress(scores, teamA, teamB)) {
		t.Errorf("holes differ from the read's series: %+v", got.Holes)
	}
	if !reflect.DeepEqual(got.StoredResult, ComputeStoredResult(scores, teamA, teamB)) {
		t.Errorf("result differs from the persisted one: %+v", got.StoredResult)
	}
}

func TestComputeMatchState_UnscoredMatchHasNoHoles(t *testing.T) {
	got := ComputeMatchState(nil, teamA, teamB)

	if len(got.Holes) != 0 {
		t.Errorf("want no holes, got %+v", got.Holes)
	}
	if got.HolesRemaining != 18 || got.Finished {
		t.Errorf("want a full match still to play, got %+v", got.StoredResult)
	}
}

// A client walking to "the next hole" off this series must not be sent past the end of the
// match, so the series has to stop where the match did.
func TestComputeMatchState_HolesStopAtTheCloseOut(t *testing.T) {
	var scores []Score
	for h := int32(1); h <= 12; h++ {
		scores = append(scores,
			Score{TeamID: teamA, HoleNumber: h, Strokes: 4},
			Score{TeamID: teamB, HoleNumber: h, Strokes: 5})
	}

	got := ComputeMatchState(scores, teamA, teamB)

	if len(got.Holes) != 10 {
		t.Errorf("want the series to end on the deciding hole, got %d holes", len(got.Holes))
	}
	if !got.Finished || got.Lead != 10 || got.HolesRemaining != 8 {
		t.Errorf("want 10 & 8, got %+v", got.StoredResult)
	}
}
