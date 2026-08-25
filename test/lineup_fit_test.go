package test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
)

// anotherDraftedPlayer enters a player in the tournament and drafts them onto a team, so a
// side has somebody to be filled with. The fixture seeds exactly one a side.
func anotherDraftedPlayer(t *testing.T, fix *util.Fixture, teamID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	conn, err := util.Connect(ctx, util.LoadConfig().DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	if _, err := conn.Exec(ctx, "SET app.current_tenant_id = '"+fix.TenantID.String()+"'"); err != nil {
		t.Fatal(err)
	}

	var id uuid.UUID
	if err := conn.QueryRow(ctx,
		`INSERT INTO players (tenant_id, first_name, last_name) VALUES ($1, 'Spare', $2) RETURNING id`,
		fix.TenantID, uuid.NewString()[:8],
	).Scan(&id); err != nil {
		t.Fatalf("player: %v", err)
	}
	// Entering precedes drafting: the team_members FK requires it.
	if _, err := conn.Exec(ctx,
		`INSERT INTO tournament_players (tournament_id, player_id, tenant_id) VALUES ($1, $2, $3)`,
		fix.TournamentID, id, fix.TenantID); err != nil {
		t.Fatalf("enter: %v", err)
	}
	if _, err := conn.Exec(ctx,
		`INSERT INTO team_members (team_id, player_id, tournament_id, tenant_id) VALUES ($1, $2, $3, $4)`,
		teamID, id, fix.TournamentID, fix.TenantID); err != nil {
		t.Fatalf("draft: %v", err)
	}
	return id
}

// The fixture plays Singles, which takes one a side.
func TestAddingASecondPlayerToASinglesSideIsRefused(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	spare := anotherDraftedPlayer(t, fix, fix.TeamRed)

	_, err := client.AddParticipant(ctx, fix.MatchID,
		sdk.AddParticipantRequest{PlayerID: spare, TeamID: fix.TeamRed})

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %v", err)
	}
	if apiErr.Message != "That would be too many players a side for this format." {
		t.Errorf("message = %q", apiErr.Message)
	}

	participants, err := client.ListParticipants(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participants) != 2 {
		t.Errorf("the player was added anyway: %d participants", len(participants))
	}
}

// A format with room takes the player the smaller one refused. A separate match, because a
// format is chosen when the match is created and not changed afterwards.
func TestAFourballSideTakesTwoPlayers(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	fourball := matchInFormat(t, client, fix, "Fourball")

	for _, playerID := range []uuid.UUID{fix.RedPlayer, anotherDraftedPlayer(t, fix, fix.TeamRed)} {
		if _, err := client.AddParticipant(ctx, fourball,
			sdk.AddParticipantRequest{PlayerID: playerID, TeamID: fix.TeamRed}); err != nil {
			t.Fatalf("want two a side allowed under fourball, got %v", err)
		}
	}

	_, err := client.AddParticipant(ctx, fourball,
		sdk.AddParticipantRequest{PlayerID: anotherDraftedPlayer(t, fix, fix.TeamRed), TeamID: fix.TeamRed})

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want the third refused, got %v", err)
	}
}

// A player already in the match is a duplicate, not a side with no room. Counting them twice
// would refuse the lineup of someone who has not changed it at all.
func TestAddingTheSamePlayerTwiceIsNotReportedAsALineupProblem(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)

	_, err := client.AddParticipant(context.Background(), fix.MatchID,
		sdk.AddParticipantRequest{PlayerID: fix.RedPlayer, TeamID: fix.TeamRed})

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %v", err)
	}
	if apiErr.Message == "That would be too many players a side for this format." {
		t.Error("a duplicate was reported as a lineup that does not fit")
	}
}
