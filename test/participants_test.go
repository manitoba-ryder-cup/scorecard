package test

import (
	"context"
	"errors"
	"net/http"
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
		Name: "End To End Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
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
		CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID,
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	// Participants: each drafted player joins the match on their team.
	if _, err := client.AddParticipant(ctx, match.ID, sdk.AddParticipantRequest{PlayerID: redPlayer, TeamID: redTeam}); err != nil {
		t.Fatalf("add red participant: %v", err)
	}
	if _, err := client.AddParticipant(ctx, match.ID, sdk.AddParticipantRequest{PlayerID: bluePlayer, TeamID: blueTeam}); err != nil {
		t.Fatalf("add blue participant: %v", err)
	}
	participants, err := client.ListParticipants(ctx, match.ID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("want 2 participants, got %d", len(participants))
	}

	// Score hole 1: Red 4, Blue 5. Red leads — the whole chain works end to end.
	if err := client.SubmitScore(ctx, match.ID, sdk.ScoreSubmission{HoleNumber: 1, Strokes: 4, TeamID: redTeam, PlayerID: &redPlayer}); err != nil {
		t.Fatalf("submit red score: %v", err)
	}
	if err := client.SubmitScore(ctx, match.ID, sdk.ScoreSubmission{HoleNumber: 1, Strokes: 5, TeamID: blueTeam, PlayerID: &bluePlayer}); err != nil {
		t.Fatalf("submit blue score: %v", err)
	}
	holes, err := client.GetMatchScores(ctx, match.ID)
	if err != nil {
		t.Fatalf("get scores: %v", err)
	}
	if len(holes) != 1 || holes[0].LeaderTeamID == nil || *holes[0].LeaderTeamID != redTeam || holes[0].Lead != 1 {
		t.Fatalf("want hole 1 led by Red, lead 1, got %+v", holes)
	}
}

// draftedMatch sets up a tournament with one drafted Red player and a match, returning
// what an AddParticipant call needs.
func draftedMatch(t *testing.T, client *sdk.Client) (matchID, redTeam, redPlayer uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Participant Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	redTeam = teamByColor(t, client, tour.ID, sdk.TeamColorRed)
	p, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "Red", LastName: "Player"})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	if _, err := client.EnterTournamentPlayer(ctx, tour.ID, sdk.EnterTournamentPlayerRequest{PlayerID: p.ID}); err != nil {
		t.Fatalf("enter: %v", err)
	}
	if _, err := client.DraftPlayer(ctx, redTeam, sdk.DraftPlayerRequest{PlayerID: p.ID}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	courseID, teeColorID, formatID := playableCourse(t, client)
	match, err := client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	return match.ID, redTeam, p.ID
}

func TestAddUndraftedPlayerRejected(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	matchID, redTeam, _ := draftedMatch(t, client)

	// A brand-new player, not drafted onto redTeam -> team_members FK violation -> 400.
	other, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "Undrafted", LastName: "Player"})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	_, err = client.AddParticipant(ctx, matchID, sdk.AddParticipantRequest{PlayerID: other.ID, TeamID: redTeam})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}

func TestAddDuplicateParticipantConflicts(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	matchID, redTeam, redPlayer := draftedMatch(t, client)

	if _, err := client.AddParticipant(ctx, matchID, sdk.AddParticipantRequest{PlayerID: redPlayer, TeamID: redTeam}); err != nil {
		t.Fatalf("first add: %v", err)
	}
	_, err := client.AddParticipant(ctx, matchID, sdk.AddParticipantRequest{PlayerID: redPlayer, TeamID: redTeam})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 APIError, got %v", err)
	}
}

func TestAddParticipantToNonexistentMatchReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	_, err := client.AddParticipant(context.Background(), uuid.New(), sdk.AddParticipantRequest{
		PlayerID: uuid.New(), TeamID: uuid.New(),
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

func TestRemoveParticipant(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	matchID, redTeam, redPlayer := draftedMatch(t, client)
	if _, err := client.AddParticipant(ctx, matchID, sdk.AddParticipantRequest{PlayerID: redPlayer, TeamID: redTeam}); err != nil {
		t.Fatalf("add participant: %v", err)
	}

	if err := client.RemoveParticipant(ctx, matchID, redPlayer); err != nil {
		t.Fatalf("remove participant: %v", err)
	}

	parts, err := client.ListParticipants(ctx, matchID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(parts) != 0 {
		t.Fatalf("want 0 participants after removal, got %d", len(parts))
	}
	// Removing a match assignment leaves the player drafted on their team.
	members, err := client.ListTeamMembers(ctx, redTeam)
	if err != nil {
		t.Fatalf("list team members: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("want player still drafted, got %d members", len(members))
	}
}

func TestRemoveParticipantNotInMatchReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	matchID, _, redPlayer := draftedMatch(t, client)

	// Drafted but never added to the match -> not a participant -> 404.
	err := client.RemoveParticipant(ctx, matchID, redPlayer)
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

func TestUndraftRemovesMatchParticipant(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	matchID, redTeam, redPlayer := draftedMatch(t, client)
	if _, err := client.AddParticipant(ctx, matchID, sdk.AddParticipantRequest{PlayerID: redPlayer, TeamID: redTeam}); err != nil {
		t.Fatalf("add participant: %v", err)
	}

	// Undrafting a player cascades (ON DELETE CASCADE): they're pulled from the match too.
	if err := client.UndraftPlayer(ctx, redTeam, redPlayer); err != nil {
		t.Fatalf("undraft: %v", err)
	}

	parts, err := client.ListParticipants(ctx, matchID)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(parts) != 0 {
		t.Fatalf("want participant removed via cascade, got %d", len(parts))
	}
}
