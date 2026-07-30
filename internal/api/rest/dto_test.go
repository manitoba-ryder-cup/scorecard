package rest

import (
	"testing"
	"time"

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
	// The individual scores must survive the mapping — the side's best ball alone can't
	// tell a client what the other player shot.
	if len(ts.PlayerScores) != 2 ||
		ts.PlayerScores[0] != (sdk.PlayerHoleScore{PlayerID: p1, Strokes: 4}) ||
		ts.PlayerScores[1] != (sdk.PlayerHoleScore{PlayerID: p2, Strokes: 6}) {
		t.Errorf("player scores not mapped: %+v", ts.PlayerScores)
	}
}

func TestToHoleStatusDTO_OneBallHoleHasNoPlayerScores(t *testing.T) {
	// Alt shot / scramble: the score belongs to the team, so there is nothing to break
	// down. It serialises as [] rather than null so clients can iterate unconditionally.
	h := golf.HoleResult{HoleNumber: 1, TeamScores: []golf.TeamHoleScore{{TeamID: uuid.New(), Strokes: 5}}}

	got := toHoleStatusDTO(h)

	if got.TeamScores[0].PlayerScores == nil || len(got.TeamScores[0].PlayerScores) != 0 {
		t.Errorf("want empty non-nil player scores, got %+v", got.TeamScores[0].PlayerScores)
	}
}

func TestToMatchResultDTO_ServesBothTheInstantAndTheCourseWallClock(t *testing.T) {
	teeTime := time.Date(2026, 9, 18, 13, 0, 0, 0, time.UTC)
	r := golf.MatchResult{
		MatchID:        uuid.New(),
		FormatName:     "Singles",
		CourseName:     "Pine Ridge",
		CourseTimeZone: "America/Winnipeg",
		TeeTime:        teeTime,
	}

	got := toMatchResultDTO(r)

	// The instant is what every spectator view renders, in the viewer's own zone.
	if got.TeeTime != "2026-09-18T13:00:00Z" {
		t.Errorf("tee_time = %q, want the RFC3339 instant", got.TeeTime)
	}
	// The wall clock is what an admin edits against, so it is the course's clock and
	// carries no zone — exactly what a datetime-local input holds.
	if got.TeeTimeLocal != "2026-09-18T08:00" {
		t.Errorf("tee_time_local = %q, want 2026-09-18T08:00", got.TeeTimeLocal)
	}
}

// A course with no stored zone must not fall through to UTC, which would show a tee time
// shifted by the course's offset — 13:00 for a round that goes out at 08:00.
func TestToMatchResultDTO_FallsBackToTheCupsZone(t *testing.T) {
	r := golf.MatchResult{
		MatchID: uuid.New(),
		TeeTime: time.Date(2026, 9, 18, 13, 0, 0, 0, time.UTC),
	}

	if got := toMatchResultDTO(r); got.TeeTimeLocal != "2026-09-18T08:00" {
		t.Errorf("tee_time_local = %q, want the cup's zone applied", got.TeeTimeLocal)
	}
}
