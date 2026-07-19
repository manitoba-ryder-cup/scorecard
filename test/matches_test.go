package test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// playableCourse sets up a course with a White tee (slope/rating + 18 holes) and
// returns (courseID, teeColorID, singlesFormatID) — enough to create a match.
func playableCourse(t *testing.T, client *sdk.Client) (courseID, teeColorID, formatID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	tc, err := client.CreateTeeColor(ctx, sdk.CreateTeeColorRequest{Color: "White"})
	if err != nil {
		t.Fatalf("create tee color: %v", err)
	}
	course, err := client.CreateCourse(ctx, sdk.CreateCourseRequest{Name: "Pine Ridge"})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	if _, err := client.AddTeeSet(ctx, course.ID, sdk.CreateTeeSetRequest{
		TeeColorID: tc.ID, Slope: 113, Rating: 71.2, Holes: eighteenHoles(),
	}); err != nil {
		t.Fatalf("add tee set: %v", err)
	}
	formats, err := client.ListMatchFormats(ctx)
	if err != nil {
		t.Fatalf("list formats: %v", err)
	}
	for _, f := range formats {
		if f.Name == "Singles" {
			formatID = f.ID
		}
	}
	if formatID == uuid.Nil {
		t.Fatal("Singles format not seeded")
	}
	return course.ID, tc.ID, formatID
}

func TestCreateAndListMatch(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Match Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	courseID, teeColorID, formatID := playableCourse(t, client)

	teeTime := "2026-08-01T08:00:00Z"
	match, err := client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{
		CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID, TeeTime: &teeTime,
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	if match.ID == uuid.Nil || match.TournamentID != tour.ID || match.CourseID != courseID || match.Handicapped {
		t.Fatalf("unexpected match: %+v", match)
	}
	if match.TeeTime == nil {
		t.Fatal("want a scheduled tee time")
	}
	gotT, _ := time.Parse(time.RFC3339, *match.TeeTime)
	wantT, _ := time.Parse(time.RFC3339, teeTime)
	if !gotT.Equal(wantT) {
		t.Fatalf("tee_time round-trip: want %s, got %s", teeTime, *match.TeeTime)
	}

	matches, err := client.ListMatches(ctx, tour.ID)
	if err != nil {
		t.Fatalf("list matches: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != match.ID {
		t.Fatalf("want the created match in the list, got %+v", matches)
	}
}

func TestCreateMatchUnknownFormatRejected(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Bad Format Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Brandon",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	courseID, teeColorID, _ := playableCourse(t, client)

	// match_format_id doesn't exist -> FK violation -> 400.
	_, err = client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{
		CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: uuid.New(),
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}

func TestCreateMatchWithoutTeeSetRejected(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()
	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "No Tee Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Selkirk",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	// Course and tee color exist, but no tee set links them.
	tc, err := client.CreateTeeColor(ctx, sdk.CreateTeeColorRequest{Color: "Blue"})
	if err != nil {
		t.Fatalf("create tee color: %v", err)
	}
	course, err := client.CreateCourse(ctx, sdk.CreateCourseRequest{Name: "Unconfigured GC"})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	formats, _ := client.ListMatchFormats(ctx)

	// No tee_set for (course, tee color) -> FK violation -> 400.
	_, err = client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{
		CourseID: course.ID, TeeColorID: tc.ID, MatchFormatID: formats[0].ID,
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}
