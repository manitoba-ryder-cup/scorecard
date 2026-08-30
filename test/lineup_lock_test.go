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

// Both routes that could take a player out of a scored match are refused.

func TestRemovingAParticipantFromAScoredMatchIsRefused(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	playHole(t, client, fix, 1, 4, 5)

	err := setTheSameLineup(t, client, fix)
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 APIError, got %v", err)
	}

	participants, err := client.ListParticipants(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participants) != 2 {
		t.Errorf("want both sides still in the match, got %d", len(participants))
	}
}

func TestUndraftingAScoredPlayerIsRefused(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	playHole(t, client, fix, 1, 4, 5)

	err := client.UndraftPlayer(ctx, fix.TeamRed, fix.RedPlayer)
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 APIError, got %v", err)
	}
	if apiErr.Message != "That player is participating in a match." {
		t.Errorf("message = %q", apiErr.Message)
	}
}

// The grain the scores foreign key cannot see: a one-ball format records against the team with
// no player, so those rows reference no participant and the guard is all that stops the lineup
// moving. The undraft beside it is refused by a constraint that does not depend on the format.
func TestAOneBallScoreAlsoLocksTheLineup(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()

	if _, err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
		HoleNumber: 1,
		Scores: []sdk.ScoreEntry{
			{TeamID: fix.TeamRed, PlayerID: nil, Strokes: 4},
			{TeamID: fix.TeamBlue, PlayerID: nil, Strokes: 5},
		},
	}); err != nil {
		t.Fatalf("submit a team score: %v", err)
	}

	var apiErr *sdk.APIError
	if err := setTheSameLineup(t, client, fix); !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Errorf("set lineup: want 409, got %v", err)
	}
	if err := client.UndraftPlayer(ctx, fix.TeamRed, fix.RedPlayer); !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Errorf("undraft: want 409, got %v", err)
	}
}

// Reset is the way through, which is what makes it more than a testing tool: clear the
// match, then the lineup is editable again.
func TestResetReopensAScoredMatchesLineup(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	playHole(t, client, fix, 1, 4, 5)

	if err := client.ResetMatchScores(ctx, fix.MatchID); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := setTheSameLineup(t, client, fix); err != nil {
		t.Fatalf("set lineup after reset: %v", err)
	}
}

// Reset clears the scores, not who was named to play, so it is not on its own a way to undraft
// anyone. Naming someone else in her place is.
func TestUndraftingWorksOnceThePlayerIsSubstitutedOut(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()

	if err := client.UndraftPlayer(ctx, fix.TeamRed, fix.RedPlayer); err == nil {
		t.Fatal("want the undraft refused while she holds a lineup place")
	}
	if err := client.SetLineup(ctx, fix.MatchID, theLineup(
		onSide(anotherDraftedPlayer(t, client, fix, fix.TeamRed), fix.TeamRed),
		onSide(fix.BluePlayer, fix.TeamBlue),
	)); err != nil {
		t.Fatalf("substitute her out: %v", err)
	}

	if err := client.UndraftPlayer(ctx, fix.TeamRed, fix.RedPlayer); err != nil {
		t.Fatalf("undraft once she is out of the lineup: %v", err)
	}
}

// The guard returns the clean 409; the foreign key is what makes the rule true. Asserted
// against the database directly, since a test through the API cannot tell the two apart.
//
// Per-player scores only: a one-ball format records against the team with no player, so the
// guard is all those rows have.
func TestTheDatabaseRefusesToOrphanAScoredParticipant(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	playHole(t, client, fix, 1, 4, 5)

	conn := util.ConnectAs(t, fix.TenantID)
	_, err := conn.Exec(ctx,
		"DELETE FROM match_participants WHERE match_id = $1 AND player_id = $2 AND tenant_id = $3",
		fix.MatchID, fix.RedPlayer, fix.TenantID)
	// Named, not merely non-nil: RLS refusing the row would read as the constraint working.
	if err == nil || !strings.Contains(err.Error(), "fk__scores__match_id_player_id__match_participants") {
		t.Fatalf("want the scores foreign key to refuse the delete, got %v", err)
	}
}

func theLineup(pairs ...sdk.LineupPlayer) sdk.SetLineupRequest {
	return sdk.SetLineupRequest{Participants: pairs}
}

func onSide(playerID, teamID uuid.UUID) sdk.LineupPlayer {
	return sdk.LineupPlayer{PlayerID: playerID, TeamID: teamID}
}

// The lineup the fixture already holds. Writing it back is still a write, and a scored match
// refuses it — the rule is about the lineup moving at all, not about it ending up different.
func setTheSameLineup(t *testing.T, client *sdk.Client, fix *util.Fixture) error {
	t.Helper()
	return client.SetLineup(context.Background(), fix.MatchID,
		theLineup(onSide(fix.RedPlayer, fix.TeamRed), onSide(fix.BluePlayer, fix.TeamBlue)))
}
