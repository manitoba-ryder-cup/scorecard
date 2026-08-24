package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// playersPerSide is the field size per team. 8 divides evenly into both singles
// (8 one-on-one matches) and pairs (4 two-on-two matches), so every player plays
// exactly once in every format.
const playersPerSide = 8

// outcome is the result a match is deliberately driven to. It is the test's oracle:
// scores are generated from it, and expected match/tournament winners are tallied
// from it independently of the scoring engine — so if the engine derives a different
// winner from those scores, the assertion fails.
type outcome int

const (
	redWin outcome = iota
	blueWin
	halved
)

// TestFullRyderCupCorrectness runs a whole Ryder-Cup-shaped tournament through the public API
// and checks the winners against a result known by construction.
//
// Sixteen players, four rounds, one in each format, every player in exactly one match per
// round. Each match is driven to a pre-chosen outcome and the API is asked what it thinks —
// per match and, once every match is final, for the Cup and the point tally.
func TestFullRyderCupCorrectness(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Manitoba Ryder Cup", StartDate: cupStart, EndDate: cupEnd, Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	redTeam := teamByColor(t, client, tour.ID, sdk.TeamColorRed)
	blueTeam := teamByColor(t, client, tour.ID, sdk.TeamColorBlue)

	// One playable course (White tee, 18 holes). We ignore the returned Singles format
	// id and look formats up by name below, since we need all four.
	courseID, teeColorID, _ := playableCourse(t, client)
	formats := formatsByName(t, client)

	// Build both benches: create each player, enter them in the tournament, draft them
	// onto their side. Indices are stable, so red[i] and blue[i] are paired opponents.
	red := make([]uuid.UUID, playersPerSide)
	blue := make([]uuid.UUID, playersPerSide)
	for i := 0; i < playersPerSide; i++ {
		red[i] = enterAndDraft(t, client, tour.ID, redTeam, "Red", fmt.Sprintf("Player%d", i))
		blue[i] = enterAndDraft(t, client, tour.ID, blueTeam, "Blue", fmt.Sprintf("Player%d", i))
	}

	// Each side names a captain (one of its drafted players).
	if err := client.SetTeamCaptain(ctx, redTeam, sdk.SetTeamCaptainRequest{CaptainID: red[0]}); err != nil {
		t.Fatalf("set red captain: %v", err)
	}
	if err := client.SetTeamCaptain(ctx, blueTeam, sdk.SetTeamCaptainRequest{CaptainID: blue[0]}); err != nil {
		t.Fatalf("set blue captain: %v", err)
	}

	// perSide is the match grain, and outcomes needs one entry per match, covering each side's
	// players exactly once. Engineered so Red takes the cup 13-7.
	rounds := []struct {
		format   string
		perSide  int
		outcomes []outcome
	}{
		{"Singles", 1, []outcome{redWin, redWin, redWin, redWin, redWin, blueWin, blueWin, halved}},
		{"Fourball", 2, []outcome{redWin, redWin, redWin, blueWin}},
		{"Alt Shot", 2, []outcome{redWin, redWin, blueWin, halved}},
		{"Scramble", 2, []outcome{redWin, redWin, blueWin, blueWin}},
	}

	// Pass 1: create every match and enter its participants, but score nothing yet.
	type matchPlan struct {
		id     uuid.UUID
		redPs  []uuid.UUID
		bluePs []uuid.UUID
		out    outcome
	}
	var plans []matchPlan
	for _, rd := range rounds {
		formatID, ok := formats[rd.format]
		if !ok {
			t.Fatalf("format %q not seeded", rd.format)
		}
		if want := playersPerSide / rd.perSide; len(rd.outcomes) != want {
			t.Fatalf("%s: %d outcomes but %d matches needed to use each player once", rd.format, len(rd.outcomes), want)
		}
		for m, out := range rd.outcomes {
			match, err := client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{
				CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID, TeeTime: fixtureTeeTime(),
			})
			if err != nil {
				t.Fatalf("create %s match %d: %v", rd.format, m, err)
			}
			base := m * rd.perSide
			redPs := red[base : base+rd.perSide]
			bluePs := blue[base : base+rd.perSide]
			for _, p := range redPs {
				if _, err := client.AddParticipant(ctx, match.ID, sdk.AddParticipantRequest{PlayerID: p, TeamID: redTeam}); err != nil {
					t.Fatalf("add red participant to %s match %d: %v", rd.format, m, err)
				}
			}
			for _, p := range bluePs {
				if _, err := client.AddParticipant(ctx, match.ID, sdk.AddParticipantRequest{PlayerID: p, TeamID: blueTeam}); err != nil {
					t.Fatalf("add blue participant to %s match %d: %v", rd.format, m, err)
				}
			}
			plans = append(plans, matchPlan{id: match.ID, redPs: redPs, bluePs: bluePs, out: out})
		}
	}

	// With matches created but unscored, the Cup is undecided.
	if status, err := client.GetTournamentStatus(ctx, tour.ID); err != nil {
		t.Fatalf("tournament status: %v", err)
	} else if status.Finished {
		t.Fatal("tournament should not be finished before any scores are entered")
	}
	if w, err := client.GetTournamentWinner(ctx, tour.ID); err != nil {
		t.Fatalf("tournament winner: %v", err)
	} else if w.Finished || w.WinnerTeamID != nil {
		t.Fatalf("no Cup winner before play, got %+v", w)
	}

	// Pass 2: play each match to its intended outcome and check the engine agrees.
	// Tally the expected point totals from the same outcomes, independently of the API.
	var redPts, bluePts float64
	for _, pl := range plans {
		playMatch(t, client, pl.id, redTeam, blueTeam, pl.redPs, pl.bluePs, pl.out)

		got, err := client.GetMatchStatus(ctx, pl.id)
		if err != nil {
			t.Fatalf("get match %s status: %v", pl.id, err)
		}
		if !got.Finished {
			t.Fatalf("match %s should be finished after scoring", pl.id)
		}
		var want *uuid.UUID
		switch pl.out {
		case redWin:
			redPts += 1
			want = &redTeam
		case blueWin:
			bluePts += 1
			want = &blueTeam
		case halved:
			redPts += 0.5
			bluePts += 0.5
		}
		if !uuidPtrEqual(got.WinnerTeamID, want) {
			t.Fatalf("match %s winner: want %v, got %v", pl.id, fmtUUIDPtr(want), fmtUUIDPtr(got.WinnerTeamID))
		}
	}

	// Guard the oracle itself: the schedule must be the 13-7 split we designed.
	if redPts != 13 || bluePts != 7 {
		t.Fatalf("schedule sanity: expected Red 13 / Blue 7, tallied Red %v / Blue %v", redPts, bluePts)
	}

	// Every match final: the Cup is decided, and Red has the majority.
	if status, err := client.GetTournamentStatus(ctx, tour.ID); err != nil {
		t.Fatalf("tournament status: %v", err)
	} else if !status.Finished {
		t.Fatal("tournament should be finished once every match is decided")
	}
	winner, err := client.GetTournamentWinner(ctx, tour.ID)
	if err != nil {
		t.Fatalf("tournament winner: %v", err)
	}
	if !winner.Finished || winner.WinnerTeamID == nil || *winner.WinnerTeamID != redTeam {
		t.Fatalf("Cup winner: want Red %s, got %+v", redTeam, winner)
	}

	// The materialized point tally must match our independent count.
	teams, err := client.GetTournamentTeams(ctx, tour.ID)
	if err != nil {
		t.Fatalf("tournament teams: %v", err)
	}
	for _, tm := range teams {
		switch tm.ID {
		case redTeam:
			if tm.Points != redPts {
				t.Fatalf("Red points: want %v, got %v", redPts, tm.Points)
			}
			if tm.Captain == nil || tm.Captain.ID != red[0] {
				t.Fatalf("Red captain: want %s, got %+v", red[0], tm.Captain)
			}
		case blueTeam:
			if tm.Points != bluePts {
				t.Fatalf("Blue points: want %v, got %v", bluePts, tm.Points)
			}
			if tm.Captain == nil || tm.Captain.ID != blue[0] {
				t.Fatalf("Blue captain: want %s, got %+v", blue[0], tm.Captain)
			}
		}
	}
}

// playMatch drives a match to an outcome. A decisive one has the winner shoot 3 to the
// loser's 5, closing out at hole 10; a halved one has both shoot 4 for all eighteen.
//
// One player posts per team each hole. A team is scored by its best ball, so a single score
// is the team's score, and this keeps submissions — the suite's dominant cost — to two per
// hole in every format.
func playMatch(t *testing.T, client *sdk.Client, matchID, redTeam, blueTeam uuid.UUID, redPs, bluePs []uuid.UUID, out outcome) {
	t.Helper()
	ctx := context.Background()
	post := func(hole int32, teamA, playerA uuid.UUID, strokesA int32, teamB, playerB uuid.UUID, strokesB int32) {
		pa, pb := playerA, playerB
		if _, err := client.SubmitScore(ctx, matchID, sdk.ScoreSubmission{
			HoleNumber: hole,
			Scores: []sdk.ScoreEntry{
				{TeamID: teamA, PlayerID: &pa, Strokes: strokesA},
				{TeamID: teamB, PlayerID: &pb, Strokes: strokesB},
			},
		}); err != nil {
			t.Fatalf("submit scores (match %s hole %d): %v", matchID, hole, err)
		}
	}

	if out == halved {
		for h := int32(1); h <= 18; h++ {
			post(h, redTeam, redPs[0], 4, blueTeam, bluePs[0], 4)
		}
		return
	}

	winTeam, winP, loseTeam, loseP := redTeam, redPs[0], blueTeam, bluePs[0]
	if out == blueWin {
		winTeam, winP, loseTeam, loseP = blueTeam, bluePs[0], redTeam, redPs[0]
	}
	// Holes 1-10 are enough: after 10 holes won the lead is 10 with 8 to play (10&8).
	for h := int32(1); h <= 10; h++ {
		post(h, winTeam, winP, 3, loseTeam, loseP, 5)
	}
}

// enterAndDraft creates a player, enters them in the tournament, and drafts them onto
// the given team — the full path a player takes before they can appear in a match.
func enterAndDraft(t *testing.T, client *sdk.Client, tournamentID, teamID uuid.UUID, first, last string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	p, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: first, LastName: last})
	if err != nil {
		t.Fatalf("create player %s %s: %v", first, last, err)
	}
	if _, err := client.EnterTournamentPlayer(ctx, tournamentID, sdk.EnterTournamentPlayerRequest{PlayerID: p.ID}); err != nil {
		t.Fatalf("enter player %s %s: %v", first, last, err)
	}
	if _, err := client.DraftPlayer(ctx, teamID, sdk.DraftPlayerRequest{PlayerID: p.ID}); err != nil {
		t.Fatalf("draft player %s %s: %v", first, last, err)
	}
	return p.ID
}

// formatsByName indexes the seeded match formats by name.
func formatsByName(t *testing.T, client *sdk.Client) map[string]uuid.UUID {
	t.Helper()
	fs, err := client.ListMatchFormats(context.Background())
	if err != nil {
		t.Fatalf("list formats: %v", err)
	}
	m := make(map[string]uuid.UUID, len(fs))
	for _, f := range fs {
		m[f.Name] = f.ID
	}
	return m
}

func uuidPtrEqual(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func fmtUUIDPtr(p *uuid.UUID) string {
	if p == nil {
		return "<nil>"
	}
	return p.String()
}

// cupWithCourse creates a tournament, its two teams, and one playable course, returning the
// ids needed to enter players and add Singles matches (course setup done once per cup).
func cupWithCourse(t *testing.T, client *sdk.Client, name string) (tourID, redTeam, blueTeam, courseID, teeColorID, singles uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: name, StartDate: cupStart, EndDate: cupEnd, Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	redTeam = teamByColor(t, client, tour.ID, sdk.TeamColorRed)
	blueTeam = teamByColor(t, client, tour.ID, sdk.TeamColorBlue)
	courseID, teeColorID, singles = playableCourse(t, client)
	return tour.ID, redTeam, blueTeam, courseID, teeColorID, singles
}

// addSinglesMatch enters+drafts one player per side (named with `tag`) and puts them in a
// new Singles match. Returns the match and the two players.
func addSinglesMatch(t *testing.T, client *sdk.Client, tourID, redTeam, blueTeam, courseID, teeColorID, singles uuid.UUID, tag string) (matchID, redP, blueP uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	redP = enterAndDraft(t, client, tourID, redTeam, "Red", tag)
	blueP = enterAndDraft(t, client, tourID, blueTeam, "Blue", tag)
	match, err := client.CreateMatch(ctx, tourID, sdk.CreateMatchRequest{CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: singles, TeeTime: fixtureTeeTime()})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	if _, err := client.AddParticipant(ctx, match.ID, sdk.AddParticipantRequest{PlayerID: redP, TeamID: redTeam}); err != nil {
		t.Fatalf("add red: %v", err)
	}
	if _, err := client.AddParticipant(ctx, match.ID, sdk.AddParticipantRequest{PlayerID: blueP, TeamID: blueTeam}); err != nil {
		t.Fatalf("add blue: %v", err)
	}
	return match.ID, redP, blueP
}

// A Cup that finishes level on points has no winner. There is no holder to retain it;
// a real tie would be replayed by the last group, which has never happened. A single
// halved match is the minimal tie (0.5 each).
func TestTiedCupIsFinishedWithNoWinner(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tourID, redTeam, blueTeam, courseID, teeColorID, singles := cupWithCourse(t, client, "Tied Cup")
	m, redP, blueP := addSinglesMatch(t, client, tourID, redTeam, blueTeam, courseID, teeColorID, singles, "One")

	playMatch(t, client, m, redTeam, blueTeam, []uuid.UUID{redP}, []uuid.UUID{blueP}, halved)

	if status, err := client.GetTournamentStatus(ctx, tourID); err != nil {
		t.Fatalf("status: %v", err)
	} else if !status.Finished {
		t.Fatal("a tournament with all matches done should be finished")
	}
	w, err := client.GetTournamentWinner(ctx, tourID)
	if err != nil {
		t.Fatalf("winner: %v", err)
	}
	if !w.Finished || w.WinnerTeamID != nil {
		t.Fatalf("a tied Cup should be finished with no winner, got %+v", w)
	}
	teams, err := client.GetTournamentTeams(ctx, tourID)
	if err != nil {
		t.Fatalf("teams: %v", err)
	}
	for _, tm := range teams {
		if tm.Points != 0.5 {
			t.Fatalf("want 0.5 points each on a halved match, got %v for %s", tm.Points, tm.Color)
		}
	}
}

// The Cup is only decided once every match is final — no early "clinch" — so it stays
// unfinished while any match is unscored, even if a side already leads.
func TestCupNotFinishedWhileAMatchRemains(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	tourID, redTeam, blueTeam, courseID, teeColorID, singles := cupWithCourse(t, client, "Unfinished Cup")
	m1, redP, blueP := addSinglesMatch(t, client, tourID, redTeam, blueTeam, courseID, teeColorID, singles, "One")
	addSinglesMatch(t, client, tourID, redTeam, blueTeam, courseID, teeColorID, singles, "Two") // second match, left unscored

	// Finish only the first match (Red win); the second stays unscored.
	playMatch(t, client, m1, redTeam, blueTeam, []uuid.UUID{redP}, []uuid.UUID{blueP}, redWin)

	if status, err := client.GetTournamentStatus(ctx, tourID); err != nil {
		t.Fatalf("status: %v", err)
	} else if status.Finished {
		t.Fatal("the Cup must not be finished while a match is unscored")
	}
	if w, err := client.GetTournamentWinner(ctx, tourID); err != nil {
		t.Fatalf("winner: %v", err)
	} else if w.Finished || w.WinnerTeamID != nil {
		t.Fatalf("no Cup winner while a match remains, got %+v", w)
	}
}
