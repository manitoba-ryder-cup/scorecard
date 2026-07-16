package golf

import (
	"context"
	"fmt"
)

// CourseService owns course setup reference data: tee colors (tenant-level markers),
// courses (venues), and — later — their tee sets and holes. Request-shape validation
// happens at the API boundary; these methods persist and surface repository errors.
type CourseService struct {
	TeeColorDB teeColorDB
	CourseDB   courseDB
	TeeSetDB   teeSetDB
	Logger     logger
}

// CreateTeeColorInput is the intent to add a tenant-level tee color.
type CreateTeeColorInput struct {
	Color string
}

// CreateCourseInput is the intent to add a course (venue).
type CreateCourseInput struct {
	Name string
}

// HoleInput is one hole's setup within a tee-set creation.
type HoleInput struct {
	Number int32
	Par    int32
	Hdcp   int32
	Yards  int32
}

// CreateTeeSetInput is the intent to add a tee set (and its holes) to a course. The
// 18-holes/uniqueness shape is validated at the API boundary; the database enforces
// it too.
type CreateTeeSetInput struct {
	CourseID   int32
	TeeColorID int32
	Slope      int32
	Rating     float64
	Holes      []HoleInput
}

// TeeSetWithHoles is a tee set plus its holes — a course's playable configuration for
// one tee color.
type TeeSetWithHoles struct {
	TeeSet TeeSet
	Holes  []Hole
}

func (s *CourseService) CreateTeeColor(ctx context.Context, in CreateTeeColorInput) (*TeeColor, error) {
	teeColor, err := s.TeeColorDB.CreateTeeColor(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to create tee color: %w", err)
	}
	return teeColor, nil
}

func (s *CourseService) ListTeeColors(ctx context.Context) ([]TeeColor, error) {
	teeColors, err := s.TeeColorDB.ListTeeColors(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tee colors: %w", err)
	}
	return teeColors, nil
}

func (s *CourseService) CreateCourse(ctx context.Context, in CreateCourseInput) (*Course, error) {
	course, err := s.CourseDB.CreateCourse(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to create course: %w", err)
	}
	return course, nil
}

func (s *CourseService) GetCourse(ctx context.Context, id int32) (*Course, error) {
	course, err := s.CourseDB.GetCourse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}
	return course, nil
}

// CreateTeeSet adds a tee set and its holes to a course. It first confirms the course
// exists (so a bad course in the path is a clean 404), then persists the tee set and
// all its holes atomically.
func (s *CourseService) CreateTeeSet(ctx context.Context, in CreateTeeSetInput) (*TeeSetWithHoles, error) {
	if _, err := s.CourseDB.GetCourse(ctx, in.CourseID); err != nil {
		return nil, fmt.Errorf("failed to load course: %w", err)
	}
	teeSet, err := s.TeeSetDB.CreateTeeSet(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to create tee set: %w", err)
	}
	return teeSet, nil
}

func (s *CourseService) ListCourses(ctx context.Context) ([]Course, error) {
	courses, err := s.CourseDB.ListCourses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list courses: %w", err)
	}
	return courses, nil
}
