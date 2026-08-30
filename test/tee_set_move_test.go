package test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
)

// secondTeeSet adds another playable tee on the fixture's course, so a move has somewhere
// legitimate to go — an unknown id would be refused for a different reason entirely.
// A second tee on the fixture's own course, so a match has somewhere to be moved to.
func secondTeeSet(t *testing.T, client *sdk.Client, fix *util.Fixture) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	tc, err := client.CreateTeeColor(ctx, sdk.CreateTeeColorRequest{Color: "Gold-" + uuid.NewString()[:8]})
	if err != nil {
		t.Fatalf("tee colour: %v", err)
	}
	if _, err := client.AddTeeSet(ctx, fix.CourseID, sdk.CreateTeeSetRequest{
		TeeColorID: tc.ID, Slope: 113, Rating: 72.0, Holes: eighteenHoles(),
	}); err != nil {
		t.Fatalf("tee set: %v", err)
	}
	return tc.ID
}

func TestAnUnscoredMatchMovesToAnotherTeeSet(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	gold := secondTeeSet(t, client, fix)

	updated, err := client.UpdateMatch(context.Background(), fix.MatchID, sdk.UpdateMatchRequest{TeeColorID: &gold})
	if err != nil {
		t.Fatalf("want the move allowed, got %v", err)
	}
	if updated.TeeColorID != gold {
		t.Errorf("want the new tee set, got %s", updated.TeeColorID)
	}
}

// The database refuses this on its own, with a foreign key and an unexplained 400. The 409
// is what tells someone the reset that makes it possible.
func TestAScoredMatchMovesOnceItHasBeenReset(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	gold := secondTeeSet(t, client, fix)
	closeOutRedWin(t, client, fix)

	if err := client.ResetMatchScores(ctx, fix.MatchID); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{TeeColorID: &gold}); err != nil {
		t.Fatalf("want the reset match to move, got %v", err)
	}
}

// Everything else stays editable after a match has been played: the tee set is refused
// because scores read their par from it, which is not true of a tee time or a handicap.
func TestAScoredMatchStillTakesOtherEdits(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix)

	moved := "2026-09-18T15:30:00Z"
	off := false
	if _, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{TeeTime: &moved, Handicapped: &off}); err != nil {
		t.Fatalf("want a scored match's tee time still editable, got %v", err)
	}
}

// Re-sending what a match already has is not a move, so it must not be refused.
func TestResendingTheSameTeeSetIsNotAMove(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix)

	same := fix.TeeColorID
	if _, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{TeeColorID: &same}); err != nil {
		t.Fatalf("want a no-op accepted, got %v", err)
	}
}

// The refusal is only useful if the reason reaches whoever is reading it. Every route that
// refuses a scored match says which reset makes it possible, on the wire and not only in the log.
// Every edit that would reinterpret a scored match is refused, each naming the reset that
// makes it possible, and the refusals leave the scores where they were.
func TestARefusalSaysWhyAndWhatToDo(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	gold := secondTeeSet(t, client, fix)
	closeOutRedWin(t, client, fix)

	_, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{TeeColorID: &gold})
	saysScored(t, err, "moving the tee set", "changing its course or tees")
	saysScored(t, setTheSameLineup(t, client, fix), "setting the lineup", "changing its lineup")
	saysScored(t, client.DeleteMatch(ctx, fix.MatchID), "deleting", "deleting it")

	scores, err := client.GetMatchScores(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("get scores: %v", err)
	}
	if len(scores) == 0 {
		t.Error("want the scores untouched by the refusals, got none")
	}
}

// Every refusal a scored match makes names the reset that undoes it, so each is checked for
// the way through rather than only for the status.
func saysScored(t *testing.T, err error, what, wayThrough string) {
	t.Helper()
	msg := wantsStatus(t, err, http.StatusConflict)
	if !strings.HasPrefix(msg, "That match has scores.") || !strings.Contains(msg, wayThrough) {
		t.Errorf("%s: want a refusal naming the reset and %q, got %q", what, wayThrough, msg)
	}
}
