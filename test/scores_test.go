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
	conn.Close(ctx)

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
	t.Cleanup(func() { conn.Close(ctx) })

	fix, err := util.SeedSinglesMatch(ctx, conn)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	client := sdk.NewClient(cfg.BaseURL)
	client.SetToken(testjwt.MintAccessToken(t, fix.TenantID, uuid.New()))
	return client, fix
}

// TestScoreEntryClosesTheMaterializationLoop drives the full write→materialize→derive
// path against real Postgres: submitting scores through the API updates match_results,
// which the winner/status/player-record reads then reflect.
func TestScoreEntryClosesTheMaterializationLoop(t *testing.T) {
	client, fix := authedClient(t)
	ctx := context.Background()

	// Red wins holes 1-10 outright (4 vs 5): 10 up with 8 to play closes the match out.
	for h := int32(1); h <= 10; h++ {
		red := fix.RedPlayer
		blue := fix.BluePlayer
		if err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
			HoleNumber: h, Strokes: 4, TeamID: fix.TeamRed, PlayerID: &red,
		}); err != nil {
			t.Fatalf("submit red hole %d: %v", h, err)
		}
		if err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
			HoleNumber: h, Strokes: 5, TeamID: fix.TeamBlue, PlayerID: &blue,
		}); err != nil {
			t.Fatalf("submit blue hole %d: %v", h, err)
		}
	}

	// Materialized winner.
	winner, err := client.GetMatchWinner(ctx, fix.MatchID)
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

// TestSubmitScoreUpsertsHole confirms the ON CONFLICT update path: re-submitting the
// same hole overwrites the prior strokes rather than inserting a duplicate.
func TestSubmitScoreUpsertsHole(t *testing.T) {
	client, fix := authedClient(t)
	ctx := context.Background()
	red := fix.RedPlayer
	blue := fix.BluePlayer

	// A match-play hole only surfaces in the progression once both sides have scored
	// it, so give Blue a score on hole 1; the test then upserts Red's score.
	if err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
		HoleNumber: 1, Strokes: 5, TeamID: fix.TeamBlue, PlayerID: &blue,
	}); err != nil {
		t.Fatalf("submit blue: %v", err)
	}

	submit := func(strokes int32) {
		if err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
			HoleNumber: 1, Strokes: strokes, TeamID: fix.TeamRed, PlayerID: &red,
		}); err != nil {
			t.Fatalf("submit strokes %d: %v", strokes, err)
		}
	}
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

	submit(5)
	if got := redHole1(); got != 5 {
		t.Fatalf("after first submit want 5, got %d", got)
	}
	submit(3)
	if got := redHole1(); got != 3 {
		t.Fatalf("after upsert want 3, got %d", got)
	}
}

// TestSubmitScoreRejectsInvalidStrokes confirms the server rejects non-positive
// strokes with 400. Sent raw (bypassing the SDK client's validation); shape checks
// run before any match lookup, so no seeded match is needed.
func TestSubmitScoreRejectsInvalidStrokes(t *testing.T) {
	body := `{"hole_number":1,"strokes":0,"team_id":1}`
	status, _ := request.Raw(t, http.MethodPost, "/v1/matches/1/scores", body, freshToken(t))
	if status != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", status)
	}
}

// TestUnauthenticatedRequestRejected confirms the JWT middleware guards the API.
func TestUnauthenticatedRequestRejected(t *testing.T) {
	cfg := util.LoadConfig()
	client := sdk.NewClient(cfg.BaseURL) // no token
	_, err := client.ListPlayers(context.Background())
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 APIError, got %v", err)
	}
}
