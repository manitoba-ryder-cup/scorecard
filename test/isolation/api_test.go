package isolation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// TestTenantIsolation_Tournaments checks a tournament and its teams stay within their tenant.
func TestTenantIsolation_Tournaments(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tenantA, tenantB := tenantClient(t, uuid.New()), tenantClient(t, uuid.New())

	tourA, err := tenantA.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Tenant A Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}

	t.Run("tenant B cannot list tenant A tournaments", func(t *testing.T) {
		tours, err := tenantB.ListTournaments(ctx)
		if err != nil {
			t.Fatalf("list tournaments: %v", err)
		}
		for _, tour := range tours {
			if tour.ID == tourA.ID {
				t.Fatalf("tenant B sees tenant A tournament %s", tourA.ID)
			}
		}
	})

	t.Run("tenant B cannot get tenant A tournament by ID", func(t *testing.T) {
		_, err := tenantB.GetTournament(ctx, tourA.ID)
		requireNotFound(t, err, "get tournament")
	})

	t.Run("tenant B sees no teams for tenant A tournament", func(t *testing.T) {
		// The tournament seeds two teams, so a non-empty result means they leaked.
		teams, err := tenantB.GetTournamentTeams(ctx, tourA.ID)
		if err != nil {
			t.Fatalf("get teams: %v", err)
		}
		if len(teams) != 0 {
			t.Fatalf("tenant B sees %d team(s) of tenant A's tournament", len(teams))
		}
	})

	t.Run("tenant B cannot create a match in tenant A tournament", func(t *testing.T) {
		_, err := tenantB.CreateMatch(ctx, tourA.ID, sdk.CreateMatchRequest{
			CourseID: uuid.New(), TeeColorID: uuid.New(), MatchFormatID: uuid.New(), TeeTime: time.Now().UTC().Format(time.RFC3339),
		})
		requireRejected(t, err, "create match")
	})
}

// TestTenantIsolation_Players checks players and their profiles stay within their tenant.
func TestTenantIsolation_Players(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tenantA, tenantB := tenantClient(t, uuid.New()), tenantClient(t, uuid.New())

	playerA, err := tenantA.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "Tenant", LastName: "AOnly"})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}

	t.Run("tenant B cannot list tenant A players", func(t *testing.T) {
		players, err := tenantB.ListPlayers(ctx)
		if err != nil {
			t.Fatalf("list players: %v", err)
		}
		for _, p := range players {
			if p.ID == playerA.ID {
				t.Fatalf("tenant B sees tenant A player %s", playerA.ID)
			}
		}
	})

	t.Run("tenant B cannot get tenant A player by ID", func(t *testing.T) {
		_, err := tenantB.GetPlayer(ctx, playerA.ID)
		requireNotFound(t, err, "get player")
	})

	t.Run("tenant B sees no tournament history for tenant A player", func(t *testing.T) {
		history, err := tenantB.GetPlayerTournaments(ctx, playerA.ID)
		if err != nil {
			t.Fatalf("get player tournaments: %v", err)
		}
		if len(history) != 0 {
			t.Fatalf("tenant B sees %d history entries for tenant A player", len(history))
		}
	})
}

// TestTenantIsolation_Courses checks course reference data stays within its tenant.
func TestTenantIsolation_Courses(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tenantA, tenantB := tenantClient(t, uuid.New()), tenantClient(t, uuid.New())

	courseA, err := tenantA.CreateCourse(ctx, sdk.CreateCourseRequest{Name: "Tenant A Golf Club"})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	teeColorA, err := tenantA.CreateTeeColor(ctx, sdk.CreateTeeColorRequest{Color: "TenantAWhite"})
	if err != nil {
		t.Fatalf("create tee color: %v", err)
	}

	t.Run("tenant B cannot list tenant A courses", func(t *testing.T) {
		courses, err := tenantB.ListCourses(ctx)
		if err != nil {
			t.Fatalf("list courses: %v", err)
		}
		for _, c := range courses {
			if c.ID == courseA.ID {
				t.Fatalf("tenant B sees tenant A course %s", courseA.ID)
			}
		}
	})

	t.Run("tenant B cannot get tenant A course by ID", func(t *testing.T) {
		_, err := tenantB.GetCourse(ctx, courseA.ID)
		requireNotFound(t, err, "get course")
	})

	t.Run("tenant B cannot list tenant A tee colors", func(t *testing.T) {
		colors, err := tenantB.ListTeeColors(ctx)
		if err != nil {
			t.Fatalf("list tee colors: %v", err)
		}
		for _, c := range colors {
			if c.ID == teeColorA.ID {
				t.Fatalf("tenant B sees tenant A tee color %s", teeColorA.ID)
			}
		}
	})
}

// TestTenantIsolation_MatchesAndScores checks a seeded match's lineup, scoring, and
// results are invisible and unwritable to another tenant.
func TestTenantIsolation_MatchesAndScores(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fixA := seedFullFixture(t, ctx, connectSuperuser(t, ctx))
	tenantB := tenantClient(t, uuid.New())

	t.Run("tenant B sees no participants in tenant A match", func(t *testing.T) {
		participants, err := tenantB.ListParticipants(ctx, fixA.MatchID)
		if err != nil {
			t.Fatalf("list participants: %v", err)
		}
		if len(participants) != 0 {
			t.Fatalf("tenant B sees %d participant(s) in tenant A's match", len(participants))
		}
	})

	t.Run("tenant B sees no scores for tenant A match", func(t *testing.T) {
		scores, err := tenantB.GetMatchScores(ctx, fixA.MatchID)
		if err != nil {
			t.Fatalf("get match scores: %v", err)
		}
		if len(scores) != 0 {
			t.Fatalf("tenant B sees %d scored hole(s) of tenant A's match", len(scores))
		}
	})

	t.Run("tenant B sees no results for tenant A tournament", func(t *testing.T) {
		results, err := tenantB.GetTournamentResults(ctx, fixA.TournamentID)
		if err != nil {
			t.Fatalf("get results: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("tenant B sees %d result(s) of tenant A's tournament", len(results))
		}
	})

	t.Run("tenant B cannot get tenant A match holes", func(t *testing.T) {
		_, err := tenantB.GetMatchHoles(ctx, fixA.MatchID)
		requireNotFound(t, err, "get match holes")
	})

	t.Run("tenant B cannot submit a score to tenant A match", func(t *testing.T) {
		_, err := tenantB.SubmitScore(ctx, fixA.MatchID, sdk.ScoreSubmission{
			HoleNumber: 1,
			Scores:     []sdk.ScoreEntry{{TeamID: fixA.TeamRed, PlayerID: &fixA.RedPlayer, Strokes: 4}},
		})
		requireRejected(t, err, "submit score")
	})

	t.Run("tenant B cannot add a participant to tenant A match", func(t *testing.T) {
		_, err := tenantB.AddParticipant(ctx, fixA.MatchID, sdk.AddParticipantRequest{
			PlayerID: fixA.RedPlayer, TeamID: fixA.TeamRed,
		})
		requireRejected(t, err, "add participant")
	})

	t.Run("tenant B cannot remove a participant from tenant A match", func(t *testing.T) {
		err := tenantB.RemoveParticipant(ctx, fixA.MatchID, fixA.RedPlayer)
		requireRejected(t, err, "remove participant")

		// The participant must still be there for tenant A.
		tenantA := tenantClient(t, fixA.TenantID)
		participants, err := tenantA.ListParticipants(ctx, fixA.MatchID)
		if err != nil {
			t.Fatalf("list participants as tenant A: %v", err)
		}
		if len(participants) != 2 {
			t.Fatalf("tenant A's match has %d participant(s) after tenant B's delete; want 2", len(participants))
		}
	})

	t.Run("tenant B cannot move the tee time of tenant A match", func(t *testing.T) {
		_, err := tenantB.UpdateMatchTeeTime(ctx, fixA.MatchID, sdk.UpdateTeeTimeRequest{TeeTime: "2026-08-01T08:20"})
		requireRejected(t, err, "update tee time")
	})
}

// TestTenantIsolation_Roster checks the draft and captaincy of another tenant's team
// cannot be read or altered.
func TestTenantIsolation_Roster(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fixA := seedFullFixture(t, ctx, connectSuperuser(t, ctx))
	tenantB := tenantClient(t, uuid.New())

	t.Run("tenant B sees no members of tenant A team", func(t *testing.T) {
		members, err := tenantB.ListTeamMembers(ctx, fixA.TeamRed)
		if err != nil {
			t.Fatalf("list team members: %v", err)
		}
		if len(members) != 0 {
			t.Fatalf("tenant B sees %d member(s) of tenant A's team", len(members))
		}
	})

	t.Run("tenant B sees no roster for tenant A tournament", func(t *testing.T) {
		roster, err := tenantB.ListTournamentPlayers(ctx, fixA.TournamentID)
		if err != nil {
			t.Fatalf("list tournament players: %v", err)
		}
		if len(roster) != 0 {
			t.Fatalf("tenant B sees %d roster entr(ies) of tenant A's tournament", len(roster))
		}
	})

	t.Run("tenant B cannot draft onto tenant A team", func(t *testing.T) {
		_, err := tenantB.DraftPlayer(ctx, fixA.TeamRed, sdk.DraftPlayerRequest{PlayerID: fixA.BluePlayer})
		requireRejected(t, err, "draft player")
	})

	t.Run("tenant B cannot set the captain of tenant A team", func(t *testing.T) {
		err := tenantB.SetTeamCaptain(ctx, fixA.TeamRed, sdk.SetTeamCaptainRequest{CaptainID: fixA.BluePlayer})
		requireRejected(t, err, "set captain")
	})

	t.Run("tenant B cannot clear the captain of tenant A team", func(t *testing.T) {
		err := tenantB.ClearTeamCaptain(ctx, fixA.TeamRed)
		requireRejected(t, err, "clear captain")

		// Tenant A's captain must be untouched.
		tenantA := tenantClient(t, fixA.TenantID)
		teams, err := tenantA.GetTournamentTeams(ctx, fixA.TournamentID)
		if err != nil {
			t.Fatalf("get teams as tenant A: %v", err)
		}
		for _, team := range teams {
			if team.ID == fixA.TeamRed && team.Captain == nil {
				t.Fatal("tenant B cleared tenant A's captain")
			}
		}
	})

	t.Run("tenant B cannot undraft from tenant A team", func(t *testing.T) {
		err := tenantB.UndraftPlayer(ctx, fixA.TeamRed, fixA.RedPlayer)
		requireRejected(t, err, "undraft player")
	})
}
