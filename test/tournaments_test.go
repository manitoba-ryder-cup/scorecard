package test

import (
	"context"
	"errors"
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
var writeScopes = []string{sdk.ScopeTournamentsWrite, sdk.ScopePlayersWrite, sdk.ScopeScoresWrite}

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
	client := freshClient(t)
	ctx := context.Background()

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Manitoba Ryder Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	if tour.ID == 0 || tour.Name != "Manitoba Ryder Cup" || tour.StartDate != "2026-08-01" || tour.EndDate != "2026-08-03" {
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

	// Creating the tournament must have seeded exactly its two sides, Red and Blue —
	// no separate team-creation step exists.
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
	client := sdk.NewClient(util.LoadConfig().BaseURL)
	client.SetToken(testjwt.MintAccessToken(t, uuid.New(), uuid.New())) // no scopes

	_, err := client.CreateTournament(context.Background(), sdk.CreateTournamentRequest{
		Name: "Unauthorized Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 APIError, got %v", err)
	}
}

func TestGetNonexistentTournamentReturns404(t *testing.T) {
	client := freshClient(t)

	_, err := client.GetTournament(context.Background(), 999999)
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

// The SDK client would reject end-before-start before sending, so this hits the
// server directly to confirm it validates too (a non-SDK caller must get 400).
func TestCreateTournamentInvalidDatesRejectedByServer(t *testing.T) {
	body := `{"name":"Backwards Cup","start_date":"2026-08-03","end_date":"2026-08-01","location":"Winnipeg"}`
	status, _ := request.Raw(t, http.MethodPost, sdk.RouteV1Tournaments, body, freshToken(t))
	if status != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", status)
	}
}
