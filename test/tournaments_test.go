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
)

// freshClient returns a client authenticated for a brand-new tenant, with nothing
// seeded — the write endpoints build their own state from scratch.
func freshClient(t *testing.T) *sdk.Client {
	t.Helper()
	cfg := util.LoadConfig()
	client := sdk.NewClient(cfg.BaseURL)
	client.SetToken(testjwt.MintAccessToken(t, uuid.New(), uuid.New()))
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

func TestCreateTournamentInvalidDatesRejected(t *testing.T) {
	client := freshClient(t)

	_, err := client.CreateTournament(context.Background(), sdk.CreateTournamentRequest{
		Name: "Backwards Cup", StartDate: "2026-08-03", EndDate: "2026-08-01", Location: "Winnipeg",
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}
