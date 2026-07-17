package rest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type CourseService interface {
	CreateTeeColor(ctx context.Context, in golf.CreateTeeColorInput) (*golf.TeeColor, error)
	ListTeeColors(ctx context.Context) ([]golf.TeeColor, error)
	CreateCourse(ctx context.Context, in golf.CreateCourseInput) (*golf.Course, error)
	GetCourse(ctx context.Context, id uuid.UUID) (*golf.Course, error)
	ListCourses(ctx context.Context) ([]golf.Course, error)
	CreateTeeSet(ctx context.Context, in golf.CreateTeeSetInput) (*golf.TeeSetWithHoles, error)
}

type CoursesHandler struct {
	courseService CourseService
}

func NewCoursesHandler(courseService CourseService) *CoursesHandler {
	return &CoursesHandler{courseService: courseService}
}

// GET /v1/tee-colors
func (h *CoursesHandler) ListTeeColors(w http.ResponseWriter, r *http.Request) {
	teeColors, err := h.courseService.ListTeeColors(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list tee colors", err)
		return
	}
	respondJSON(w, http.StatusOK, toTeeColorDTOs(teeColors))
}

// POST /v1/tee-colors
func (h *CoursesHandler) CreateTeeColor(w http.ResponseWriter, r *http.Request) {
	var req sdk.CreateTeeColorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	teeColor, err := h.courseService.CreateTeeColor(r.Context(), golf.CreateTeeColorInput{Color: req.Color})
	if err != nil {
		respondDomainError(w, "Failed to create tee color", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeeColorDTO(*teeColor))
}

// GET /v1/courses
func (h *CoursesHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	courses, err := h.courseService.ListCourses(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list courses", err)
		return
	}
	respondJSON(w, http.StatusOK, toCourseDTOs(courses))
}

// GET /v1/courses/{id}
func (h *CoursesHandler) GetCourse(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid course ID", err)
		return
	}
	course, err := h.courseService.GetCourse(r.Context(), id)
	if err != nil {
		respondDomainError(w, "Failed to get course", err)
		return
	}
	respondJSON(w, http.StatusOK, toCourseDTO(*course))
}

// POST /v1/courses/{id}/tees
func (h *CoursesHandler) AddTeeSet(w http.ResponseWriter, r *http.Request) {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid course ID", err)
		return
	}
	var req sdk.CreateTeeSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	holes := make([]golf.HoleInput, len(req.Holes))
	for i, h := range req.Holes {
		holes[i] = golf.HoleInput{Number: h.Number, Par: h.Par, Hdcp: h.Hdcp, Yards: h.Yards}
	}
	teeSet, err := h.courseService.CreateTeeSet(r.Context(), golf.CreateTeeSetInput{
		CourseID:   courseID,
		TeeColorID: req.TeeColorID,
		Slope:      req.Slope,
		Rating:     req.Rating,
		Holes:      holes,
	})
	if err != nil {
		respondDomainError(w, "Failed to add tee set", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeeSetDTO(*teeSet))
}

// POST /v1/courses
func (h *CoursesHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	var req sdk.CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	course, err := h.courseService.CreateCourse(r.Context(), golf.CreateCourseInput{Name: req.Name})
	if err != nil {
		respondDomainError(w, "Failed to create course", err)
		return
	}
	respondJSON(w, http.StatusCreated, toCourseDTO(*course))
}
