package sdk

import (
	"context"
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

func TestScoreSubmission_Validate(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		req     ScoreSubmission
		wantErr bool
	}{
		{"valid", ScoreSubmission{HoleNumber: 1, Strokes: 4, TeamID: uuid.New()}, false},
		{"hole too low", ScoreSubmission{HoleNumber: 0, Strokes: 4, TeamID: uuid.New()}, true},
		{"hole too high", ScoreSubmission{HoleNumber: 19, Strokes: 4, TeamID: uuid.New()}, true},
		{"non-positive strokes", ScoreSubmission{HoleNumber: 1, Strokes: 0, TeamID: uuid.New()}, true},
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
