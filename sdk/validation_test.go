package sdk

import (
	"context"
	"testing"
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

func TestScoreSubmission_Validate(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		req     ScoreSubmission
		wantErr bool
	}{
		{"valid", ScoreSubmission{HoleNumber: 1, Strokes: 4, TeamID: 1}, false},
		{"hole too low", ScoreSubmission{HoleNumber: 0, Strokes: 4, TeamID: 1}, true},
		{"hole too high", ScoreSubmission{HoleNumber: 19, Strokes: 4, TeamID: 1}, true},
		{"non-positive strokes", ScoreSubmission{HoleNumber: 1, Strokes: 0, TeamID: 1}, true},
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
