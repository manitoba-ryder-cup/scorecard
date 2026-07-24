package test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

func TestSetTeamCaptainAndRead(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Captain Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	redTeam := teamByColor(t, client, tour.ID, sdk.TeamColorRed)

	p, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "Cap", LastName: "Tain"})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	if _, err := client.EnterTournamentPlayer(ctx, tour.ID, sdk.EnterTournamentPlayerRequest{PlayerID: p.ID}); err != nil {
		t.Fatalf("enter: %v", err)
	}
	if _, err := client.DraftPlayer(ctx, redTeam, sdk.DraftPlayerRequest{PlayerID: p.ID}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if err := client.SetTeamCaptain(ctx, redTeam, sdk.SetTeamCaptainRequest{CaptainID: p.ID}); err != nil {
		t.Fatalf("set captain: %v", err)
	}

	// The captain now shows up on the tournament teams view (single-query LEFT JOIN).
	teams, err := client.GetTournamentTeams(ctx, tour.ID)
	if err != nil {
		t.Fatalf("get teams: %v", err)
	}
	var found bool
	for _, tm := range teams {
		if tm.ID == redTeam {
			if tm.Captain == nil || tm.Captain.ID != p.ID {
				t.Fatalf("want captain %s, got %+v", p.ID, tm.Captain)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("red team missing from teams list")
	}
}

func TestSetCaptainUnknownTeamReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	err := client.SetTeamCaptain(context.Background(), uuid.New(), sdk.SetTeamCaptainRequest{CaptainID: uuid.New()})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

func TestClearTeamCaptain(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Clear Captain Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	redTeam := teamByColor(t, client, tour.ID, sdk.TeamColorRed)
	p, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "Cap", LastName: "Tain"})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	if _, err := client.EnterTournamentPlayer(ctx, tour.ID, sdk.EnterTournamentPlayerRequest{PlayerID: p.ID}); err != nil {
		t.Fatalf("enter: %v", err)
	}
	if _, err := client.DraftPlayer(ctx, redTeam, sdk.DraftPlayerRequest{PlayerID: p.ID}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if err := client.SetTeamCaptain(ctx, redTeam, sdk.SetTeamCaptainRequest{CaptainID: p.ID}); err != nil {
		t.Fatalf("set captain: %v", err)
	}

	// Clearing removes the captain but leaves the player drafted on the team.
	if err := client.ClearTeamCaptain(ctx, redTeam); err != nil {
		t.Fatalf("clear captain: %v", err)
	}
	teams, err := client.GetTournamentTeams(ctx, tour.ID)
	if err != nil {
		t.Fatalf("get teams: %v", err)
	}
	for _, tm := range teams {
		if tm.ID == redTeam && tm.Captain != nil {
			t.Fatalf("want captain cleared, got %+v", tm.Captain)
		}
	}
	members, err := client.ListTeamMembers(ctx, redTeam)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("want player still drafted after clearing captain, got %d", len(members))
	}
}

func TestClearCaptainUnknownTeamReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	err := client.ClearTeamCaptain(context.Background(), uuid.New())
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

func TestSetCaptainUnknownPlayerRejected(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Bad Captain Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Brandon",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	redTeam := teamByColor(t, client, tour.ID, sdk.TeamColorRed)

	// captain_id references a nonexistent player -> FK violation -> 400.
	err = client.SetTeamCaptain(ctx, redTeam, sdk.SetTeamCaptainRequest{CaptainID: uuid.New()})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}
