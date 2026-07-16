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

func (s *CourseService) ListCourses(ctx context.Context) ([]Course, error) {
	courses, err := s.CourseDB.ListCourses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list courses: %w", err)
	}
	return courses, nil
}
