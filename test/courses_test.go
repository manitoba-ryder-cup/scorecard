package test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

func TestCreateTeeColorAndCourse(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	tc, err := client.CreateTeeColor(ctx, sdk.CreateTeeColorRequest{Color: "White"})
	if err != nil {
		t.Fatalf("create tee color: %v", err)
	}
	if tc.ID == uuid.Nil || tc.Color != "White" {
		t.Fatalf("unexpected tee color: %+v", tc)
	}

	course, err := client.CreateCourse(ctx, sdk.CreateCourseRequest{Name: "Pine Ridge"})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	if course.ID == uuid.Nil || course.Name != "Pine Ridge" {
		t.Fatalf("unexpected course: %+v", course)
	}

	// Round-trips through reads.
	got, err := client.GetCourse(ctx, course.ID)
	if err != nil {
		t.Fatalf("get course: %v", err)
	}
	if got.ID != course.ID || got.Name != "Pine Ridge" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	colors, err := client.ListTeeColors(ctx)
	if err != nil {
		t.Fatalf("list tee colors: %v", err)
	}
	if len(colors) != 1 || colors[0].Color != "White" {
		t.Fatalf("want [White], got %+v", colors)
	}

	courses, err := client.ListCourses(ctx)
	if err != nil {
		t.Fatalf("list courses: %v", err)
	}
	if len(courses) != 1 || courses[0].Name != "Pine Ridge" {
		t.Fatalf("want [Pine Ridge], got %+v", courses)
	}
}

func TestCreateCourseDuplicateNameConflicts(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	if _, err := client.CreateCourse(ctx, sdk.CreateCourseRequest{Name: "Dup Course"}); err != nil {
		t.Fatalf("create first: %v", err)
	}
	_, err := client.CreateCourse(ctx, sdk.CreateCourseRequest{Name: "Dup Course"})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 APIError, got %v", err)
	}
}

func TestCreateTeeColorDuplicateConflicts(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	if _, err := client.CreateTeeColor(ctx, sdk.CreateTeeColorRequest{Color: "Blue"}); err != nil {
		t.Fatalf("create first: %v", err)
	}
	_, err := client.CreateTeeColor(ctx, sdk.CreateTeeColorRequest{Color: "Blue"})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 APIError, got %v", err)
	}
}

func TestGetNonexistentCourseReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	_, err := client.GetCourse(context.Background(), uuid.New())
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}
