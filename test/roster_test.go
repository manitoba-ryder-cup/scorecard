package test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// enterPrereqs creates a tournament and a player, returning their IDs.
func enterPrereqs(t *testing.T, client *sdk.Client) (tournamentID, playerID uuid.UUID) {
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "No Player Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Brandon",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}

	// player_id uuid.New() doesn't exist -> FK violation -> 400.
	_, err = client.EnterTournamentPlayer(ctx, tour.ID, sdk.EnterTournamentPlayerRequest{PlayerID: uuid.New()})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}

// teamByColor returns the tournament's team of the given color.
func teamByColor(t *testing.T, client *sdk.Client, tournamentID uuid.UUID, color string) uuid.UUID {
	t.Helper()
	teams, err := client.GetTournamentTeams(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("get teams: %v", err)
	}
	for _, tm := range teams {
		if tm.Color == color {
			return tm.ID
		}
	}
	t.Fatalf("no %s team found", color)
	return uuid.Nil
}

func TestDraftPlayerOntoTeam(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tournamentID, playerID := enterPrereqs(t, client)
	if _, err := client.EnterTournamentPlayer(ctx, tournamentID, sdk.EnterTournamentPlayerRequest{PlayerID: playerID}); err != nil {
		t.Fatalf("enter: %v", err)
	}
	redTeam := teamByColor(t, client, tournamentID, sdk.TeamColorRed)

	member, err := client.DraftPlayer(ctx, redTeam, sdk.DraftPlayerRequest{PlayerID: playerID})
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	if member.TeamID != redTeam || member.PlayerID != playerID || member.TournamentID != tournamentID {
		t.Fatalf("unexpected member: %+v", member)
	}

	// The team-members list (same entry shape, filtered) includes the drafted player.
	members, err := client.ListTeamMembers(ctx, redTeam)
	if err != nil {
		t.Fatalf("list team members: %v", err)
	}
	if len(members) != 1 || members[0].PlayerID != playerID || members[0].TeamID == nil || *members[0].TeamID != redTeam {
		t.Fatalf("unexpected team members: %+v", members)
	}

	// The tournament roster now shows the player's team assignment.
	roster, err := client.ListTournamentPlayers(ctx, tournamentID)
	if err != nil {
		t.Fatalf("list roster: %v", err)
	}
	if len(roster) != 1 || roster[0].TeamID == nil || *roster[0].TeamID != redTeam {
		t.Fatalf("roster should show team assignment: %+v", roster)
	}
}

func TestDraftUnenteredPlayerRejected(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tournamentID, playerID := enterPrereqs(t, client) // player created but NOT entered
	redTeam := teamByColor(t, client, tournamentID, sdk.TeamColorRed)

	// Not a tournament_player -> composite FK violation -> 400.
	_, err := client.DraftPlayer(ctx, redTeam, sdk.DraftPlayerRequest{PlayerID: playerID})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}

func TestDraftAlreadyDraftedConflicts(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tournamentID, playerID := enterPrereqs(t, client)
	if _, err := client.EnterTournamentPlayer(ctx, tournamentID, sdk.EnterTournamentPlayerRequest{PlayerID: playerID}); err != nil {
		t.Fatalf("enter: %v", err)
	}
	redTeam := teamByColor(t, client, tournamentID, sdk.TeamColorRed)
	if _, err := client.DraftPlayer(ctx, redTeam, sdk.DraftPlayerRequest{PlayerID: playerID}); err != nil {
		t.Fatalf("first draft: %v", err)
	}

	// Drafting the same player again -> 409.
	_, err := client.DraftPlayer(ctx, redTeam, sdk.DraftPlayerRequest{PlayerID: playerID})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 APIError, got %v", err)
	}
}

func TestDraftToNonexistentTeamReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	_, err := client.DraftPlayer(context.Background(), uuid.New(), sdk.DraftPlayerRequest{PlayerID: uuid.New()})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

func TestUpdateUnenteredPlayerReturns404(t *testing.T) {
	t.Parallel()
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

func TestUndraftPlayer(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tournamentID, playerID := enterPrereqs(t, client)
	if _, err := client.EnterTournamentPlayer(ctx, tournamentID, sdk.EnterTournamentPlayerRequest{PlayerID: playerID}); err != nil {
		t.Fatalf("enter: %v", err)
	}
	redTeam := teamByColor(t, client, tournamentID, sdk.TeamColorRed)
	if _, err := client.DraftPlayer(ctx, redTeam, sdk.DraftPlayerRequest{PlayerID: playerID}); err != nil {
		t.Fatalf("draft: %v", err)
	}

	if err := client.UndraftPlayer(ctx, redTeam, playerID); err != nil {
		t.Fatalf("undraft: %v", err)
	}

	// The team is empty again.
	members, err := client.ListTeamMembers(ctx, redTeam)
	if err != nil {
		t.Fatalf("list team members: %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("want no members after undraft, got %d", len(members))
	}
	// The player is still entered in the tournament, just unassigned (team_id cleared).
	roster, err := client.ListTournamentPlayers(ctx, tournamentID)
	if err != nil {
		t.Fatalf("list roster: %v", err)
	}
	if len(roster) != 1 || roster[0].TeamID != nil {
		t.Fatalf("want entered-but-unassigned player, got %+v", roster)
	}
}

func TestUndraftClearsCaptaincy(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tournamentID, playerID := enterPrereqs(t, client)
	if _, err := client.EnterTournamentPlayer(ctx, tournamentID, sdk.EnterTournamentPlayerRequest{PlayerID: playerID}); err != nil {
		t.Fatalf("enter: %v", err)
	}
	redTeam := teamByColor(t, client, tournamentID, sdk.TeamColorRed)
	if _, err := client.DraftPlayer(ctx, redTeam, sdk.DraftPlayerRequest{PlayerID: playerID}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if err := client.SetTeamCaptain(ctx, redTeam, sdk.SetTeamCaptainRequest{CaptainID: playerID}); err != nil {
		t.Fatalf("set captain: %v", err)
	}

	// Undrafting the captain must also strip their captaincy — a team can't keep a captain
	// who is no longer on it (otherwise the team name derived from the captain goes stale).
	if err := client.UndraftPlayer(ctx, redTeam, playerID); err != nil {
		t.Fatalf("undraft: %v", err)
	}

	teams, err := client.GetTournamentTeams(ctx, tournamentID)
	if err != nil {
		t.Fatalf("get teams: %v", err)
	}
	var red *sdk.TournamentTeam
	for i := range teams {
		if teams[i].ID == redTeam {
			red = &teams[i]
		}
	}
	if red == nil {
		t.Fatalf("red team not found in %+v", teams)
	}
	if red.Captain != nil {
		t.Fatalf("want captain cleared after undraft, got %+v", red.Captain)
	}
}

func TestUndraftPlayerNotOnTeamReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tournamentID, playerID := enterPrereqs(t, client)
	if _, err := client.EnterTournamentPlayer(ctx, tournamentID, sdk.EnterTournamentPlayerRequest{PlayerID: playerID}); err != nil {
		t.Fatalf("enter: %v", err)
	}
	redTeam := teamByColor(t, client, tournamentID, sdk.TeamColorRed)

	// Entered but never drafted -> not a member -> 404.
	err := client.UndraftPlayer(ctx, redTeam, playerID)
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}
