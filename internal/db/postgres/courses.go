package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
)

type CoursesDB struct {
	db *DB
}

func NewCoursesDB(db *DB) *CoursesDB {
	return &CoursesDB{db: db}
}

func (c *CoursesDB) CreateCourse(ctx context.Context, in golf.CreateCourseInput) (*golf.Course, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Course
	err = c.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		course, err := q.CreateCourse(ctx, sqlc.CreateCourseParams{TenantID: tenantID, Name: in.Name})
		if err != nil {
			return fmt.Errorf("creating course: %w", mapWriteErr(err))
		}
		cs := toDomainCourse(course)
		result = &cs
		return nil
	})
	return result, err
}

func (c *CoursesDB) GetCourse(ctx context.Context, id uuid.UUID) (*golf.Course, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *golf.Course
	err = c.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		course, err := q.GetCourse(ctx, sqlc.GetCourseParams{ID: id, TenantID: tenantID})
		if err != nil {
			return fmt.Errorf("getting course %d: %w", id, mapReadErr(err))
		}
		cs := toDomainCourse(course)
		result = &cs
		return nil
	})
	return result, err
}

func (c *CoursesDB) ListCourses(ctx context.Context) ([]golf.Course, error) {
	tenantID, err := identity.GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result []golf.Course
	err = c.db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
		courses, err := q.ListCourses(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("listing courses: %w", err)
		}
		result = make([]golf.Course, len(courses))
		for i, course := range courses {
			result[i] = toDomainCourse(course)
		}
		return nil
	})
	return result, err
}

func toDomainCourse(c sqlc.Course) golf.Course {
	return golf.Course{ID: c.ID, Name: c.Name}
}
