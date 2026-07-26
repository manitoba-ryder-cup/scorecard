package test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
	testjwt "github.com/manitoba-ryder-cup/scorecard/test/_util/jwt"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

// TestMain preflights the infrastructure so `go test ./test/...` without a running
// stack skips with a clear hint instead of failing on connection errors.
func TestMain(m *testing.M) {
	cfg := util.LoadConfig()
	ctx := context.Background()

	conn, err := util.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Printf("SKIP: cannot reach test database (%v). Run `make test-setup` first.\n", err)
		os.Exit(0)
	}
	_ = conn.Close(ctx)

	if _, err := http.Get(cfg.BaseURL + sdk.RouteHealth); err != nil {
		fmt.Printf("SKIP: cannot reach scorecard at %s (%v). Run `make test-setup` first.\n", cfg.BaseURL, err)
		os.Exit(0)
	}

	os.Exit(m.Run())
}

// authedClient seeds a fresh single-match fixture under a new tenant and returns an
// SDK client authenticated for that tenant.
func authedClient(t *testing.T) (*sdk.Client, *util.Fixture) {
	t.Helper()
	cfg := util.LoadConfig()
	ctx := context.Background()

	conn, err := util.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })

	fix, err := util.SeedSinglesMatch(ctx, conn)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	client := sdk.NewClient(cfg.BaseURL)
	client.SetToken(testjwt.MintAccessToken(t, fix.TenantID, uuid.New(), writeScopes...))
	return client, fix
}

// playHole records both sides' scores for one hole, the way the app does: one request.
func playHole(t *testing.T, client *sdk.Client, fix *util.Fixture, hole, red, blue int32) sdk.MatchStatus {
	t.Helper()
	rp, bp := fix.RedPlayer, fix.BluePlayer
	status, err := client.SubmitScore(context.Background(), fix.MatchID, sdk.ScoreSubmission{
		HoleNumber: hole,
		Scores: []sdk.ScoreEntry{
			{TeamID: fix.TeamRed, PlayerID: &rp, Strokes: red},
			{TeamID: fix.TeamBlue, PlayerID: &bp, Strokes: blue},
		},
	})
	if err != nil {
		t.Fatalf("play hole %d: %v", hole, err)
	}
	return status
}

// TestScoreEntryClosesTheMaterializationLoop drives the full write→materialize→derive
// path against real Postgres: submitting scores through the API updates match_results,
// which the winner/status/player-record reads then reflect.
func TestScoreEntryClosesTheMaterializationLoop(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()

	// Red wins holes 1-10 outright (4 vs 5): 10 up with 8 to play closes the match out.
	for h := int32(1); h <= 10; h++ {
		playHole(t, client, fix, h, 4, 5)
	}

	// Materialized winner.
	winner, err := client.GetMatchStatus(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("get winner: %v", err)
	}
	if !winner.Finished {
		t.Fatalf("want finished match, got %+v", winner)
	}
	if winner.WinnerTeamID == nil || *winner.WinnerTeamID != fix.TeamRed {
		t.Fatalf("want winner team %d, got %+v", fix.TeamRed, winner.WinnerTeamID)
	}

	// Live progression stops at the decided hole, led by Red.
	holes, err := client.GetMatchScores(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("get scores: %v", err)
	}
	if len(holes) == 0 {
		t.Fatal("want hole-by-hole progression, got none")
	}
	last := holes[len(holes)-1]
	if !last.Decided || last.LeaderTeamID == nil || *last.LeaderTeamID != fix.TeamRed {
		t.Fatalf("want decided hole led by Red, got %+v", last)
	}

	// Derived player records: Red 1-0-0, Blue 0-1-0.
	red, err := client.GetPlayer(ctx, fix.RedPlayer)
	if err != nil {
		t.Fatalf("get red player: %v", err)
	}
	if red.Record.Wins != 1 || red.Record.Losses != 0 || red.Record.Ties != 0 {
		t.Errorf("red record want 1-0-0, got %+v", red.Record)
	}
	blue, err := client.GetPlayer(ctx, fix.BluePlayer)
	if err != nil {
		t.Fatalf("get blue player: %v", err)
	}
	if blue.Record.Losses != 1 || blue.Record.Wins != 0 {
		t.Errorf("blue record want 0-1-0, got %+v", blue.Record)
	}
}

// TestSubmitScoreReturnsTheRecomputedStatus checks the write's own response tracks the
// match, so a client never has to re-derive the close-out rule to know it just ended.
func TestSubmitScoreReturnsTheRecomputedStatus(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)

	var status sdk.MatchStatus
	for h := int32(1); h <= 10; h++ {
		status = playHole(t, client, fix, h, 4, 5)
		// Only the hole that closes it out reports finished.
		if want := h == 10; status.Finished != want {
			t.Fatalf("hole %d: want finished=%v, got %+v", h, want, status)
		}
	}

	if status.WinnerTeamID == nil || *status.WinnerTeamID != fix.TeamRed {
		t.Errorf("want Red as winner, got %+v", status)
	}
	if status.Lead != 10 || status.HolesRemaining != 8 {
		t.Errorf("want 10 & 8, got %+v", status)
	}
}

// TestSubmitScoreRejectsHolesAfterTheCloseOut locks a finished match against new holes —
// the stale-client case where "next hole" keeps walking past the end of the match.
func TestSubmitScoreRejectsHolesAfterTheCloseOut(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	red := fix.RedPlayer

	// Red wins holes 1-10 outright: 10 up with 8 to play closes the match out.
	for h := int32(1); h <= 10; h++ {
		playHole(t, client, fix, h, 4, 5)
	}

	_, err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
		HoleNumber: 11, Scores: []sdk.ScoreEntry{{TeamID: fix.TeamRed, PlayerID: &red, Strokes: 4}},
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 for a hole after the close-out, got %v", err)
	}

	// A hole that was played stays correctable: a typo can be what ended the match early.
	status, err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
		HoleNumber: 5, Scores: []sdk.ScoreEntry{{TeamID: fix.TeamRed, PlayerID: &red, Strokes: 9}},
	})
	if err != nil {
		t.Fatalf("want the correction accepted, got %v", err)
	}
	if status.Finished || status.Lead != 8 {
		t.Errorf("want the correction to reopen the match at 8 up, got %+v", status)
	}
}

// TestSubmitScoreWritesTheHoleAtomically is why a hole is one request: a rejected entry
// must take the whole hole with it, rather than leaving one side scored and the other not
// for a client that never retries.
func TestSubmitScoreWritesTheHoleAtomically(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	red, blue := fix.RedPlayer, fix.BluePlayer

	_, err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
		HoleNumber: 1,
		Scores: []sdk.ScoreEntry{
			{TeamID: fix.TeamRed, PlayerID: &red, Strokes: 4},
			{TeamID: uuid.New(), PlayerID: &blue, Strokes: 5}, // a team not in this match
		},
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 for a team not in the match, got %v", err)
	}

	// Red's valid score must not have landed on its own.
	scores, err := client.GetMatchScores(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("get scores: %v", err)
	}
	if len(scores) != 0 {
		t.Fatalf("want nothing written, got %+v", scores)
	}
	status, err := client.GetMatchStatus(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if status.Finished || status.LeaderTeamID != nil {
		t.Errorf("want an untouched match, got %+v", status)
	}
}

// TestSubmitScoreUpsertsHole confirms the ON CONFLICT update path: re-submitting the
// same hole overwrites the prior strokes rather than inserting a duplicate.
func TestSubmitScoreUpsertsHole(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()

	redHole1 := func() int32 {
		holes, err := client.GetMatchScores(ctx, fix.MatchID)
		if err != nil {
			t.Fatalf("get scores: %v", err)
		}
		for _, h := range holes {
			if h.HoleNumber != 1 {
				continue
			}
			for _, ts := range h.TeamScores {
				if ts.TeamID == fix.TeamRed {
					return ts.Strokes
				}
			}
		}
		t.Fatal("hole 1 Red score not found")
		return 0
	}

	playHole(t, client, fix, 1, 5, 5)
	if got := redHole1(); got != 5 {
		t.Fatalf("after first submit want 5, got %d", got)
	}
	// Re-recording the hole corrects Red rather than inserting a second row.
	playHole(t, client, fix, 1, 3, 5)
	if got := redHole1(); got != 3 {
		t.Fatalf("after upsert want 3, got %d", got)
	}
}

// TestSubmitScoreRejectsInvalidStrokes confirms the server rejects non-positive
// strokes with 400. Sent raw (bypassing the SDK client's validation); shape checks
// run before any match lookup, so no seeded match is needed.
func TestSubmitScoreRejectsInvalidStrokes(t *testing.T) {
	t.Parallel()
	// Valid UUID path + team_id so the request reaches strokes validation.
	body := `{"hole_number":1,"scores":[{"team_id":"11111111-1111-1111-1111-111111111111","strokes":0}]}`
	status, _ := request.Raw(t, http.MethodPost, "/v1/matches/11111111-1111-1111-1111-111111111111/scores", body, freshToken(t))
	if status != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", status)
	}
}

// TestSubmitScoreToNonexistentMatchReturns404 confirms a missing match surfaces as
// 404, not 500. The body is shape-valid so it passes client + server validation and
// reaches the match lookup, which fails with not-found.
func TestSubmitScoreToNonexistentMatchReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	_, err := client.SubmitScore(context.Background(), uuid.New(), sdk.ScoreSubmission{
		HoleNumber: 1, Scores: []sdk.ScoreEntry{{TeamID: uuid.New(), Strokes: 4}},
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

// TestUnauthenticatedWriteRejected confirms writes require a token: reads are public
// now, but an unauthenticated write is 401. Sent raw so no client token is attached.
func TestUnauthenticatedWriteRejected(t *testing.T) {
	t.Parallel()
	body := `{"first_name":"No","last_name":"Auth"}`
	status, _ := request.Raw(t, http.MethodPost, sdk.RouteV1Players, body, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", status)
	}
}
