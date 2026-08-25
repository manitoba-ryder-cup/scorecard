package test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// TestFullTournamentFlowToScoring builds an entire tournament through the public API
// — tournament, players, entry, draft, course, match, participants — and submits a
// score, proving the whole setup chain is reachable end to end.
func TestFullTournamentFlowToScoring(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "End To End Cup", StartDate: cupStart, EndDate: cupEnd, Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	redTeam := teamByColor(t, client, tour.ID, sdk.TeamColorRed)
	blueTeam := teamByColor(t, client, tour.ID, sdk.TeamColorBlue)

	// Roster: create, enter, and draft one player per side.
	draft := func(name string, team uuid.UUID) uuid.UUID {
		p, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: name, LastName: "Player"})
		if err != nil {
			t.Fatalf("create player: %v", err)
		}
		if _, err := client.EnterTournamentPlayer(ctx, tour.ID, sdk.EnterTournamentPlayerRequest{PlayerID: p.ID}); err != nil {
			t.Fatalf("enter player: %v", err)
		}
		if _, err := client.DraftPlayer(ctx, team, sdk.DraftPlayerRequest{PlayerID: p.ID}); err != nil {
			t.Fatalf("draft player: %v", err)
		}
		return p.ID
	}
	redPlayer := draft("Red", redTeam)
	bluePlayer := draft("Blue", blueTeam)

	// Course + match.
	courseID, teeColorID, formatID := playableCourse(t, client)
	match, err := client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{
		CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID, TeeTime: fixtureTeeTime(),
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	if err := client.SetLineup(ctx, match.ID, theLineup(
		onSide(redPlayer, redTeam), onSide(bluePlayer, blueTeam),
	)); err != nil {
		t.Fatalf("set lineup: %v", err)
	}
	participants, err := client.ListParticipants(ctx, match.ID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("want 2 participants, got %d", len(participants))
	}

	// Score hole 1: Red 4, Blue 5. Red leads — the whole chain works end to end.
	if _, err := client.SubmitScore(ctx, match.ID, sdk.ScoreSubmission{
		HoleNumber: 1,
		Scores: []sdk.ScoreEntry{
			{TeamID: redTeam, PlayerID: &redPlayer, Strokes: 4},
			{TeamID: blueTeam, PlayerID: &bluePlayer, Strokes: 5},
		},
	}); err != nil {
		t.Fatalf("submit hole 1: %v", err)
	}
	holes, err := client.GetMatchScores(ctx, match.ID)
	if err != nil {
		t.Fatalf("get scores: %v", err)
	}
	if len(holes) != 1 || holes[0].LeaderTeamID == nil || *holes[0].LeaderTeamID != redTeam || holes[0].Lead != 1 {
		t.Fatalf("want hole 1 led by Red, lead 1, got %+v", holes)
	}
}

// draftedMatch sets up a tournament with a drafted player on each side and a match, which
// is the smallest thing a lineup can be set on.
func draftedMatch(t *testing.T, client *sdk.Client) (matchID, redTeam, redPlayer, blueTeam, bluePlayer uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Participant Cup", StartDate: cupStart, EndDate: cupEnd, Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	redTeam = teamByColor(t, client, tour.ID, sdk.TeamColorRed)
	blueTeam = teamByColor(t, client, tour.ID, sdk.TeamColorBlue)
	onto := func(name string, team uuid.UUID) uuid.UUID {
		p, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: name, LastName: "Player"})
		if err != nil {
			t.Fatalf("create player: %v", err)
		}
		if _, err := client.EnterTournamentPlayer(ctx, tour.ID, sdk.EnterTournamentPlayerRequest{PlayerID: p.ID}); err != nil {
			t.Fatalf("enter: %v", err)
		}
		if _, err := client.DraftPlayer(ctx, team, sdk.DraftPlayerRequest{PlayerID: p.ID}); err != nil {
			t.Fatalf("draft: %v", err)
		}
		return p.ID
	}
	redPlayer = onto("Red", redTeam)
	bluePlayer = onto("Blue", blueTeam)
	courseID, teeColorID, formatID := playableCourse(t, client)
	match, err := client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID, TeeTime: fixtureTeeTime()})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	return match.ID, redTeam, redPlayer, blueTeam, bluePlayer
}

// The composite FK to team_members is what rejects a player who is not on the side they are
// named for, and mapWriteErr reads that as the caller's error rather than ours.
func TestALineupNamingAnUndraftedPlayerIsRejected(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	matchID, redTeam, _, blueTeam, bluePlayer := draftedMatch(t, client)

	other, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "Undrafted", LastName: "Player"})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}

	err = client.SetLineup(ctx, matchID, theLineup(onSide(other.ID, redTeam), onSide(bluePlayer, blueTeam)))

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}

// A player named twice would pass the size rule by filling a side on their own, so the shape
// check refuses it before the wire rather than leaving the primary key to answer.
func TestALineupNamingAPlayerTwiceIsRejected(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	matchID, redTeam, redPlayer, _, _ := draftedMatch(t, client)

	err := client.SetLineup(context.Background(), matchID,
		theLineup(onSide(redPlayer, redTeam), onSide(redPlayer, redTeam)))

	if err == nil || !strings.Contains(err.Error(), "already in the lineup") {
		t.Fatalf("want the duplicate named, got %v", err)
	}
}

func TestSettingALineupOnAnUnknownMatchIs404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	err := client.SetLineup(context.Background(), uuid.New(),
		theLineup(onSide(uuid.New(), uuid.New()), onSide(uuid.New(), uuid.New())))

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

// A lineup is who plays this match, not who is on the team. Replacing it leaves the draft be.
func TestSettingALineupLeavesTheDraftAlone(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	matchID, redTeam, redPlayer, blueTeam, bluePlayer := draftedMatch(t, client)
	if err := client.SetLineup(ctx, matchID, theLineup(onSide(redPlayer, redTeam), onSide(bluePlayer, blueTeam))); err != nil {
		t.Fatalf("set lineup: %v", err)
	}

	members, err := client.ListTeamMembers(ctx, redTeam)
	if err != nil {
		t.Fatalf("list team members: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("want the player still drafted, got %d members", len(members))
	}
}

func TestUndraftRemovesMatchParticipant(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	matchID, redTeam, redPlayer, blueTeam, bluePlayer := draftedMatch(t, client)
	if err := client.SetLineup(ctx, matchID, theLineup(onSide(redPlayer, redTeam), onSide(bluePlayer, blueTeam))); err != nil {
		t.Fatalf("set lineup: %v", err)
	}

	// Undrafting a player cascades (ON DELETE CASCADE): they're pulled from the match too.
	if err := client.UndraftPlayer(ctx, redTeam, redPlayer); err != nil {
		t.Fatalf("undraft: %v", err)
	}

	parts, err := client.ListParticipants(ctx, matchID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("want the undrafted player pulled by cascade, got %d participants", len(parts))
	}
}
