package test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
)

// closeOutRedWin submits scores so Red wins holes 1-10 outright (4 vs 5), closing the
// only match at 10 up with 8 to play — and, since it's the tournament's lone match,
// deciding the tournament for Red.
func closeOutRedWin(t *testing.T, client *sdk.Client, fix *util.Fixture) {
	t.Helper()
	ctx := context.Background()
	red, blue := fix.RedPlayer, fix.BluePlayer
	for h := int32(1); h <= 10; h++ {
		if err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
			HoleNumber: h, Strokes: 4, TeamID: fix.TeamRed, PlayerID: &red,
		}); err != nil {
			t.Fatalf("submit red hole %d: %v", h, err)
		}
		if err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
			HoleNumber: h, Strokes: 5, TeamID: fix.TeamBlue, PlayerID: &blue,
		}); err != nil {
			t.Fatalf("submit blue hole %d: %v", h, err)
		}
	}
}

func TestTournamentResultsReflectAClosedMatch(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix)

	results, err := client.GetTournamentResults(ctx, fix.TournamentID)
	if err != nil {
		t.Fatalf("get results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	r := results[0]
	if r.MatchID != fix.MatchID || r.FormatName != "Singles" || r.CourseName != "Test GC" {
		t.Fatalf("unexpected result identity: %+v", r)
	}
	if !r.Finished || r.WinnerTeamID == nil || *r.WinnerTeamID != fix.TeamRed {
		t.Fatalf("want finished, won by Red: %+v", r)
	}
	if r.Lead != 10 || r.HolesRemaining != 8 {
		t.Fatalf("want 10 & 8, got lead=%d remaining=%d", r.Lead, r.HolesRemaining)
	}
	if len(r.Sides) != 2 {
		t.Fatalf("want two sides, got %d", len(r.Sides))
	}

	// Each side is one player; Red's side holds the Red player with their name.
	for _, side := range r.Sides {
		if len(side.Players) != 1 {
			t.Fatalf("want one player per side, got %+v", side)
		}
		if side.TeamID == fix.TeamRed {
			if p := side.Players[0]; p.PlayerID != fix.RedPlayer || p.FirstName != "Red" || p.LastName != "Player" {
				t.Fatalf("unexpected red player: %+v", p)
			}
		}
	}

	// Ten played holes, every one won by Red.
	if len(r.HoleResults) != 10 {
		t.Fatalf("want 10 hole results, got %d", len(r.HoleResults))
	}
	for i, hr := range r.HoleResults {
		if hr == nil || *hr != fix.TeamRed {
			t.Fatalf("hole %d: want Red, got %v", i+1, hr)
		}
	}
}

func TestTournamentResultsForAnUnplayedMatch(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)

	results, err := client.GetTournamentResults(context.Background(), fix.TournamentID)
	if err != nil {
		t.Fatalf("get results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Finished || r.WinnerTeamID != nil || r.Lead != 0 || r.HolesRemaining != 18 {
		t.Fatalf("want an unplayed match, got %+v", r)
	}
	// Serialized as [], never null, so the client can iterate unconditionally.
	if r.HoleResults == nil {
		t.Fatal("want empty hole_results slice, got null")
	}
	if len(r.HoleResults) != 0 {
		t.Fatalf("want no hole results, got %d", len(r.HoleResults))
	}
}

func TestMatchHolesReturnTheTeeSet(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)

	holes, err := client.GetMatchHoles(context.Background(), fix.MatchID)
	if err != nil {
		t.Fatalf("get holes: %v", err)
	}
	if len(holes) != 18 {
		t.Fatalf("want 18 holes, got %d", len(holes))
	}
	for i, h := range holes {
		if h.Number != int32(i+1) {
			t.Fatalf("holes out of order at %d: %+v", i, h)
		}
		if h.Par != 4 || h.Yards != 400 {
			t.Fatalf("unexpected hole setup: %+v", h)
		}
	}
}

func TestMatchHolesForNonexistentMatchReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	_, err := client.GetMatchHoles(context.Background(), uuid.New())
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

func TestRosterCarriesRecordAndCups(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	closeOutRedWin(t, client, fix)

	roster, err := client.ListTournamentPlayers(context.Background(), fix.TournamentID)
	if err != nil {
		t.Fatalf("list roster: %v", err)
	}
	byPlayer := map[uuid.UUID]sdk.TournamentPlayer{}
	for _, p := range roster {
		byPlayer[p.PlayerID] = p
	}

	red := byPlayer[fix.RedPlayer]
	if red.Record.Wins != 1 || red.Record.Losses != 0 || red.Record.Ties != 0 {
		t.Errorf("red record want 1-0-0, got %+v", red.Record)
	}
	if red.CupsWon != 1 {
		t.Errorf("red cups_won want 1, got %d", red.CupsWon)
	}
	blue := byPlayer[fix.BluePlayer]
	if blue.Record.Losses != 1 || blue.Record.Wins != 0 {
		t.Errorf("blue record want 0-1-0, got %+v", blue.Record)
	}
	if blue.CupsWon != 0 {
		t.Errorf("blue cups_won want 0, got %d", blue.CupsWon)
	}
}

func TestPlayerTournamentHistory(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	closeOutRedWin(t, client, fix)

	history, err := client.GetPlayerTournaments(context.Background(), fix.RedPlayer)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("want 1 entry, got %d", len(history))
	}
	h := history[0]
	if h.TournamentID != fix.TournamentID || h.Name != "Test Cup" || h.Location != "Winnipeg" {
		t.Fatalf("unexpected tournament: %+v", h)
	}
	if h.Result != sdk.ResultWon {
		t.Fatalf("want result won, got %q", h.Result)
	}
	if h.Record.Wins != 1 || h.Record.Losses != 0 || h.Record.Ties != 0 {
		t.Fatalf("want record 1-0-0, got %+v", h.Record)
	}
	// The Red player captains their own side in the fixture.
	if h.CaptainLastName != "Player" {
		t.Fatalf("want captain Player, got %q", h.CaptainLastName)
	}
}

func TestPlayerTournamentHistoryInProgress(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)

	history, err := client.GetPlayerTournaments(context.Background(), fix.RedPlayer)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 || history[0].Result != sdk.ResultInProgress {
		t.Fatalf("want a single in_progress entry, got %+v", history)
	}
}
