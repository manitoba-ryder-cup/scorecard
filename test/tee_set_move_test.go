package test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
)

// secondTeeSet adds another playable tee on the fixture's course, so a move has somewhere
// legitimate to go — an unknown id would be refused for a different reason entirely.
func secondTeeSet(t *testing.T, fix *util.Fixture) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	conn := util.ConnectAs(t, fix.TenantID)

	var id uuid.UUID
	if err := conn.QueryRow(ctx, `INSERT INTO tee_colors (tenant_id, color) VALUES ($1, $2) RETURNING id`,
		fix.TenantID, "Gold-"+uuid.NewString()[:8]).Scan(&id); err != nil {
		t.Fatalf("tee colour: %v", err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO tee_sets (course_id, tee_color_id, tenant_id, slope, rating) VALUES ($1,$2,$3,113,72.0)`,
		fix.CourseID, id, fix.TenantID); err != nil {
		t.Fatalf("tee set: %v", err)
	}
	for n := 1; n <= 18; n++ {
		if _, err := conn.Exec(ctx, `INSERT INTO holes (course_id, tee_color_id, number, tenant_id, par, hdcp, yards) VALUES ($1,$2,$3,$4,4,$3,400)`,
			fix.CourseID, id, n, fix.TenantID); err != nil {
			t.Fatalf("holes: %v", err)
		}
	}
	return id
}

func TestAnUnscoredMatchMovesToAnotherTeeSet(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	gold := secondTeeSet(t, fix)

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
func TestMovingAScoredMatchIsRefusedWithTheWayThrough(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	gold := secondTeeSet(t, fix)
	closeOutRedWin(t, client, fix)

	_, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{TeeColorID: &gold})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %v", err)
	}

	scores, err := client.GetMatchScores(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("get scores: %v", err)
	}
	if len(scores) == 0 {
		t.Error("want the scores untouched by the refusal, got none")
	}
}

func TestAScoredMatchMovesOnceItHasBeenReset(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	gold := secondTeeSet(t, fix)
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
func TestARefusalSaysWhyAndWhatToDo(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	gold := secondTeeSet(t, fix)
	closeOutRedWin(t, client, fix)

	_, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{TeeColorID: &gold})
	saysScored(t, err, "moving the tee set", "changing its course or tees")
	saysScored(t, setTheSameLineup(t, client, fix), "setting the lineup", "changing its lineup")
	saysScored(t, client.DeleteMatch(ctx, fix.MatchID), "deleting", "deleting it")
}

// Every refusal a scored match makes names the reset that undoes it, so each is checked for
// the way through rather than only for the status.
func saysScored(t *testing.T, err error, what, wayThrough string) {
	t.Helper()
	msg := wantsConflict(t, err, what)
	if !strings.HasPrefix(msg, "That match has scores.") || !strings.Contains(msg, wayThrough) {
		t.Errorf("%s: want a refusal naming the reset and %q, got %q", what, wayThrough, msg)
	}
}
