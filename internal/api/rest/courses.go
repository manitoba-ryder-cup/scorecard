package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// GET /v1/tee-colors
func (rt *Router) ListTeeColors(w http.ResponseWriter, r *http.Request) {
	teeColors, err := rt.CourseService.ListTeeColors(r.Context())
	if err != nil {
		respondError(r.Context(), w, http.StatusInternalServerError, "Failed to list tee colors", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(teeColors, toTeeColorDTO))
}

// POST /v1/tee-colors
func (rt *Router) CreateTeeColor(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeAndValidate[sdk.CreateTeeColorRequest](w, r)
	if !ok {
		return
	}
	teeColor, err := rt.CourseService.CreateTeeColor(r.Context(), golf.CreateTeeColorInput{Color: req.Color})
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to create tee color", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeeColorDTO(*teeColor))
}

// GET /v1/courses
func (rt *Router) ListCourses(w http.ResponseWriter, r *http.Request) {
	courses, err := rt.CourseService.ListCourses(r.Context())
	if err != nil {
		respondError(r.Context(), w, http.StatusInternalServerError, "Failed to list courses", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(courses, toCourseDTO))
}

// GET /v1/courses/{id}
func (rt *Router) GetCourse(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "course")
	if !ok {
		return
	}
	course, err := rt.CourseService.GetCourse(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to get course", err)
		return
	}
	respondJSON(w, http.StatusOK, toCourseDTO(*course))
}

// POST /v1/courses/{id}/tees
func (rt *Router) AddTeeSet(w http.ResponseWriter, r *http.Request) {
	courseID, ok := pathUUIDOr400(w, r, "id", "course")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[sdk.CreateTeeSetRequest](w, r)
	if !ok {
		return
	}
	holes := make([]golf.HoleInput, len(req.Holes))
	for i, h := range req.Holes {
		holes[i] = golf.HoleInput{Number: h.Number, Par: h.Par, Hdcp: h.Hdcp, Yards: h.Yards}
	}
	teeSet, err := rt.CourseService.CreateTeeSet(r.Context(), golf.CreateTeeSetInput{
		CourseID:   courseID,
		TeeColorID: req.TeeColorID,
		Slope:      req.Slope,
		Rating:     req.Rating,
		Holes:      holes,
	})
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to add tee set", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeeSetDTO(*teeSet))
}

// GET /v1/courses/{id}/tees
// Lists a course's configured tee sets (with colour names) for match setup.
func (rt *Router) ListCourseTeeSets(w http.ResponseWriter, r *http.Request) {
	courseID, ok := pathUUIDOr400(w, r, "id", "course")
	if !ok {
		return
	}
	teeSets, err := rt.CourseService.ListCourseTeeSets(r.Context(), courseID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list course tee sets", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(teeSets, toCourseTeeSetDTO))
}

// POST /v1/courses
func (rt *Router) CreateCourse(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeAndValidate[sdk.CreateCourseRequest](w, r)
	if !ok {
		return
	}
	timeZone := req.TimeZone
	if timeZone == "" {
		timeZone = sdk.DefaultTimeZone
	}
	course, err := rt.CourseService.CreateCourse(r.Context(), golf.CreateCourseInput{Name: req.Name, TimeZone: timeZone})
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to create course", err)
		return
	}
	respondJSON(w, http.StatusCreated, toCourseDTO(*course))
}
