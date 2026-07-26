package test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
	testjwt "github.com/manitoba-ryder-cup/scorecard/test/_util/jwt"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

// clientAndToken returns an SDK client and the raw token behind it, both for the same
// fresh tenant — freshClient and freshToken each mint their own, so a test that mixes
// them reads a different tenant and passes vacuously against an empty result.
func clientAndToken(t *testing.T) (*sdk.Client, string) {
	t.Helper()
	token := testjwt.MintAccessToken(t, uuid.New(), uuid.New(), writeScopes...)
	client := sdk.NewClient(util.LoadConfig().BaseURL)
	client.SetToken(token)
	return client, token
}

// Email is write-only: it is the seed CLI's identity key for recognising a returning
// player, never something a client needs back. Reads are public, so returning it would
// publish the roster's contact details to anonymous spectators.
func TestPlayerEmailIsNeverReturned(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client, token := clientAndToken(t)

	email := "roster.player@example.com"
	player, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{
		FirstName: "Roster", LastName: "Player", Email: &email,
	})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}

	// Raw responses, so the assertion survives the field being removed from the SDK type.
	for _, path := range []string{"/v1/players", "/v1/players/" + player.ID.String()} {
		status, body := request.Raw(t, "GET", path, "", token)
		if status != 200 {
			t.Fatalf("GET %s: status %d: %s", path, status, body)
		}
		if strings.Contains(body, "email") {
			t.Errorf("GET %s leaks email: %s", path, body)
		}
	}
}

// The roster and team listings join the player record, so they are a second way the
// address could reach the wire.
func TestRosterAndCaptainResponsesOmitEmail(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client, token := clientAndToken(t)

	email := "captain@example.com"
	player, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{
		FirstName: "Team", LastName: "Captain", Email: &email,
	})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Email Privacy Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	teams, err := client.GetTournamentTeams(ctx, tour.ID)
	if err != nil {
		t.Fatalf("get teams: %v", err)
	}
	if _, err := client.EnterTournamentPlayer(ctx, tour.ID, sdk.EnterTournamentPlayerRequest{PlayerID: player.ID}); err != nil {
		t.Fatalf("enter player: %v", err)
	}
	if _, err := client.DraftPlayer(ctx, teams[0].ID, sdk.DraftPlayerRequest{PlayerID: player.ID}); err != nil {
		t.Fatalf("draft player: %v", err)
	}
	if err := client.SetTeamCaptain(ctx, teams[0].ID, sdk.SetTeamCaptainRequest{CaptainID: player.ID}); err != nil {
		t.Fatalf("set captain: %v", err)
	}

	paths := []string{
		"/v1/tournaments/" + tour.ID.String() + "/players",
		"/v1/tournaments/" + tour.ID.String() + "/teams",
		"/v1/teams/" + teams[0].ID.String() + "/members",
	}
	for _, path := range paths {
		status, body := request.Raw(t, "GET", path, "", token)
		if status != 200 {
			t.Fatalf("GET %s: status %d: %s", path, status, body)
		}
		if strings.Contains(body, "email") {
			t.Errorf("GET %s leaks email: %s", path, body)
		}
	}
}
