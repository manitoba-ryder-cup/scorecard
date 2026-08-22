package test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
)

// A scored match's lineup is what its scores are attributed to. Both routes that could
// take a player out of one are refused, because the delete reaches the scores and nothing
// recomputes the stored result they leave behind — the same cup then reads finished from
// one endpoint and never-played from another.

func TestRemovingAParticipantFromAScoredMatchIsRefused(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	playHole(t, client, fix, 1, 4, 5)

	err := client.RemoveParticipant(ctx, fix.MatchID, fix.RedPlayer)
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
	if err := client.RemoveParticipant(ctx, fix.MatchID, fix.RedPlayer); err != nil {
		t.Fatalf("remove after reset: %v", err)
	}
	if err := client.UndraftPlayer(ctx, fix.TeamRed, fix.RedPlayer); err != nil {
		t.Fatalf("undraft after reset: %v", err)
	}
}

// The service guard is what returns a clean 409, but it is not what makes the rule true:
// the foreign key is. Asserted against the database directly, because a test going through
// the API cannot tell the two apart — exactly why test/isolation exists for RLS.
//
// It covers per-player scores only. A one-ball format records against the team with no
// player, so those rows do not reference the participant at all and the guard above is the
// only thing standing between them and a cascade.
func TestTheDatabaseRefusesToOrphanAScoredParticipant(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	playHole(t, client, fix, 1, 4, 5)

	conn, err := util.Connect(ctx, util.LoadConfig().DatabaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	if _, err := conn.Exec(ctx, "SET LOCAL app.current_tenant_id = '"+fix.TenantID.String()+"'"); err != nil {
		t.Fatalf("set tenant: %v", err)
	}
	_, err = conn.Exec(ctx,
		"DELETE FROM match_participants WHERE match_id = $1 AND player_id = $2 AND tenant_id = $3",
		fix.MatchID, fix.RedPlayer, fix.TenantID)
	// Named, not merely non-nil: RLS refusing the row, or a typo in the statement, would
	// otherwise read as the constraint doing its job.
	if err == nil || !strings.Contains(err.Error(), "fk__scores__match_id_player_id__match_participants") {
		t.Fatalf("want the scores foreign key to refuse the delete, got %v", err)
	}
}
