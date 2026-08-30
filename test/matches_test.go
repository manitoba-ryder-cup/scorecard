package test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
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
	wantsStatus(t, err, http.StatusBadRequest)
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
	wantsStatus(t, err, http.StatusBadRequest)
}

func TestUpdateMatchLeavesUnmentionedFieldsAlone(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Update Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	courseID, teeColorID, formatID := playableCourse(t, client)

	match, err := client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{
		CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID,
		TeeTime: "2026-08-01T08:00:00Z", Handicapped: true,
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	moved := "2026-08-01T14:30:00Z"
	updated, err := client.UpdateMatch(ctx, match.ID, sdk.UpdateMatchRequest{TeeTime: &moved})
	if err != nil {
		t.Fatalf("update match: %v", err)
	}

	got, _ := time.Parse(time.RFC3339, updated.TeeTime)
	want, _ := time.Parse(time.RFC3339, moved)
	if !got.Equal(want) {
		t.Errorf("tee_time: want %s, got %s", moved, updated.TeeTime)
	}
	// A caller who mentioned only the tee time must not have blanked everything else.
	if updated.CourseID != courseID || updated.TeeColorID != teeColorID || updated.MatchFormatID != formatID {
		t.Errorf("unmentioned references changed: %+v", updated)
	}
	if !updated.Handicapped {
		t.Error("handicapped was true and unmentioned, so it should still be true")
	}
	if updated.TournamentID != tour.ID {
		t.Errorf("tournament changed: want %s, got %s", tour.ID, updated.TournamentID)
	}
}

// A false a caller actually sent is not the same as one it left out: with a *bool the
// first must be applied and the second must not. The case above covers the omission.
func TestUpdateMatchAppliesAnExplicitFalse(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()
	courseID, teeColorID, formatID := playableCourse(t, client)

	tour, err := client.CreateTournament(ctx, sdk.CreateTournamentRequest{
		Name: "Explicit False Cup", StartDate: "2026-08-01", EndDate: "2026-08-03", Location: "Winnipeg",
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	match, err := client.CreateMatch(ctx, tour.ID, sdk.CreateMatchRequest{
		CourseID: courseID, TeeColorID: teeColorID, MatchFormatID: formatID,
		TeeTime: "2026-08-01T08:00:00Z", Handicapped: true,
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	off := false
	updated, err := client.UpdateMatch(ctx, match.ID, sdk.UpdateMatchRequest{Handicapped: &off})
	if err != nil {
		t.Fatalf("update match: %v", err)
	}
	if updated.Handicapped {
		t.Error("handicapped was sent as false, so it should no longer be set")
	}
}

func TestUpdateMatchUnknownMatchIsNotFound(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	moved := "2026-08-01T14:30:00Z"

	_, err := client.UpdateMatch(context.Background(), uuid.New(), sdk.UpdateMatchRequest{TeeTime: &moved})
	if err == nil {
		t.Fatal("want an error updating a match that does not exist")
	}
	wantsStatus(t, err, http.StatusNotFound)
}
