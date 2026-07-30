package test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

// playableCourse sets up a course with a White tee (slope/rating + 18 holes) and
// returns (courseID, teeColorID, singlesFormatID) — enough to create a match.
// fixtureTeeTime puts a match under way right now: scores are only accepted from 2h
// before the tee time to 12h after, and most fixtures below go on to record some.
func fixtureTeeTime() string { return time.Now().UTC().Format(time.RFC3339) }

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
	t.Parallel()
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
		CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID, TeeTime: teeTime,
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	if match.ID == uuid.Nil || match.TournamentID != tour.ID || match.CourseID != courseID || match.Handicapped {
		t.Fatalf("unexpected match: %+v", match)
	}
	gotT, _ := time.Parse(time.RFC3339, match.TeeTime)
	wantT, _ := time.Parse(time.RFC3339, teeTime)
	if !gotT.Equal(wantT) {
		t.Fatalf("tee_time round-trip: want %s, got %s", teeTime, match.TeeTime)
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
	t.Parallel()
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
		CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: uuid.New(), TeeTime: fixtureTeeTime(),
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}

func TestCreateMatchWithoutTeeSetRejected(t *testing.T) {
	t.Parallel()
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
		CourseID: course.ID, TeeColorID: tc.ID, MatchFormatID: formats[0].ID, TeeTime: fixtureTeeTime(),
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}

func TestUpdateMatchTeeTime(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Tee Time Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	// playableCourse creates the course without a zone, so it takes the cup's default.
	courseID, teeColorID, formatID := playableCourse(t, client)
	match, err := client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{
		CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID, TeeTime: fixtureTeeTime(),
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	// The wall clock a tee sheet says, at a Winnipeg course in August (CDT, UTC-5).
	updated, err := client.UpdateMatchTeeTime(ctx, match.ID, sdk.UpdateTeeTimeRequest{TeeTime: "2026-08-01T08:20"})
	if err != nil {
		t.Fatalf("update tee time: %v", err)
	}
	if updated.TeeTime != "2026-08-01T13:20:00Z" {
		t.Errorf("tee_time = %q, want 2026-08-01T13:20:00Z", updated.TeeTime)
	}

	results, err := client.GetTournamentResults(ctx, tour.ID)
	if err != nil {
		t.Fatalf("get results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].TeeTime != "2026-08-01T13:20:00Z" {
		t.Errorf("results tee_time = %q, want the new instant", results[0].TeeTime)
	}
	if results[0].TeeTimeLocal != "2026-08-01T08:20" {
		t.Errorf("results tee_time_local = %q, want the course's wall clock", results[0].TeeTimeLocal)
	}
}

func TestUpdateMatchTeeTime_UnknownMatchIs404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	_, err := client.UpdateMatchTeeTime(context.Background(), uuid.New(), sdk.UpdateTeeTimeRequest{
		TeeTime: "2026-08-01T08:20",
	})

	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("err = %v, want a 404 APIError", err)
	}
}

// Sent raw so the SDK's own Validate cannot be what rejects it: a non-SDK caller posting a
// malformed tee time must still be turned away by the server.
func TestUpdateMatchTeeTime_ServerRejectsAMalformedTeeTime(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Bad Tee Time Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	courseID, teeColorID, formatID := playableCourse(t, client)
	match, err := client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{
		CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID, TeeTime: fixtureTeeTime(),
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	path := "/v1/matches/" + match.ID.String() + "/tee-time"
	for _, body := range []string{`{"tee_time":"half eight"}`, `{"tee_time":""}`, `{}`} {
		status, respBody := request.Raw(t, http.MethodPut, path, body, freshToken(t))
		if status != http.StatusBadRequest {
			t.Errorf("body %s: status = %d (%s), want 400", body, status, respBody)
		}
	}
}
