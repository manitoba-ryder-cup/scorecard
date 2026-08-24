package test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	testjwt "github.com/manitoba-ryder-cup/scorecard/test/_util/jwt"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

func TestDeleteMatchRemovesItAndItsLineup(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()

	participants, err := client.ListParticipants(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participants) == 0 {
		t.Fatal("want a lineup to delete, got none")
	}

	if err := client.DeleteMatch(ctx, fix.MatchID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	matches, err := client.ListMatches(ctx, fix.TournamentID)
	if err != nil {
		t.Fatalf("list matches: %v", err)
	}
	for _, m := range matches {
		if m.ID == fix.MatchID {
			t.Error("want the match off the tournament's schedule, found it there")
		}
	}
}

// An unscored participant is referenced by nothing, so the lineup goes with the match.
func TestDeleteMatchWithAnUnscoredLineupIsAllowed(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()

	if err := client.DeleteMatch(ctx, fix.MatchID); err != nil {
		t.Fatalf("want an unscored match with a lineup to delete, got %v", err)
	}

	participants, err := client.ListParticipants(ctx, fix.MatchID)
	if err == nil && len(participants) != 0 {
		t.Errorf("want the lineup gone with the match, got %d participants", len(participants))
	}
}

// Deleting a played match would take its scores and its stored result with it, which is a
// decision rather than a tidy-up. The 409 names the way through.
func TestDeleteAScoredMatchIsRefused(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix)

	err := client.DeleteMatch(ctx, fix.MatchID)
	if err == nil {
		t.Fatal("want a scored match refused, got a delete")
	}
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %v", err)
	}

	// Refused, and nothing taken on the way out.
	scores, err := client.GetMatchScores(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("get scores: %v", err)
	}
	if len(scores) == 0 {
		t.Error("want the scores untouched by the refusal, got none")
	}
}

// Reset then delete: the two-step the 409 points at.
func TestAScoredMatchDeletesOnceItHasBeenReset(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix)

	if err := client.ResetMatchScores(ctx, fix.MatchID); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := client.DeleteMatch(ctx, fix.MatchID); err != nil {
		t.Fatalf("want the reset match to delete, got %v", err)
	}
}

func TestDeleteUnknownMatchIsNotFound(t *testing.T) {
	t.Parallel()
	path := strings.Replace(sdk.RouteV1Match, "{id}", uuid.NewString(), 1)
	token := testjwt.MintAccessToken(t, uuid.New(), uuid.New(), sdk.ScopeTournamentsWrite)

	if status, body := request.Raw(t, http.MethodDelete, path, "", token); status != http.StatusNotFound {
		t.Fatalf("want 404, got %d (%s)", status, body)
	}
}

// Scoring is the grant handed to someone on the course; removing a match is not.
func TestDeleteMatchRefusesAScoresWriteToken(t *testing.T) {
	t.Parallel()
	path := strings.Replace(sdk.RouteV1Match, "{id}", uuid.NewString(), 1)
	token := testjwt.MintAccessToken(t, uuid.New(), uuid.New(), sdk.ScopeScoresWrite)

	if status, body := request.Raw(t, http.MethodDelete, path, "", token); status != http.StatusForbidden {
		t.Fatalf("want 403, got %d (%s)", status, body)
	}
}
