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
func TestASinglesLineupTakesOneASide(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	spare := anotherDraftedPlayer(t, fix, fix.TeamRed)

	err := client.SetLineup(ctx, fix.MatchID, theLineup(
		onSide(fix.RedPlayer, fix.TeamRed),
		onSide(spare, fix.TeamRed),
		onSide(fix.BluePlayer, fix.TeamBlue),
	))

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 for two on a singles side, got %v", err)
	}
	if apiErr.Message != "That lineup isn't the right size for this match's format." {
		t.Errorf("message = %q", apiErr.Message)
	}

	participants, err := client.ListParticipants(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participants) != 2 {
		t.Errorf("the lineup was written anyway: %d participants", len(participants))
	}
}

// A lineup arrives complete, so a side short is refused for the same reason a side over is.
func TestALineupMissingASideIsRefused(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)

	err := client.SetLineup(context.Background(), fix.MatchID, theLineup(onSide(fix.RedPlayer, fix.TeamRed)))

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 for one side only, got %v", err)
	}
}

// A format with room takes the players the smaller one refused. A separate match, because a
// format is chosen when the match is created and not changed afterwards.
func TestAFourballLineupTakesTwoASide(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	fourball := matchInFormat(t, client, fix, "Fourball")

	err := client.SetLineup(context.Background(), fourball, theLineup(
		onSide(fix.RedPlayer, fix.TeamRed),
		onSide(anotherDraftedPlayer(t, fix, fix.TeamRed), fix.TeamRed),
		onSide(fix.BluePlayer, fix.TeamBlue),
		onSide(anotherDraftedPlayer(t, fix, fix.TeamBlue), fix.TeamBlue),
	))

	if err != nil {
		t.Fatalf("want two a side allowed under fourball, got %v", err)
	}
}

// The write is a replacement, so the lineup that comes back is the one sent and not the one
// merged with what was there.
func TestSettingALineupReplacesTheOneBefore(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	ctx := context.Background()
	replacement := anotherDraftedPlayer(t, fix, fix.TeamRed)

	if err := client.SetLineup(ctx, fix.MatchID, theLineup(
		onSide(replacement, fix.TeamRed),
		onSide(fix.BluePlayer, fix.TeamBlue),
	)); err != nil {
		t.Fatalf("set lineup: %v", err)
	}

	participants, err := client.ListParticipants(ctx, fix.MatchID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("want two participants, got %d", len(participants))
	}
	for _, p := range participants {
		if p.PlayerID == fix.RedPlayer {
			t.Error("the player who was replaced is still in the match")
		}
	}
}
