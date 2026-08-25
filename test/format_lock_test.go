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

// theMatch reads the fixture's match back off the tournament listing; the SDK has no
// single-match read.
func theMatch(t *testing.T, client *sdk.Client, fix *util.Fixture) sdk.Match {
	t.Helper()
	matches, err := client.ListMatches(context.Background(), fix.TournamentID)
	if err != nil {
		t.Fatalf("list matches: %v", err)
	}
	for _, m := range matches {
		if m.ID == fix.MatchID {
			return m
		}
	}
	t.Fatalf("match %s not in the tournament listing", fix.MatchID)
	return sdk.Match{}
}

// otherFormat picks a format the fixture's match is not already playing, so a change has
// somewhere legitimate to go — an unknown id would be refused for a different reason.
func otherFormat(t *testing.T, client *sdk.Client, fix *util.Fixture) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	formats, err := client.ListMatchFormats(ctx)
	if err != nil {
		t.Fatalf("list formats: %v", err)
	}
	playing := theMatch(t, client, fix).MatchFormatID
	for _, f := range formats {
		if f.ID != playing {
			return f.ID
		}
	}
	t.Fatal("the fixture has only one match format, so a change cannot be tested")
	return uuid.Nil
}

func TestAnUnscoredMatchChangesFormat(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	target := otherFormat(t, client, fix)

	updated, err := client.UpdateMatch(context.Background(), fix.MatchID, sdk.UpdateMatchRequest{MatchFormatID: &target})
	if err != nil {
		t.Fatalf("want the change allowed, got %v", err)
	}
	if updated.MatchFormatID != target {
		t.Errorf("want the new format, got %s", updated.MatchFormatID)
	}
}

// The refusal has to reach the caller as a sentence and not only a status: the reset is the
// only way through, and nothing else tells them.
func TestChangingAScoredMatchsFormatIsRefused(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	target := otherFormat(t, client, fix)
	closeOutRedWin(t, client, fix)

	_, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{MatchFormatID: &target})

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %v", err)
	}
	if apiErr.Message != "That match has scores. Reset it before changing its course, tees or format." {
		t.Errorf("message = %q", apiErr.Message)
	}

	if theMatch(t, client, fix).MatchFormatID == target {
		t.Error("the format changed anyway")
	}
}

// The refusal names a reset, so the reset has to actually make the change possible.
func TestAScoredMatchChangesFormatOnceItHasBeenReset(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	target := otherFormat(t, client, fix)
	closeOutRedWin(t, client, fix)

	if err := client.ResetMatchScores(ctx, fix.MatchID); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{MatchFormatID: &target}); err != nil {
		t.Fatalf("want the reset match to change format, got %v", err)
	}
}
