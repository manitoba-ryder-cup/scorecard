package test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// enterPrereqs creates a tournament and a player, returning their IDs.
func enterPrereqs(t *testing.T, client *sdk.Client) (tournamentID, playerID int32) {
	t.Helper()
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Roster Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	email := "roster@example.com"
	player, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "Rory", LastName: "McIlroy", Email: &email})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	return tour.ID, player.ID
}

func TestEnterUpdateAndListTournamentPlayers(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()
	tournamentID, playerID := enterPrereqs(t, client)

	// Enter with attributes set independently of any team.
	entry, err := client.EnterTournamentPlayer(ctx, tournamentID, sdk.EnterTournamentPlayerRequest{
		PlayerID: playerID, Tier: "gold", Biography: "Four-time major winner", Hdcp: 2.5,
	})
	if err != nil {
		t.Fatalf("enter player: %v", err)
	}
	if entry.PlayerID != playerID || entry.Tier != "gold" || entry.Hdcp != 2.5 {
		t.Fatalf("unexpected entry: %+v", entry)
	}

	// Listing includes the player's identity alongside the attributes.
	list, err := client.ListTournamentPlayers(ctx, tournamentID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].PlayerID != playerID || list[0].LastName != "McIlroy" || list[0].Tier != "gold" {
		t.Fatalf("unexpected roster: %+v", list)
	}

	// Attributes can be updated independently.
	updated, err := client.UpdateTournamentPlayer(ctx, tournamentID, playerID, sdk.UpdateTournamentPlayerRequest{
		Tier: "silver", Biography: "Updated", Hdcp: 3,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Tier != "silver" || updated.Hdcp != 3 {
		t.Fatalf("unexpected update: %+v", updated)
	}
}

func TestEnterTournamentPlayerDefaultsTier(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()
	tournamentID, playerID := enterPrereqs(t, client)

	entry, err := client.EnterTournamentPlayer(ctx, tournamentID, sdk.EnterTournamentPlayerRequest{PlayerID: playerID})
	if err != nil {
		t.Fatalf("enter player: %v", err)
	}
	if entry.Tier != "white" {
		t.Fatalf("want default tier 'white', got %q", entry.Tier)
	}
}

func TestEnterTournamentPlayerDuplicateConflicts(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()
	tournamentID, playerID := enterPrereqs(t, client)

	if _, err := client.EnterTournamentPlayer(ctx, tournamentID, sdk.EnterTournamentPlayerRequest{PlayerID: playerID}); err != nil {
		t.Fatalf("first enter: %v", err)
	}
	_, err := client.EnterTournamentPlayer(ctx, tournamentID, sdk.EnterTournamentPlayerRequest{PlayerID: playerID})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 APIError, got %v", err)
	}
}

func TestEnterUnknownPlayerRejected(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "No Player Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Brandon",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}

	// player_id 999999 doesn't exist -> FK violation -> 400.
	_, err = client.EnterTournamentPlayer(ctx, tour.ID, sdk.EnterTournamentPlayerRequest{PlayerID: 999999})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}

func TestUpdateUnenteredPlayerReturns404(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()
	tournamentID, playerID := enterPrereqs(t, client)

	// Player exists but was never entered in this tournament.
	_, err := client.UpdateTournamentPlayer(ctx, tournamentID, playerID, sdk.UpdateTournamentPlayerRequest{Tier: "gold"})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}
