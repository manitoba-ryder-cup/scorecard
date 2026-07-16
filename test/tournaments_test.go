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

func TestCreateTournamentAndTeams(t *testing.T) {
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

	red, err := client.CreateTeam(ctx, tour.ID, sdk.CreateTeamRequest{Color: sdk.TeamColorRed})
	if err != nil {
		t.Fatalf("create red: %v", err)
	}
	if red.Color != sdk.TeamColorRed || red.TournamentID != tour.ID {
		t.Fatalf("unexpected red team: %+v", red)
	}
	if _, err := client.CreateTeam(ctx, tour.ID, sdk.CreateTeamRequest{Color: sdk.TeamColorBlue}); err != nil {
		t.Fatalf("create blue: %v", err)
	}

	teams, err := client.GetTournamentTeams(ctx, tour.ID)
	if err != nil {
		t.Fatalf("get teams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("want 2 teams, got %d", len(teams))
	}
	colors := map[string]bool{}
	for _, tm := range teams {
		colors[tm.Color] = true
	}
	if !colors[sdk.TeamColorRed] || !colors[sdk.TeamColorBlue] {
		t.Fatalf("want Red and Blue, got %v", colors)
	}
}

func TestCreateTeamDuplicateColorConflicts(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Dup Color Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Brandon",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	if _, err := client.CreateTeam(ctx, tour.ID, sdk.CreateTeamRequest{Color: sdk.TeamColorRed}); err != nil {
		t.Fatalf("create first red: %v", err)
	}

	// A second Red team collides with UNIQUE(tournament_id, color) -> 409.
	_, err = client.CreateTeam(ctx, tour.ID, sdk.CreateTeamRequest{Color: sdk.TeamColorRed})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 APIError, got %v", err)
	}
}

func TestCreateTeamInvalidColorRejected(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Bad Color Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Selkirk",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}

	_, err = client.CreateTeam(ctx, tour.ID, sdk.CreateTeamRequest{Color: "Green"})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
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
