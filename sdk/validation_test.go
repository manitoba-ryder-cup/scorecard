package sdk

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func strptr(s string) *string { return &s }

func TestCreatePlayerRequest_Validate(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		req     CreatePlayerRequest
		wantErr bool
	}{
		{"valid with email", CreatePlayerRequest{FirstName: "Dustin", LastName: "Johnson", Email: strptr("dj@example.com")}, false},
		{"valid roster-only", CreatePlayerRequest{FirstName: "Roster", LastName: "Only"}, false},
		{"empty first", CreatePlayerRequest{FirstName: " ", LastName: "Johnson"}, true},
		{"empty last", CreatePlayerRequest{FirstName: "Dustin", LastName: ""}, true},
		{"bad email", CreatePlayerRequest{FirstName: "Dustin", LastName: "Johnson", Email: strptr("not-an-email")}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate(ctx)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestCreateTeeColorRequest_Validate(t *testing.T) {
	if (CreateTeeColorRequest{Color: "White"}).Validate(context.Background()) != nil {
		t.Error("valid tee color should pass")
	}
	if (CreateTeeColorRequest{Color: " "}).Validate(context.Background()) == nil {
		t.Error("empty color should fail")
	}
}

func TestCreateCourseRequest_Validate(t *testing.T) {
	if (CreateCourseRequest{Name: "Pine Ridge"}).Validate(context.Background()) != nil {
		t.Error("valid course should pass")
	}
	if (CreateCourseRequest{Name: ""}).Validate(context.Background()) == nil {
		t.Error("empty name should fail")
	}
}

func TestCreateTournamentRequest_Validate(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		req     CreateTournamentRequest
		wantErr bool
	}{
		{"valid", CreateTournamentRequest{Name: "Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg"}, false},
		{"empty name", CreateTournamentRequest{Name: " ", StartDate: "2026-08-01", EndDate: "2026-08-03"}, true},
		{"missing start", CreateTournamentRequest{Name: "Cup", EndDate: "2026-08-03"}, true},
		{"unparseable date", CreateTournamentRequest{Name: "Cup", StartDate: "Aug 1", EndDate: "2026-08-03"}, true},
		{"end before start", CreateTournamentRequest{Name: "Cup", StartDate: "2026-08-03", EndDate: "2026-08-01"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate(ctx)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

// validHoles returns 18 well-formed holes (numbers and stroke indexes 1-18).
func validHoles() []Hole {
	holes := make([]Hole, 18)
	for i := int32(0); i < 18; i++ {
		holes[i] = Hole{Number: i + 1, Par: 4, Hdcp: i + 1, Yards: 400}
	}
	return holes
}

func TestCreateTeeSetRequest_Validate(t *testing.T) {
	ctx := context.Background()

	valid := CreateTeeSetRequest{TeeColorID: uuid.New(), Slope: 113, Rating: 71.2, Holes: validHoles()}
	if err := valid.Validate(ctx); err != nil {
		t.Fatalf("valid tee set should pass: %v", err)
	}

	// Bad slope.
	bad := valid
	bad.Slope = 200
	if bad.Validate(ctx) == nil {
		t.Error("slope out of range should fail")
	}

	// Wrong hole count.
	bad = valid
	bad.Holes = valid.Holes[:17]
	if bad.Validate(ctx) == nil {
		t.Error("17 holes should fail")
	}

	// Duplicate stroke index.
	bad = valid
	dupHoles := validHoles()
	dupHoles[1].Hdcp = dupHoles[0].Hdcp
	bad.Holes = dupHoles
	if bad.Validate(ctx) == nil {
		t.Error("duplicate hdcp should fail")
	}

	// Missing tee color.
	bad = valid
	bad.TeeColorID = uuid.Nil
	if bad.Validate(ctx) == nil {
		t.Error("missing tee_color_id should fail")
	}
}

var (
	teamA   = uuid.New()
	teamB   = uuid.New()
	playerA = uuid.New()
	playerB = uuid.New()
)

func hole(n int32, scores ...ScoreEntry) ScoreSubmission {
	return ScoreSubmission{HoleNumber: n, Scores: scores}
}
func score(team uuid.UUID, strokes int32) ScoreEntry {
	return ScoreEntry{TeamID: team, Strokes: strokes}
}
func playerScore(player, team uuid.UUID, strokes int32) ScoreEntry {
	return ScoreEntry{TeamID: team, PlayerID: &player, Strokes: strokes}
}

func TestScoreSubmission_Validate(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		req     ScoreSubmission
		wantErr bool
	}{
		{"valid", hole(1, score(teamA, 4), score(teamB, 5)), false},
		{"hole too low", hole(0, score(teamA, 4)), true},
		{"hole too high", hole(19, score(teamA, 4)), true},
		{"non-positive strokes", hole(1, score(teamA, 0)), true},
		{"non-positive strokes on a later entry", hole(1, score(teamA, 4), score(teamB, 0)), true},
		// A hole with no scores is a client bug, not an instruction to clear it.
		{"no scores", hole(1), true},
		// Two scores for one player would resolve to whichever the write applied last.
		{"the same player twice", hole(1, playerScore(playerA, teamA, 4), playerScore(playerA, teamA, 5)), true},
		{"both players on a side", hole(1, playerScore(playerA, teamA, 4), playerScore(playerB, teamA, 5)), false},
		{"the same team twice, one ball", hole(1, score(teamA, 4), score(teamA, 5)), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate(ctx)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

// Length caps mirror the schema's VARCHARs so an over-long value fails as a client
// error at the boundary, instead of reaching Postgres and tripping a truncation error.
func TestValidateRejectsOverlongFields(t *testing.T) {
	long := strings.Repeat("a", 300)

	tests := []struct {
		name string
		req  interface{ Validate(context.Context) error }
	}{
		{"player first name", CreatePlayerRequest{FirstName: long, LastName: "Smith"}},
		{"player last name", CreatePlayerRequest{FirstName: "Bob", LastName: long}},
		{"tee color", CreateTeeColorRequest{Color: long}},
		{"course name", CreateCourseRequest{Name: long}},
		{"tournament name", CreateTournamentRequest{Name: long, StartDate: "2026-08-01", EndDate: "2026-08-03"}},
		{"tournament location", CreateTournamentRequest{Name: "Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: long}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.req.Validate(context.Background()); err == nil {
				t.Error("want a validation error for an over-long value, got nil")
			}
		})
	}
}

func uuidptr(u uuid.UUID) *uuid.UUID { return &u }
func boolptr(b bool) *bool           { return &b }

func TestUpdateMatchRequest_Validate(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	cases := []struct {
		name    string
		req     UpdateMatchRequest
		wantErr bool
	}{
		{"tee time only", UpdateMatchRequest{TeeTime: strptr("2026-09-18T13:00:00Z")}, false},
		{"course only", UpdateMatchRequest{CourseID: uuidptr(id)}, false},
		// false is a real value, so unsetting handicapped is not an empty body.
		{"handicapped false only", UpdateMatchRequest{Handicapped: boolptr(false)}, false},
		{"every field", UpdateMatchRequest{
			CourseID: uuidptr(id), TeeColorID: uuidptr(id), MatchFormatID: uuidptr(id),
			TeeTime: strptr("2026-09-18T13:00:00Z"), Handicapped: boolptr(true),
		}, false},

		{"nothing set", UpdateMatchRequest{}, true},
		{"explicit nil course", UpdateMatchRequest{CourseID: uuidptr(uuid.Nil)}, true},
		{"explicit nil tee colour", UpdateMatchRequest{TeeColorID: uuidptr(uuid.Nil)}, true},
		{"explicit nil format", UpdateMatchRequest{MatchFormatID: uuidptr(uuid.Nil)}, true},
		{"blank tee time", UpdateMatchRequest{TeeTime: strptr("")}, true},
		{"wall clock tee time", UpdateMatchRequest{TeeTime: strptr("2026-09-18T13:00")}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate(ctx)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func f32ptr(f float32) *float32 { return &f }

func TestUpdateTournamentPlayerRequest_Validate(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		req     UpdateTournamentPlayerRequest
		wantErr bool
	}{
		{"biography only", UpdateTournamentPlayerRequest{Biography: strptr("Holed out from the car park.")}, false},
		{"tier only", UpdateTournamentPlayerRequest{Tier: strptr("gold")}, false},
		// Zero is a real handicap — read as absent it could never be set back to scratch.
		{"handicap of zero", UpdateTournamentPlayerRequest{Hdcp: f32ptr(0)}, false},
		// Clearing a biography is a legitimate edit; clearing a tier is not.
		{"biography cleared", UpdateTournamentPlayerRequest{Biography: strptr("")}, false},

		{"nothing set", UpdateTournamentPlayerRequest{}, true},
		{"tier blanked", UpdateTournamentPlayerRequest{Tier: strptr("")}, true},
		{"tier all spaces", UpdateTournamentPlayerRequest{Tier: strptr("   ")}, true},
		{"tier too long", UpdateTournamentPlayerRequest{Tier: strptr(strings.Repeat("g", 200))}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate(ctx)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestUpdatePlayerRequest_Validate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		req     UpdatePlayerRequest
		wantErr bool
	}{
		{"photo only", UpdatePlayerRequest{PhotoPath: strptr("/img/x.webp")}, false},
		{"name only", UpdatePlayerRequest{FirstName: strptr("Jon")}, false},
		// Clearing a photo is how one is taken down.
		{"photo cleared", UpdatePlayerRequest{PhotoPath: strptr("")}, false},
		{"email only", UpdatePlayerRequest{Email: strptr("new@example.com")}, false},
		{"nothing set", UpdatePlayerRequest{}, true},
		// Blanking one would orphan the player from every future seed.
		{"email blanked", UpdatePlayerRequest{Email: strptr("")}, true},
		{"email malformed", UpdatePlayerRequest{Email: strptr("not-an-address")}, true},
		// A name can be corrected but not removed.
		{"first name blanked", UpdatePlayerRequest{FirstName: strptr("")}, true},
		{"last name blanked", UpdatePlayerRequest{LastName: strptr("   ")}, true},
		{"name too long", UpdatePlayerRequest{LastName: strptr(strings.Repeat("g", 200))}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate(context.Background())
			if tc.wantErr && err == nil {
				t.Errorf("want an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("want no error, got %v", err)
			}
		})
	}
}
