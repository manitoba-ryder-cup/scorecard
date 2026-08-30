package test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
	testjwt "github.com/manitoba-ryder-cup/scorecard/test/_util/jwt"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

// writeScopes are all the scopes an admin-style test client needs to exercise the
// write endpoints.
var writeScopes = []string{sdk.ScopeTournamentsWrite, sdk.ScopePlayersWrite, sdk.ScopeScoresWrite, sdk.ScopeCoursesWrite}

// freshToken mints an access token for a brand-new tenant, carrying all write scopes.
func freshToken(t *testing.T) string {
	t.Helper()
	return testjwt.MintAccessToken(t, uuid.New(), uuid.New(), writeScopes...)
}

// freshClient returns a client authenticated for a brand-new tenant, with nothing
// seeded — the write endpoints build their own state from scratch.
func freshClient(t *testing.T) *sdk.Client {
	t.Helper()
	client := sdk.NewClient(util.LoadConfig().BaseURL)
	client.SetToken(freshToken(t))
	return client
}

func TestCreateTournamentSeedsBothTeams(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Manitoba Ryder Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	if tour.ID == uuid.Nil || tour.Name != "Manitoba Ryder Cup" || tour.StartDate != "2026-08-01" || tour.EndDate != "2026-08-03" {
		t.Fatalf("unexpected tournament: %+v", tour)
	}

	// Round-trips through a fresh read.
	got, err := client.GetTournament(ctx, tour.ID)
	if err != nil {
		t.Fatalf("get tournament: %v", err)
	}
	if got.ID != tour.ID || got.Location != "Winnipeg" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if tour.Phase != sdk.PhaseUpcoming || got.Phase != sdk.PhaseUpcoming {
		t.Fatalf("a cup with no matches is upcoming: created %q, read %q", tour.Phase, got.Phase)
	}

	// Creating a tournament seeds its two sides; there is no separate creation step.
	teams, err := client.GetTournamentTeams(ctx, tour.ID)
	if err != nil {
		t.Fatalf("get teams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("want 2 teams seeded, got %d", len(teams))
	}
	colors := map[string]bool{}
	for _, tm := range teams {
		colors[tm.Color] = true
	}
	if !colors[sdk.TeamColorRed] || !colors[sdk.TeamColorBlue] {
		t.Fatalf("want Red and Blue, got %v", colors)
	}
}

// TestWriteWithoutScopeForbidden confirms a valid token lacking the write scope is
// authorized-but-forbidden (403), distinct from unauthenticated (401).
func TestWriteWithoutScopeForbidden(t *testing.T) {
	t.Parallel()
	client := sdk.NewClient(util.LoadConfig().BaseURL)
	client.SetToken(testjwt.MintAccessToken(t, uuid.New(), uuid.New())) // no scopes

	_, err := client.CreateTournament(context.Background(), sdk.CreateTournamentRequest{
		Name: "Unauthorized Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	wantsStatus(t, err, http.StatusForbidden)
}

func TestGetNonexistentTournamentReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	_, err := client.GetTournament(context.Background(), uuid.New())
	wantsStatus(t, err, http.StatusNotFound)
}

// The SDK client would reject end-before-start before sending, so this hits the
// server directly to confirm it validates too (a non-SDK caller must get 400).
func TestCreateTournamentInvalidDatesRejectedByServer(t *testing.T) {
	t.Parallel()
	body := `{"name":"Backwards Cup","start_date":"2026-08-03","end_date":"2026-08-01","location":"Winnipeg"}`
	status, _ := request.Raw(t, http.MethodPost, sdk.RouteV1Tournaments, body, freshToken(t))
	if status != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", status)
	}
}

func phaseOf(t *testing.T, client *sdk.Client, id uuid.UUID) sdk.TournamentPhase {
	t.Helper()
	tour, err := client.GetTournament(context.Background(), id)
	if err != nil {
		t.Fatalf("get tournament: %v", err)
	}
	return tour.Phase
}

// The phase the landing page renders itself from. It is published rather than left to each
// client to re-derive, so this walks a cup through all three and checks the server agrees
// with what actually happened to its matches — in particular that a scheduled cup is not
// "live" merely because it is unfinished.
func TestTournamentPhaseFollowsPlay(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)

	if got := phaseOf(t, client, fix.TournamentID); got != sdk.PhaseUpcoming {
		t.Fatalf("scheduled, nothing scored: phase = %q, want upcoming", got)
	}

	playHole(t, client, fix, 1, 4, 5)
	if got := phaseOf(t, client, fix.TournamentID); got != sdk.PhaseLive {
		t.Fatalf("one hole in: phase = %q, want live", got)
	}

	for h := int32(2); h <= 10; h++ {
		playHole(t, client, fix, h, 4, 5)
	}
	if got := phaseOf(t, client, fix.TournamentID); got != sdk.PhaseFinished {
		t.Fatalf("match closed out: phase = %q, want finished", got)
	}
}
