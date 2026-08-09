package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/app"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
	"github.com/travisbale/knowhere/identity"
)

// newSeedServices connects to the test database and returns the domain services under a
// fresh tenant, so each test's data is isolated (no inter-test cleanup needed).
func newSeedServices(t *testing.T) (context.Context, *app.Services) {
	t.Helper()
	db, err := postgres.NewDB(context.Background(), util.LoadConfig().DatabaseURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(db.Close)
	ctx := identity.WithTenant(context.Background(), uuid.New())
	return ctx, app.NewServices(db)
}

// seedCourse creates the reference data a seeded tournament's matches point at (a course,
// a White tee, and a flat par-4 18), returning the names the seed input references.
func seedCourse(t *testing.T, ctx context.Context, svc *app.Services) (course, teeColor string) {
	t.Helper()
	tc, err := svc.Course.CreateTeeColor(ctx, golf.CreateTeeColorInput{Color: "White"})
	if err != nil {
		t.Fatalf("tee color: %v", err)
	}
	c, err := svc.Course.CreateCourse(ctx, golf.CreateCourseInput{Name: "Test Links"})
	if err != nil {
		t.Fatalf("course: %v", err)
	}
	holes := make([]golf.HoleInput, 18)
	for i := range holes {
		holes[i] = golf.HoleInput{Number: int32(i + 1), Par: 4, Hdcp: int32(i + 1), Yards: 400}
	}
	if _, err := svc.Course.CreateTeeSet(ctx, golf.CreateTeeSetInput{
		CourseID: c.ID, TeeColorID: tc.ID, Slope: 113, Rating: 72.0, Holes: holes,
	}); err != nil {
		t.Fatalf("tee set: %v", err)
	}
	return c.Name, tc.Color
}

func TestSeedTournamentEntersRosterSetsCaptainsAndSchedulesMatches(t *testing.T) {
	t.Parallel()
	ctx, svc := newSeedServices(t)
	course, teeColor := seedCourse(t, ctx, svc)

	in := &app.SeedInput{
		Tournament: app.SeedTournamentMeta{Name: "Seed Cup", StartDate: "2026-09-12", EndDate: "2026-09-13", Location: "Clear Lake"},
		Course:     course,
		TeeColor:   teeColor,
		Captains:   map[string]string{"Red": "rc@x.com", "Blue": "bc@x.com"},
		Players: []app.SeedPlayer{
			{FirstName: "Ryan", LastName: "Reddish", Email: "rc@x.com", Tier: "gold"},
			{FirstName: "Rex", LastName: "Redford", Email: "r2@x.com", Tier: "white"},
			{FirstName: "Bill", LastName: "Bluer", Email: "bc@x.com", Tier: "gold"},
			{FirstName: "Bob", LastName: "Bluto", Email: "b2@x.com", Tier: "blue"},
		},
		Matches: []app.SeedMatchGroup{
			{Format: "Singles", TeeTimes: []string{"2026-09-12T15:00:00Z", "2026-09-12T15:10:00Z"}},
		},
	}

	summary, err := app.SeedTournament(ctx, svc, in)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if summary.PlayersEntered != 4 || summary.Matches != 2 {
		t.Fatalf("summary = %+v, want 4 entered / 2 matches", summary)
	}

	// Each side's captain is the named player.
	wantCaptain := map[string]string{"Red": "Reddish", "Blue": "Bluer"}
	teams, err := svc.Team.ListTeamsByTournament(ctx, summary.TournamentID)
	if err != nil {
		t.Fatalf("list teams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("want 2 teams, got %d", len(teams))
	}
	for _, tm := range teams {
		if tm.Captain == nil {
			t.Fatalf("%s team has no captain", tm.Color)
		}
		if tm.Captain.LastName != wantCaptain[tm.Color] {
			t.Fatalf("%s captain = %s, want %s", tm.Color, tm.Captain.LastName, wantCaptain[tm.Color])
		}
	}

	// The whole roster is entered; only the two captains are drafted onto a team, the
	// rest await the live draft.
	roster, err := svc.Roster.ListPlayers(ctx, summary.TournamentID)
	if err != nil {
		t.Fatalf("list roster: %v", err)
	}
	if len(roster) != 4 {
		t.Fatalf("want 4 roster entries, got %d", len(roster))
	}
	drafted := map[string]bool{}
	for _, p := range roster {
		drafted[p.LastName] = p.TeamID != nil
	}
	if !drafted["Reddish"] || !drafted["Bluer"] {
		t.Fatalf("captains should be drafted onto their team: %+v", drafted)
	}
	if drafted["Redford"] || drafted["Bluto"] {
		t.Fatalf("non-captains should be undrafted (team assigned live): %+v", drafted)
	}

	// Two matches, neither with participants assigned.
	matches, err := svc.Match.ListMatches(ctx, summary.TournamentID)
	if err != nil {
		t.Fatalf("list matches: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("want 2 matches, got %d", len(matches))
	}
	for _, m := range matches {
		parts, err := svc.Match.ListParticipants(ctx, m.ID)
		if err != nil {
			t.Fatalf("list participants: %v", err)
		}
		if len(parts) != 0 {
			t.Fatalf("match %s should have no participants, got %d", m.ID, len(parts))
		}
	}
}

func TestSeedTournamentRejectsCaptainNotInRoster(t *testing.T) {
	t.Parallel()
	ctx, svc := newSeedServices(t)
	course, teeColor := seedCourse(t, ctx, svc)

	in := &app.SeedInput{
		Tournament: app.SeedTournamentMeta{Name: "Bad Captain Cup", StartDate: "2026-09-12", EndDate: "2026-09-13", Location: "Clear Lake"},
		Course:     course,
		TeeColor:   teeColor,
		Captains:   map[string]string{"Red": "nobody@x.com", "Blue": "bc@x.com"},
		Players: []app.SeedPlayer{
			{FirstName: "Bill", LastName: "Bluer", Email: "bc@x.com"},
		},
	}

	if _, err := app.SeedTournament(ctx, svc, in); err == nil {
		t.Fatal("want an error when a captain isn't in the roster, got nil")
	}

	// The error must arrive before anything is written. This used to fail only after the
	// tournament and its whole roster were committed, so a typo left an event behind and
	// a rerun made a second one.
	tournaments, err := svc.Tournament.ListTournaments(ctx)
	if err != nil {
		t.Fatalf("list tournaments: %v", err)
	}
	if len(tournaments) != 0 {
		t.Fatalf("a rejected seed should write nothing, found %d tournament(s)", len(tournaments))
	}
	players, err := svc.Player.ListPlayers(ctx)
	if err != nil {
		t.Fatalf("list players: %v", err)
	}
	if len(players) != 0 {
		t.Fatalf("a rejected seed should create no players, found %d", len(players))
	}
}

// A bad file is caught before the first write; this covers what planning cannot see —
// something failing partway through the writes themselves.
func TestSeedWritesRollBackTogether(t *testing.T) {
	t.Parallel()
	db, err := postgres.NewDB(context.Background(), util.LoadConfig().DatabaseURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(db.Close)
	ctx := identity.WithTenant(context.Background(), uuid.New())
	svc := app.NewServices(db)

	boom := errors.New("connection lost at match ten")
	err = postgres.WithinTenantTx(ctx, db, func(ctx context.Context) error {
		// CreateTournament spans two tables and opens its own withTenant closure, so this
		// also proves an inner transaction joins the outer one instead of committing early.
		if _, err := svc.Tournament.CreateTournament(ctx, golf.CreateTournamentInput{
			Name: "Doomed Cup", StartDate: time.Now(), EndDate: time.Now(), Location: "Clear Lake",
		}); err != nil {
			return err
		}
		email := "ghost@x.com"
		if _, err := svc.Player.CreatePlayer(ctx, golf.CreatePlayerInput{
			FirstName: "Ghost", LastName: "Player", Email: &email,
		}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("want the failure surfaced, got %v", err)
	}

	tournaments, err := svc.Tournament.ListTournaments(ctx)
	if err != nil {
		t.Fatalf("list tournaments: %v", err)
	}
	if len(tournaments) != 0 {
		t.Fatalf("want the tournament rolled back, found %d", len(tournaments))
	}
	players, err := svc.Player.ListPlayers(ctx)
	if err != nil {
		t.Fatalf("list players: %v", err)
	}
	if len(players) != 0 {
		t.Fatalf("want the player rolled back, found %d", len(players))
	}
}

func TestSeedTournamentReusesPlayersByEmail(t *testing.T) {
	t.Parallel()
	ctx, svc := newSeedServices(t)
	course, teeColor := seedCourse(t, ctx, svc)

	setup := func(name string) *app.SeedInput {
		return &app.SeedInput{
			Tournament: app.SeedTournamentMeta{Name: name, StartDate: "2026-09-12", EndDate: "2026-09-13", Location: "Clear Lake"},
			Course:     course,
			TeeColor:   teeColor,
			Captains:   map[string]string{"Red": "shared@x.com", "Blue": "bc@x.com"},
			Players: []app.SeedPlayer{
				{FirstName: "Sam", LastName: "Shared", Email: "shared@x.com"},
				{FirstName: "Bill", LastName: "Bluer", Email: "bc@x.com"},
			},
		}
	}

	if _, err := app.SeedTournament(ctx, svc, setup("Cup A")); err != nil {
		t.Fatalf("seed A: %v", err)
	}
	if _, err := app.SeedTournament(ctx, svc, setup("Cup B")); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	players, err := svc.Player.ListPlayers(ctx)
	if err != nil {
		t.Fatalf("list players: %v", err)
	}
	count := 0
	for _, p := range players {
		if p.Email != nil && *p.Email == "shared@x.com" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("the shared player should exist once, found %d", count)
	}
}
