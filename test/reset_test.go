package test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	testjwt "github.com/manitoba-ryder-cup/scorecard/test/_util/jwt"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

func TestResetReturnsAPlayedMatchToNeverPlayed(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix)

	if err := client.ResetMatchScores(ctx, fix.MatchID); err != nil {
		t.Fatalf("reset: %v", err)
	}

	scores, err := client.GetMatchScores(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("get scores: %v", err)
	}
	if len(scores) != 0 {
		t.Errorf("want no scores left, got %d holes", len(scores))
	}

	results, err := client.GetTournamentResults(ctx, fix.TournamentID)
	if err != nil {
		t.Fatalf("get results: %v", err)
	}
	r := results[0]
	if r.Finished || r.WinnerTeamID != nil || len(r.HoleResults) != 0 || r.HolesRemaining != 18 {
		t.Errorf("want an unplayed match, got %+v", r)
	}

	// A result row left behind would mark the match started and pin the cup live.
	tour, err := client.GetTournament(ctx, fix.TournamentID)
	if err != nil {
		t.Fatalf("get tournament: %v", err)
	}
	if tour.Phase != sdk.PhaseUpcoming {
		t.Errorf("phase = %q, want upcoming once nothing has been played", tour.Phase)
	}
}

func TestResetLeavesTheLineupInPlace(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix)

	if err := client.ResetMatchScores(ctx, fix.MatchID); err != nil {
		t.Fatalf("reset: %v", err)
	}

	participants, err := client.ListParticipants(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participants) != 2 {
		t.Errorf("want both sides still in the match, got %d", len(participants))
	}
}

func TestResetIsIdempotent(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()

	for i := range 2 {
		if err := client.ResetMatchScores(ctx, fix.MatchID); err != nil {
			t.Fatalf("reset %d of an unscored match: %v", i+1, err)
		}
	}
}

func TestResetUnknownMatchIsNotFound(t *testing.T) {
	t.Parallel()
	path := strings.Replace(sdk.RouteV1MatchScores, "{id}", uuid.NewString(), 1)
	token := testjwt.MintAccessToken(t, uuid.New(), uuid.New(), sdk.ScopeTournamentsWrite)

	if status, body := request.Raw(t, http.MethodDelete, path, "", token); status != http.StatusNotFound {
		t.Fatalf("want 404, got %d (%s)", status, body)
	}
}

func TestResetWorksAfterTheScoringWindowHasShut(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix)

	lastYear := time.Now().AddDate(-1, 0, 0).Format(time.RFC3339)
	if _, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{TeeTime: &lastYear}); err != nil {
		t.Fatalf("move the tee time: %v", err)
	}
	// Without this the reset below would pass whether or not the window is enforced.
	if _, err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
		HoleNumber: 11,
		Scores: []sdk.ScoreEntry{
			{TeamID: fix.TeamRed, PlayerID: &fix.RedPlayer, Strokes: 4},
			{TeamID: fix.TeamBlue, PlayerID: &fix.BluePlayer, Strokes: 5},
		},
	}); err == nil {
		t.Fatal("want the closed window to refuse a score, got none")
	}

	if err := client.ResetMatchScores(ctx, fix.MatchID); err != nil {
		t.Fatalf("reset outside the window: %v", err)
	}
	scores, err := client.GetMatchScores(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("get scores: %v", err)
	}
	if len(scores) != 0 {
		t.Errorf("want no scores left, got %d holes", len(scores))
	}
}
