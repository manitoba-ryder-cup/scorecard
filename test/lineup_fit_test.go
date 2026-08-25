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

func formatNamed(t *testing.T, client *sdk.Client, name string) sdk.MatchFormat {
	t.Helper()
	formats, err := client.ListMatchFormats(context.Background())
	if err != nil {
		t.Fatalf("list formats: %v", err)
	}
	for _, f := range formats {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no %s format seeded", name)
	return sdk.MatchFormat{}
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
	if apiErr.Message != "That side is full for this match's format. Remove a player before adding another." {
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

// A format with room takes the player the smaller one refused.
func TestAFourballSideTakesASecondPlayer(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	fourball := formatNamed(t, client, "Fourball")
	spare := anotherDraftedPlayer(t, fix, fix.TeamRed)

	if _, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{MatchFormatID: &fourball.ID}); err != nil {
		t.Fatalf("want the format change allowed, got %v", err)
	}
	if _, err := client.AddParticipant(ctx, fix.MatchID,
		sdk.AddParticipantRequest{PlayerID: spare, TeamID: fix.TeamRed}); err != nil {
		t.Fatalf("want the second player allowed, got %v", err)
	}
}

// Going the other way is what the rule exists for: the players are already there, and the
// smaller format has nowhere to put them.
func TestChangingToAFormatTheLineupOutgrowsIsRefused(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	singles := formatNamed(t, client, "Singles")
	fourball := formatNamed(t, client, "Fourball")

	if _, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{MatchFormatID: &fourball.ID}); err != nil {
		t.Fatalf("to fourball: %v", err)
	}
	for _, team := range []uuid.UUID{fix.TeamRed, fix.TeamBlue} {
		spare := anotherDraftedPlayer(t, fix, team)
		if _, err := client.AddParticipant(ctx, fix.MatchID,
			sdk.AddParticipantRequest{PlayerID: spare, TeamID: team}); err != nil {
			t.Fatalf("filling the side: %v", err)
		}
	}

	_, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{MatchFormatID: &singles.ID})

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %v", err)
	}
	if apiErr.Message != "This match has more players a side than that format allows. Remove the extras before changing it." {
		t.Errorf("message = %q", apiErr.Message)
	}
	if theMatch(t, client, fix).MatchFormatID != fourball.ID {
		t.Error("the format changed anyway")
	}
}

// A half-built lineup is an ordinary state, so growing the format is never refused for it.
func TestChangingToARoomierFormatIsAllowedWithAPartialLineup(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	fourball := formatNamed(t, client, "Fourball")

	if _, err := client.UpdateMatch(context.Background(), fix.MatchID,
		sdk.UpdateMatchRequest{MatchFormatID: &fourball.ID}); err != nil {
		t.Fatalf("want one a side allowed under fourball, got %v", err)
	}
}

// A player already in the match is a duplicate, not a full side. Counting them twice would
// answer "that side is full" to someone who has not changed the lineup at all.
func TestAddingTheSamePlayerTwiceIsNotReportedAsAFullSide(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)

	_, err := client.AddParticipant(context.Background(), fix.MatchID,
		sdk.AddParticipantRequest{PlayerID: fix.RedPlayer, TeamID: fix.TeamRed})

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %v", err)
	}
	if apiErr.Message == "That side is full for this match's format. Remove a player before adding another." {
		t.Error("a duplicate was reported as a full side")
	}
}
