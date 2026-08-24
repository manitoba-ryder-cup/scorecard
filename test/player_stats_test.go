package test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

// The derivation is unit-tested in internal/golf, so what this covers is the rest of the
// path: the route, the DTO, and the JSON the player page receives.

func TestPlayerStatsAfterAWin(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix) // Red takes the only match 10 & 8

	stats, err := client.GetPlayerStats(ctx, fix.RedPlayer)
	if err != nil {
		t.Fatalf("get player stats: %v", err)
	}

	// One win is one point, and the cup counts once however many matches it held.
	if stats.Points != 1 {
		t.Errorf("want 1 point for a win, got %v", stats.Points)
	}
	if stats.CupsPlayed != 1 {
		t.Errorf("want 1 cup played, got %d", stats.CupsPlayed)
	}

	// A match closed out on the 10th is decided early, not a last-hole match. The split
	// has to account for every match, so the other side must stay empty.
	if stats.DecidedEarly.Wins != 1 {
		t.Errorf("want the win recorded as decided early, got %+v", stats.DecidedEarly)
	}
	if stats.LastHole.Wins+stats.LastHole.Losses+stats.LastHole.Ties != 0 {
		t.Errorf("a 10 & 8 never reached the 18th, yet last_hole holds %+v", stats.LastHole)
	}

	if len(stats.ByFormat) != 1 || stats.ByFormat[0].FormatName != "Singles" {
		t.Errorf("want one Singles row, got %+v", stats.ByFormat)
	}
	if len(stats.Opponents) != 1 {
		t.Errorf("want the one opponent faced, got %+v", stats.Opponents)
	}

	// Best win is the heaviest result; the loser's side of it is where the heaviest loss
	// belongs, so this player must have none.
	if stats.BestWin == nil {
		t.Fatal("want a best win after winning a match, got null")
	}
	if stats.BestWin.Lead != 10 || stats.BestWin.HolesRemaining != 8 {
		t.Errorf("want the best win to be 10 & 8, got %+v", stats.BestWin)
	}
	if stats.HeaviestLoss != nil {
		t.Errorf("want no heaviest loss for an unbeaten player, got %+v", stats.HeaviestLoss)
	}
}

// The losing side of the same match, which is the only place a heaviest loss can come
// from — and the check that the two are not both filled from the winner's point of view.
func TestPlayerStatsForTheLosingSide(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	closeOutRedWin(t, client, fix)

	stats, err := client.GetPlayerStats(ctx, fix.BluePlayer)
	if err != nil {
		t.Fatalf("get player stats: %v", err)
	}
	if stats.Points != 0 {
		t.Errorf("want no points for a loss, got %v", stats.Points)
	}
	if stats.BestWin != nil {
		t.Errorf("want no best win for a player who has not won, got %+v", stats.BestWin)
	}
	if stats.HeaviestLoss == nil || stats.HeaviestLoss.Lead != 10 {
		t.Errorf("want the loss recorded at 10, got %+v", stats.HeaviestLoss)
	}
}

// A player who has never teed off still has a stats page, and every collection on it has
// to serialize as [] rather than null — the client renders them without a null guard.
func TestPlayerStatsForAPlayerWithNoMatches(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	player, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "Never", LastName: "Played"})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}

	stats, err := client.GetPlayerStats(ctx, player.ID)
	if err != nil {
		t.Fatalf("get player stats: %v", err)
	}
	if stats.Points != 0 || stats.CupsPlayed != 0 {
		t.Errorf("want an empty record, got %+v", stats)
	}
	if stats.BestWin != nil || stats.HeaviestLoss != nil {
		t.Errorf("want no notable matches, got %+v / %+v", stats.BestWin, stats.HeaviestLoss)
	}

	// Asserted on the raw body: a nil slice and an empty one are both len 0 once decoded,
	// so the decoded value cannot tell us which one went over the wire.
	path := strings.Replace(sdk.RouteV1PlayerStats, "{id}", player.ID.String(), 1)
	status, body := request.Raw(t, http.MethodGet, path, "", freshToken(t))
	if status != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", status, body)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, field := range []string{"by_format", "teammates", "opponents"} {
		if got := string(raw[field]); got != "[]" {
			t.Errorf("%s: want [], got %s", field, got)
		}
	}
}

// An unknown player is not a 404 here, unlike GET /v1/players/{id}. Both this and
// /tournaments are aggregations over match rows rather than reads of the player, so an ID
// that matches nothing produces an empty result rather than a missing one. Asserted so the
// difference is a decision on record: the entity read is what tells a client the player
// does not exist, and the client already makes that call before asking for stats.
func TestPlayerStatsForAnUnknownPlayerIsEmptyRatherThan404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	stats, err := client.GetPlayerStats(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("want an empty result, got %v", err)
	}
	if stats.CupsPlayed != 0 || stats.Points != 0 || len(stats.ByFormat) != 0 {
		t.Fatalf("want nothing recorded against an ID that matches no player, got %+v", stats)
	}
}
