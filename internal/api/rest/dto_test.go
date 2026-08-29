package rest

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

func TestToPlayerProfileDTO_CombinesPlayerAndRecord(t *testing.T) {
	id := uuid.New()
	p := golf.Player{
		ID: id, FirstName: "Dustin", LastName: "Johnson", PhotoPath: "dj.jpg",
		Record: golf.PlayerRecord{Wins: 3, Losses: 1, Ties: 2},
	}

	got := toPlayerProfileDTO(p)

	// Base player fields are promoted from the embedded Player.
	if got.ID != id || got.FirstName != "Dustin" || got.LastName != "Johnson" || got.PhotoPath != "dj.jpg" {
		t.Errorf("player fields not mapped: %+v", got)
	}
	if got.Record.Wins != 3 || got.Record.Losses != 1 || got.Record.Ties != 2 {
		t.Errorf("record not mapped: %+v", got.Record)
	}
}

func TestToMatchStatusDTO_NamesTheWinnerOnlyOnceFinished(t *testing.T) {
	leader := uuid.New()

	live := toMatchStatusDTO(golf.StoredResult{Finished: false, LeaderTeamID: &leader, Lead: 2, HolesRemaining: 4})
	if live.WinnerTeamID != nil {
		t.Errorf("a leader with holes left has not won: %+v", live)
	}
	if live.LeaderTeamID == nil || *live.LeaderTeamID != leader || live.Lead != 2 || live.HolesRemaining != 4 {
		t.Errorf("live state not mapped: %+v", live)
	}

	done := toMatchStatusDTO(golf.StoredResult{Finished: true, LeaderTeamID: &leader, Lead: 3, HolesRemaining: 2})
	if done.WinnerTeamID == nil || *done.WinnerTeamID != leader {
		t.Errorf("want the leader as winner once finished, got %+v", done)
	}
}

func TestToMatchStatusDTO_HalvedMatchHasNoWinner(t *testing.T) {
	done := toMatchStatusDTO(golf.StoredResult{Finished: true, LeaderTeamID: nil, Lead: 0})

	if done.WinnerTeamID != nil {
		t.Errorf("an all-square finish has no winner, got %+v", done)
	}
}

func TestToHoleStatusDTO_CarriesPlayerBreakdown(t *testing.T) {
	teamID, p1, p2 := uuid.New(), uuid.New(), uuid.New()
	h := golf.HoleResult{
		HoleNumber: 7,
		TeamScores: []golf.TeamHoleScore{{
			TeamID:  teamID,
			Strokes: 4,
			PlayerScores: []golf.PlayerHoleScore{
				{PlayerID: p1, Strokes: 4},
				{PlayerID: p2, Strokes: 6},
			},
		}},
	}

	got := toHoleStatusDTO(h)

	ts := got.TeamScores[0]
	if ts.TeamID != teamID || ts.Strokes != 4 {
		t.Errorf("team score not mapped: %+v", ts)
	}
	// The side's best ball alone cannot tell a client what the other player shot.
	if len(ts.PlayerScores) != 2 ||
		ts.PlayerScores[0] != (sdk.PlayerHoleScore{PlayerID: p1, Strokes: 4}) ||
		ts.PlayerScores[1] != (sdk.PlayerHoleScore{PlayerID: p2, Strokes: 6}) {
		t.Errorf("player scores not mapped: %+v", ts.PlayerScores)
	}
}

func TestToHoleStatusDTO_OneBallHoleHasNoPlayerScores(t *testing.T) {
	// A team-grain score has nothing to break down, and serialises [] so clients can iterate.
	h := golf.HoleResult{HoleNumber: 1, TeamScores: []golf.TeamHoleScore{{TeamID: uuid.New(), Strokes: 5}}}

	got := toHoleStatusDTO(h)

	if got.TeamScores[0].PlayerScores == nil || len(got.TeamScores[0].PlayerScores) != 0 {
		t.Errorf("want empty non-nil player scores, got %+v", got.TeamScores[0].PlayerScores)
	}
}

// The status fields stay flat in the JSON, which is what lets a client written against the
// old response keep decoding this one. Asserting the Go shape would not catch a tag that
// nested them, so this reads the encoded bytes.
func TestToScoreSubmissionResultDTO_KeepsTheStatusFlatAndAddsHoles(t *testing.T) {
	leader := uuid.New()
	state := golf.MatchState{
		StoredResult: golf.StoredResult{Finished: true, LeaderTeamID: &leader, Lead: 3, HolesRemaining: 2},
		Holes:        []golf.HoleResult{{HoleNumber: 16, Lead: 3, HolesRemaining: 2, Decided: true}},
	}

	raw, err := json.Marshal(toScoreSubmissionResultDTO(state))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"finished", "winner_team_id", "leader_team_id", "lead", "holes_remaining"} {
		if _, ok := got[key]; !ok {
			t.Errorf("want %q at the top level, got %s", key, raw)
		}
	}
	holes, ok := got["holes"].([]any)
	if !ok || len(holes) != 1 {
		t.Fatalf("want one hole under holes, got %s", raw)
	}
}

func TestToScoreSubmissionResultDTO_CarriesThePerHoleOutcome(t *testing.T) {
	red, blue := uuid.New(), uuid.New()
	state := golf.MatchState{HoleResults: []*uuid.UUID{&red, nil, &blue}}

	got := toScoreSubmissionResultDTO(state)

	if len(got.HoleResults) != 3 || got.HoleResults[0] != &red || got.HoleResults[1] != nil || got.HoleResults[2] != &blue {
		t.Errorf("want the domain's outcome carried through, got %v", got.HoleResults)
	}
}

// Empty rather than null for both lists: their length is how far a match has got, and a client
// would otherwise have to tell an absent field from a match nobody has started.
func TestToScoreSubmissionResultDTO_EmptyListsAreNotNull(t *testing.T) {
	raw, err := json.Marshal(toScoreSubmissionResultDTO(golf.MatchState{}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"holes":[]`, `"hole_results":[]`} {
		if !bytes.Contains(raw, []byte(want)) {
			t.Errorf("want %s, got %s", want, raw)
		}
	}
}
