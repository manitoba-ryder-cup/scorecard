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

// A scorer corrects a hole by entering it again; there is no way to unsay one, and there
// should not be. An admin unwinding a match is the other case, and these cover it.

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

	// The stored result is deleted rather than zeroed, so the cup can fall back to
	// upcoming — a row left behind would mark the match started and pin the cup live.
	tour, err := client.GetTournament(ctx, fix.TournamentID)
	if err != nil {
		t.Fatalf("get tournament: %v", err)
	}
	if tour.Phase != sdk.PhaseUpcoming {
		t.Errorf("phase = %q, want upcoming once nothing has been played", tour.Phase)
	}
}

// The lineup is what makes a reset worth having over rebuilding the match.
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

// Nothing to clear is not an error. Rows affected cannot tell a match that was never
// scored from one that does not exist, which is why the handler asks the match instead.
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

// The case the feature exists for: a mistake found after the scoring window has shut. The
// match is scored while it is open, then moved into the past — which score entry refuses
// from that point on, and a reset must not.
func TestResetWorksAfterTheScoringWindowHasShut(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix)

	lastYear := time.Now().AddDate(-1, 0, 0).Format(time.RFC3339)
	if _, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{TeeTime: &lastYear}); err != nil {
		t.Fatalf("move the tee time: %v", err)
	}
	// Proves the window really is shut, so the reset below is not passing by accident.
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
