package test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

// eighteenHoles returns a well-formed set of 18 holes (numbers and stroke indexes 1-18).
func eighteenHoles() []sdk.Hole {
	holes := make([]sdk.Hole, 18)
	for i := int32(0); i < 18; i++ {
		holes[i] = sdk.Hole{Number: i + 1, Par: 4, Hdcp: i + 1, Yards: 400}
	}
	return holes
}

func TestAddTeeSetWithHoles(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()

	tc, err := client.CreateTeeColor(ctx, sdk.CreateTeeColorRequest{Color: "White"})
	if err != nil {
		t.Fatalf("create tee color: %v", err)
	}
	course, err := client.CreateCourse(ctx, sdk.CreateCourseRequest{Name: "Pine Ridge"})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}

	teeSet, err := client.AddTeeSet(ctx, course.ID, sdk.CreateTeeSetRequest{
		TeeColorID: tc.ID, Slope: 113, Rating: 71.2, Holes: eighteenHoles(),
	})
	if err != nil {
		t.Fatalf("add tee set: %v", err)
	}
	if teeSet.CourseID != course.ID || teeSet.TeeColorID != tc.ID || teeSet.Slope != 113 {
		t.Fatalf("unexpected tee set: %+v", teeSet)
	}
	// Confirms numeric(4,1) round-trips through the float64 sqlc override.
	if teeSet.Rating < 71.15 || teeSet.Rating > 71.25 {
		t.Fatalf("rating want ~71.2, got %v", teeSet.Rating)
	}
	if len(teeSet.Holes) != 18 {
		t.Fatalf("want 18 holes, got %d", len(teeSet.Holes))
	}
	if teeSet.Holes[0].Number != 1 || teeSet.Holes[0].Par != 4 || teeSet.Holes[0].Yards != 400 {
		t.Fatalf("unexpected first hole: %+v", teeSet.Holes[0])
	}
}

func TestAddTeeSetUnknownTeeColorRejected(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()

	course, err := client.CreateCourse(ctx, sdk.CreateCourseRequest{Name: "Bad Tee Course"})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}

	// tee_color_id uuid.New() doesn't exist -> FK violation -> 400.
	_, err = client.AddTeeSet(ctx, course.ID, sdk.CreateTeeSetRequest{
		TeeColorID: uuid.New(), Slope: 113, Rating: 71.2, Holes: eighteenHoles(),
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 APIError, got %v", err)
	}
}

func TestAddTeeSetToNonexistentCourseReturns404(t *testing.T) {
	client := freshClient(t)

	_, err := client.AddTeeSet(context.Background(), uuid.New(), sdk.CreateTeeSetRequest{
		TeeColorID: uuid.New(), Slope: 113, Rating: 71.2, Holes: eighteenHoles(),
	})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

// Sent raw (bypassing the SDK client's validation) to confirm the server also
// rejects a wrong hole count.
func TestAddTeeSetWrongHoleCountRejectedByServer(t *testing.T) {
	// Valid UUID path + tee_color_id so the request reaches hole-count validation.
	body := `{"tee_color_id":"11111111-1111-1111-1111-111111111111","slope":113,"rating":71.2,"holes":[]}`
	status, _ := request.Raw(t, http.MethodPost, "/v1/courses/11111111-1111-1111-1111-111111111111/tees", body, freshToken(t))
	if status != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", status)
	}
}
