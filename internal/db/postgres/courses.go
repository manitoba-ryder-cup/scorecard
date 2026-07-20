package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres/internal/sqlc"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type CoursesDB struct {
	db *DB
}

func NewCoursesDB(db *DB) *CoursesDB {
	return &CoursesDB{db: db}
}

func (c *CoursesDB) CreateCourse(ctx context.Context, in golf.CreateCourseInput) (*golf.Course, error) {
	return withTenant(ctx, c.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Course, error) {
		course, err := q.CreateCourse(ctx, sqlc.CreateCourseParams{TenantID: tenantID, Name: in.Name})
		if err != nil {
			return nil, fmt.Errorf("creating course: %w", mapWriteErr(err))
		}
		cs := toDomainCourse(course)
		return &cs, nil
	})
}

func (c *CoursesDB) GetCourse(ctx context.Context, id uuid.UUID) (*golf.Course, error) {
	return withTenant(ctx, c.db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Course, error) {
		course, err := q.GetCourse(ctx, sqlc.GetCourseParams{ID: id, TenantID: tenantID})
		if err != nil {
			return nil, fmt.Errorf("getting course %s: %w", id, mapReadErr(err))
		}
		cs := toDomainCourse(course)
		return &cs, nil
	})
}

func (c *CoursesDB) ListCourses(ctx context.Context) ([]golf.Course, error) {
	return withTenant(ctx, c.db, func(q *sqlc.Queries, tenantID uuid.UUID) ([]golf.Course, error) {
		courses, err := q.ListCourses(ctx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("listing courses: %w", err)
		}
		return mapSlice(courses, toDomainCourse), nil
	})
}

func toDomainCourse(c sqlc.Course) golf.Course {
	return golf.Course{ID: c.ID, Name: c.Name}
}
